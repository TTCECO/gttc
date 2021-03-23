// Copyright 2018 The gttc Authors
// This file is part of the gttc library.
//
// The gttc library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gttc library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gttc library. If not, see <http://www.gnu.org/licenses/>.

// Package alien implements the delegated-proof-of-stake consensus engine.

package alien

import (
	"fmt"
	"github.com/TTCECO/gttc/params"
	"math/big"
	"strconv"
	"strings"

	"github.com/TTCECO/gttc/common"
	"github.com/TTCECO/gttc/consensus"
	"github.com/TTCECO/gttc/core/state"
	"github.com/TTCECO/gttc/core/types"
	"github.com/TTCECO/gttc/log"
	"github.com/TTCECO/gttc/rlp"
)

const (
	/*
	 *  ufo:version:category:action/data
	 */
	ufoPrefix             = "ufo"
	ufoVersion            = "1"
	ufoCategoryEvent      = "event"
	ufoCategoryLog        = "oplog"
	ufoCategorySC         = "sc"
	ufoEventVote          = "vote"
	ufoEventConfirm       = "confirm"
	ufoEventPorposal      = "proposal"
	ufoEventDeclare       = "declare"
	ufoEventSetCoinbase   = "setcb"
	ufoMinSplitLen        = 3
	posPrefix             = 0
	posVersion            = 1
	posCategory           = 2
	posEventVote          = 3
	posEventConfirm       = 3
	posEventProposal      = 3
	posEventDeclare       = 3
	posEventSetCoinbase   = 3
	posEventConfirmNumber = 4

	/*
	 *  proposal type
	 */
	proposalTypeCandidateAdd                  = 1
	proposalTypeCandidateRemove               = 2
	proposalTypeMinerRewardDistributionModify = 3 // count in one thousand
	proposalTypeSideChainAdd                  = 4
	proposalTypeSideChainRemove               = 5
	proposalTypeMinVoterBalanceModify         = 6
	proposalTypeProposalDepositModify         = 7
	proposalTypeRentSideChain                 = 8 // use TTC to buy coin on side chain

	/*
	 * proposal related
	 */
	maxValidationLoopCnt     = 50000                   // About one month if period = 3 & 21 super nodes
	minValidationLoopCnt     = 4                       // just for test, Note: 12350  About three days if seal each block per second & 21 super nodes
	defaultValidationLoopCnt = 10000                   // About one week if period = 3 & 21 super nodes
	maxProposalDeposit       = 100000                  // If no limit on max proposal deposit and 1 billion TTC deposit success passed, then no new proposal.
	minSCRentFee             = 100                     // 100 TTC
	minSCRentLength          = 850000                  // number of block about 1 month if period is 3
	defaultSCRentLength      = minSCRentLength * 3     // number of block about 3 month if period is 3
	maxSCRentLength          = defaultSCRentLength * 4 // number of block about 1 year if period is 3

	/*
	 * notice related
	 */
	noticeTypeGasCharging = 1
)

//side chain related
var minSCSetCoinbaseValue = big.NewInt(5e+18)

// RefundGas :
// refund gas to tx sender
type RefundGas map[common.Address]*big.Int

// RefundPair :
type RefundPair struct {
	Sender   common.Address
	GasPrice *big.Int
}

// RefundHash :
type RefundHash map[common.Hash]RefundPair

// Vote :
// vote come from custom tx which data like "ufo:1:event:vote"
// Sender of tx is Voter, the tx.to is Candidate
// Stake is the balance of Voter when create this vote
type Vote struct {
	Voter     common.Address
	Candidate common.Address
	Stake     *big.Int
}

// Confirmation :
// confirmation come  from custom tx which data like "ufo:1:event:confirm:123"
// 123 is the block number be confirmed
// Sender of tx is Signer only if the signer in the SignerQueue for block number 123
type Confirmation struct {
	Signer      common.Address
	BlockNumber *big.Int
}

// Proposal :
// proposal come from  custom tx which data like "ufo:1:event:proposal:candidate:add:address" or "ufo:1:event:proposal:percentage:60"
// proposal only come from the current candidates
// not only candidate add/remove , current signer can proposal for params modify like percentage of reward distribution ...
type Proposal struct {
	Hash                   common.Hash    // tx hash
	ReceivedNumber         *big.Int       // block number of proposal received
	CurrentDeposit         *big.Int       // received deposit for this proposal
	ValidationLoopCnt      uint64         // validation block number length of this proposal from the received block number
	ProposalType           uint64         // type of proposal 1 - add candidate 2 - remove candidate ...
	Proposer               common.Address // proposer
	TargetAddress          common.Address // candidate need to add/remove if candidateNeedPD == true
	MinerRewardPerThousand uint64         // reward of miner + side chain miner
	SCHash                 common.Hash    // side chain genesis parent hash need to add/remove
	SCBlockCountPerPeriod  uint64         // the number block sealed by this side chain per period, default 1
	SCBlockRewardPerPeriod uint64         // the reward of this side chain per period if SCBlockCountPerPeriod reach, default 0. SCBlockRewardPerPeriod/1000 * MinerRewardPerThousand/1000 * BlockReward is the reward for this side chain
	Declares               []*Declare     // Declare this proposal received (always empty in block header)
	MinVoterBalance        uint64         // value of minVoterBalance , need to mul big.Int(1e+18)
	ProposalDeposit        uint64         // The deposit need to be frozen during before the proposal get final conclusion. (TTC)
	SCRentFee              uint64         // number of TTC coin, not wei
	SCRentRate             uint64         // how many coin you want for 1 TTC on main chain
	SCRentLength           uint64         // minimize block number of main chain , the rent fee will be used as reward of side chain miner.
}

func (p *Proposal) copy() *Proposal {
	cpy := &Proposal{
		Hash:                   p.Hash,
		ReceivedNumber:         new(big.Int).Set(p.ReceivedNumber),
		CurrentDeposit:         new(big.Int).Set(p.CurrentDeposit),
		ValidationLoopCnt:      p.ValidationLoopCnt,
		ProposalType:           p.ProposalType,
		Proposer:               p.Proposer,
		TargetAddress:          p.TargetAddress,
		MinerRewardPerThousand: p.MinerRewardPerThousand,
		SCHash:                 p.SCHash,
		SCBlockCountPerPeriod:  p.SCBlockCountPerPeriod,
		SCBlockRewardPerPeriod: p.SCBlockRewardPerPeriod,
		Declares:               make([]*Declare, len(p.Declares)),
		MinVoterBalance:        p.MinVoterBalance,
		ProposalDeposit:        p.ProposalDeposit,
		SCRentFee:              p.SCRentFee,
		SCRentRate:             p.SCRentRate,
		SCRentLength:           p.SCRentLength,
	}

	copy(cpy.Declares, p.Declares)
	return cpy
}

// Declare :
// declare come from custom tx which data like "ufo:1:event:declare:hash:yes"
// proposal only come from the current candidates
// hash is the hash of proposal tx
type Declare struct {
	ProposalHash common.Hash
	Declarer     common.Address
	Decision     bool
}

// SCConfirmation is the confirmed tx send by side chain super node
type SCConfirmation struct {
	Hash     common.Hash
	Coinbase common.Address // the side chain signer , may be diff from signer in main chain
	Number   uint64
	LoopInfo []string
}

func (s *SCConfirmation) copy() *SCConfirmation {
	cpy := &SCConfirmation{
		Hash:     s.Hash,
		Coinbase: s.Coinbase,
		Number:   s.Number,
		LoopInfo: make([]string, len(s.LoopInfo)),
	}
	copy(cpy.LoopInfo, s.LoopInfo)
	return cpy
}

// SCSetCoinbase is the tx send by main chain super node which can set coinbase for side chain
type SCSetCoinbase struct {
	Hash     common.Hash
	Signer   common.Address
	Coinbase common.Address
}

type GasCharging struct {
	Target common.Address // target address on side chain
	Volume uint64         // volume of gas need charge (unit is ttc)
	Hash   common.Hash    // the hash of proposal, use as id of this proposal
}

// HeaderExtra is the struct of info in header.Extra[extraVanity:len(header.extra)-extraSeal]
// HeaderExtra is the current struct
type HeaderExtra struct {
	CurrentBlockConfirmations []Confirmation
	CurrentBlockVotes         []Vote
	CurrentBlockProposals     []Proposal
	CurrentBlockDeclares      []Declare
	ModifyPredecessorVotes    []Vote
	LoopStartTime             uint64
	SignerQueue               []common.Address
	SignerMissing             []common.Address
	ConfirmedBlockNumber      uint64
	SideChainConfirmations    []SCConfirmation
	SideChainSetCoinbases     []SCSetCoinbase
	SideChainNoticeConfirmed  []SCConfirmation
	SideChainCharging         []GasCharging //This only exist in side chain's header.Extra
}

// Encode HeaderExtra
func encodeHeaderExtra(config *params.AlienConfig, number *big.Int, val HeaderExtra) ([]byte, error) {

	var headerExtra interface{}
	switch {
	//case config.IsTrantor(number):

	default:
		headerExtra = val
	}
	return rlp.EncodeToBytes(headerExtra)

}

// Decode HeaderExtra
func decodeHeaderExtra(config *params.AlienConfig, number *big.Int, b []byte, val *HeaderExtra) error {
	var err error
	switch {
	//case config.IsTrantor(number):
	default:
		err = rlp.DecodeBytes(b, val)
	}
	return err
}

// Build side chain confirm data
func (a *Alien) buildSCEventConfirmData(scHash common.Hash, headerNumber *big.Int, headerTime *big.Int, lastLoopInfo string, chargingInfo string) []byte {
	return []byte(fmt.Sprintf("%s:%s:%s:%s:%s:%d:%d:%s:%s",
		ufoPrefix, ufoVersion, ufoCategorySC, ufoEventConfirm,
		scHash.Hex(), headerNumber.Uint64(), headerTime.Uint64(), lastLoopInfo, chargingInfo))

}

// Calculate Votes from transaction in this block, write into header.Extra
func (a *Alien) processCustomTx(headerExtra HeaderExtra, chain consensus.ChainReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, receipts []*types.Receipt) (HeaderExtra, RefundGas, error) {
	// if predecessor voter make transaction and vote in this block,
	// just process as vote, do it in snapshot.apply
	var (
		snap       *Snapshot
		err        error
		number     uint64
		refundGas  RefundGas
		refundHash RefundHash
	)
	refundGas = make(map[common.Address]*big.Int)
	refundHash = make(map[common.Hash]RefundPair)
	number = header.Number.Uint64()
	if number > 1 {
		snap, err = a.snapshot(chain, number-1, header.ParentHash, nil, nil, defaultLoopCntRecalculateSigners)
		if err != nil {
			return headerExtra, nil, err
		}
	}

	for _, tx := range txs {

		txSender, err := types.Sender(types.NewEIP155Signer(tx.ChainId()), tx)
		if err != nil {
			continue
		}

		if len(string(tx.Data())) >= len(ufoPrefix) {
			txData := string(tx.Data())
			txDataInfo := strings.Split(txData, ":")
			if len(txDataInfo) >= ufoMinSplitLen {
				if txDataInfo[posPrefix] == ufoPrefix {
					if txDataInfo[posVersion] == ufoVersion {
						// process vote event
						if txDataInfo[posCategory] == ufoCategoryEvent {
							if len(txDataInfo) > ufoMinSplitLen {
								// check is vote or not
								if txDataInfo[posEventVote] == ufoEventVote  && tx.To() != nil && (!candidateNeedPD || snap.isCandidate(*tx.To())) && state.GetBalance(txSender).Cmp(snap.MinVB) > 0 {
									headerExtra.CurrentBlockVotes = a.processEventVote(headerExtra.CurrentBlockVotes, state, tx, txSender)
								} else if txDataInfo[posEventConfirm] == ufoEventConfirm && snap.isCandidate(txSender) {
									headerExtra.CurrentBlockConfirmations, refundHash = a.processEventConfirm(headerExtra.CurrentBlockConfirmations, chain, txDataInfo, number, tx, txSender, refundHash)
								} else if txDataInfo[posEventProposal] == ufoEventPorposal {
									headerExtra.CurrentBlockProposals = a.processEventProposal(headerExtra.CurrentBlockProposals, txDataInfo, state, tx, txSender, snap)
								} else if txDataInfo[posEventDeclare] == ufoEventDeclare && snap.isCandidate(txSender) {
									headerExtra.CurrentBlockDeclares = a.processEventDeclare(headerExtra.CurrentBlockDeclares, txDataInfo, tx, txSender)
								}
							} else {
								// todo : something wrong, leave this transaction to process as normal transaction
							}
						} else if txDataInfo[posCategory] == ufoCategoryLog {
							// todo :
						} else if txDataInfo[posCategory] == ufoCategorySC {
							if len(txDataInfo) > ufoMinSplitLen {
								if txDataInfo[posEventConfirm] == ufoEventConfirm {
									if len(txDataInfo) > ufoMinSplitLen+5 {
										number := new(big.Int)
										if err := number.UnmarshalText([]byte(txDataInfo[ufoMinSplitLen+2])); err != nil {
											log.Trace("Side chain confirm info fail", "number", txDataInfo[ufoMinSplitLen+2])
											continue
										}
										if err := new(big.Int).UnmarshalText([]byte(txDataInfo[ufoMinSplitLen+3])); err != nil {
											log.Trace("Side chain confirm info fail", "time", txDataInfo[ufoMinSplitLen+3])
											continue
										}
										loopInfo := txDataInfo[ufoMinSplitLen+4]
										scHash := common.HexToHash(txDataInfo[ufoMinSplitLen+1])
										headerExtra.SideChainConfirmations, refundHash = a.processSCEventConfirm(headerExtra.SideChainConfirmations,
											scHash, number.Uint64(), loopInfo, tx, txSender, refundHash)

										chargingInfo := txDataInfo[ufoMinSplitLen+5]
										headerExtra.SideChainNoticeConfirmed = a.processSCEventNoticeConfirm(headerExtra.SideChainNoticeConfirmed,
											scHash, number.Uint64(), chargingInfo, txSender)

									}
								} else if txDataInfo[posEventSetCoinbase] == ufoEventSetCoinbase && snap.isCandidate(txSender) {
									if len(txDataInfo) > ufoMinSplitLen+1 {
										// the signer of main chain must send some value to coinbase of side chain for confirm tx of side chain
										if tx.Value().Cmp(minSCSetCoinbaseValue) >= 0  && tx.To() != nil {
											headerExtra.SideChainSetCoinbases = a.processSCEventSetCoinbase(headerExtra.SideChainSetCoinbases,
												common.HexToHash(txDataInfo[ufoMinSplitLen+1]), txSender, *tx.To())
										}
									}
								}
							}
						}
					}
				}
			}
		}
		// check each address
		if number > 1 {
			headerExtra.ModifyPredecessorVotes = a.processPredecessorVoter(headerExtra.ModifyPredecessorVotes, state, tx, txSender, snap)
		}

	}

	for _, receipt := range receipts {
		if pair, ok := refundHash[receipt.TxHash]; ok && receipt.Status == 1 {
			pair.GasPrice.Mul(pair.GasPrice, big.NewInt(int64(receipt.GasUsed)))
			refundGas = a.refundAddGas(refundGas, pair.Sender, pair.GasPrice)
		}
	}
	return headerExtra, refundGas, nil
}

func (a *Alien) refundAddGas(refundGas RefundGas, address common.Address, value *big.Int) RefundGas {
	if _, ok := refundGas[address]; ok {
		refundGas[address].Add(refundGas[address], value)
	} else {
		refundGas[address] = value
	}

	return refundGas
}

func (a *Alien) processSCEventNoticeConfirm(scEventNoticeConfirm []SCConfirmation, hash common.Hash, number uint64, chargingInfo string, txSender common.Address) []SCConfirmation {
	if chargingInfo != "" {
		scEventNoticeConfirm = append(scEventNoticeConfirm, SCConfirmation{
			Hash:     hash,
			Coinbase: txSender,
			Number:   number,
			LoopInfo: strings.Split(chargingInfo, "#"),
		})
	}
	return scEventNoticeConfirm
}

func (a *Alien) processSCEventConfirm(scEventConfirmaions []SCConfirmation, hash common.Hash, number uint64, loopInfo string, tx *types.Transaction, txSender common.Address, refundHash RefundHash) ([]SCConfirmation, RefundHash) {
	scEventConfirmaions = append(scEventConfirmaions, SCConfirmation{
		Hash:     hash,
		Coinbase: txSender,
		Number:   number,
		LoopInfo: strings.Split(loopInfo, "#"),
	})
	refundHash[tx.Hash()] = RefundPair{txSender, tx.GasPrice()}
	return scEventConfirmaions, refundHash
}

func (a *Alien) processSCEventSetCoinbase(scEventSetCoinbases []SCSetCoinbase, hash common.Hash, signer common.Address, coinbase common.Address) []SCSetCoinbase {
	scEventSetCoinbases = append(scEventSetCoinbases, SCSetCoinbase{
		Hash:     hash,
		Signer:   signer,
		Coinbase: coinbase,
	})
	return scEventSetCoinbases
}

func (a *Alien) processEventProposal(currentBlockProposals []Proposal, txDataInfo []string, state *state.StateDB, tx *types.Transaction, proposer common.Address, snap *Snapshot) []Proposal {
	// sample for add side chain proposal
	// eth.sendTransaction({from:eth.accounts[0],to:eth.accounts[0],value:0,data:web3.toHex("ufo:1:event:proposal:proposal_type:4:sccount:2:screward:50:schash:0x3210000000000000000000000000000000000000000000000000000000000000:vlcnt:4")})
	// sample for declare
	// eth.sendTransaction({from:eth.accounts[0],to:eth.accounts[0],value:0,data:web3.toHex("ufo:1:event:declare:hash:0x853e10706e6b9d39c5f4719018aa2417e8b852dec8ad18f9c592d526db64c725:decision:yes")})
	if len(txDataInfo) <= posEventProposal+2 {
		return currentBlockProposals
	}

	proposal := Proposal{
		Hash:                   tx.Hash(),
		ReceivedNumber:         big.NewInt(0),
		CurrentDeposit:         proposalDeposit, // for all type of deposit
		ValidationLoopCnt:      defaultValidationLoopCnt,
		ProposalType:           proposalTypeCandidateAdd,
		Proposer:               proposer,
		TargetAddress:          common.Address{},
		SCHash:                 common.Hash{},
		SCBlockCountPerPeriod:  1,
		SCBlockRewardPerPeriod: 0,
		MinerRewardPerThousand: minerRewardPerThousand,
		Declares:               []*Declare{},
		MinVoterBalance:        new(big.Int).Div(minVoterBalance, big.NewInt(1e+18)).Uint64(),
		ProposalDeposit:        new(big.Int).Div(proposalDeposit, big.NewInt(1e+18)).Uint64(), // default value
		SCRentFee:              0,
		SCRentRate:             1,
		SCRentLength:           defaultSCRentLength,
	}

	for i := 0; i < len(txDataInfo[posEventProposal+1:])/2; i++ {
		k, v := txDataInfo[posEventProposal+1+i*2], txDataInfo[posEventProposal+2+i*2]
		switch k {
		case "vlcnt":
			// If vlcnt is missing then user default value, but if the vlcnt is beyond the min/max value then ignore this proposal
			if validationLoopCnt, err := strconv.Atoi(v); err != nil || validationLoopCnt < minValidationLoopCnt || validationLoopCnt > maxValidationLoopCnt {
				return currentBlockProposals
			} else {
				proposal.ValidationLoopCnt = uint64(validationLoopCnt)
			}
		case "schash":
			proposal.SCHash.UnmarshalText([]byte(v))
		case "sccount":
			if scBlockCountPerPeriod, err := strconv.Atoi(v); err != nil {
				return currentBlockProposals
			} else {
				proposal.SCBlockCountPerPeriod = uint64(scBlockCountPerPeriod)
			}
		case "screward":
			if scBlockRewardPerPeriod, err := strconv.Atoi(v); err != nil {
				return currentBlockProposals
			} else {
				proposal.SCBlockRewardPerPeriod = uint64(scBlockRewardPerPeriod)
			}
		case "proposal_type":
			if proposalType, err := strconv.Atoi(v); err != nil {
				return currentBlockProposals
			} else {
				proposal.ProposalType = uint64(proposalType)
			}
		case "candidate":
			// not check here
			proposal.TargetAddress.UnmarshalText([]byte(v))
		case "mrpt":
			// miner reward per thousand
			if mrpt, err := strconv.Atoi(v); err != nil || mrpt <= 0 || mrpt > 1000 {
				return currentBlockProposals
			} else {
				proposal.MinerRewardPerThousand = uint64(mrpt)
			}
		case "mvb":
			// minVoterBalance
			if mvb, err := strconv.Atoi(v); err != nil || mvb <= 0 {
				return currentBlockProposals
			} else {
				proposal.MinVoterBalance = uint64(mvb)
			}
		case "mpd":
			// proposalDeposit
			if mpd, err := strconv.Atoi(v); err != nil || mpd <= 0 || mpd > maxProposalDeposit {
				return currentBlockProposals
			} else {
				proposal.ProposalDeposit = uint64(mpd)
			}
		case "scrt":
			// target address on side chain to charge gas
			proposal.TargetAddress.UnmarshalText([]byte(v))
		case "scrf":
			// side chain rent fee
			if scrf, err := strconv.Atoi(v); err != nil || scrf < minSCRentFee {
				return currentBlockProposals
			} else {
				proposal.SCRentFee = uint64(scrf)
			}
		case "scrr":
			// side chain rent rate
			if scrr, err := strconv.Atoi(v); err != nil || scrr <= 0 {
				return currentBlockProposals
			} else {
				proposal.SCRentRate = uint64(scrr)
			}
		case "scrl":
			// side chain rent length
			if scrl, err := strconv.Atoi(v); err != nil || scrl < minSCRentLength || scrl > maxSCRentLength {
				return currentBlockProposals
			} else {
				proposal.SCRentLength = uint64(scrl)
			}
		}
	}
	// now the proposal is built
	currentProposalPay := new(big.Int).Set(proposalDeposit)
	if proposal.ProposalType == proposalTypeRentSideChain {
		// check if the proposal target side chain exist
		if !snap.isSideChainExist(proposal.SCHash) {
			return currentBlockProposals
		}
		if (proposal.TargetAddress == common.Address{}) {
			return currentBlockProposals
		}
		currentProposalPay.Add(currentProposalPay, new(big.Int).Mul(new(big.Int).SetUint64(proposal.SCRentFee), big.NewInt(1e+18)))
	}
	// check enough balance for deposit
	if state.GetBalance(proposer).Cmp(currentProposalPay) < 0 {
		return currentBlockProposals
	}
	// collection the fee for this proposal (deposit and other fee , sc rent fee ...)
	state.SetBalance(proposer, new(big.Int).Sub(state.GetBalance(proposer), currentProposalPay))

	return append(currentBlockProposals, proposal)
}

func (a *Alien) processEventDeclare(currentBlockDeclares []Declare, txDataInfo []string, tx *types.Transaction, declarer common.Address) []Declare {
	if len(txDataInfo) <= posEventDeclare+2 {
		return currentBlockDeclares
	}
	declare := Declare{
		ProposalHash: common.Hash{},
		Declarer:     declarer,
		Decision:     true,
	}
	for i := 0; i < len(txDataInfo[posEventDeclare+1:])/2; i++ {
		k, v := txDataInfo[posEventDeclare+1+i*2], txDataInfo[posEventDeclare+2+i*2]
		switch k {
		case "hash":
			declare.ProposalHash.UnmarshalText([]byte(v))
		case "decision":
			if v == "yes" {
				declare.Decision = true
			} else if v == "no" {
				declare.Decision = false
			} else {
				return currentBlockDeclares
			}
		}
	}

	return append(currentBlockDeclares, declare)
}

func (a *Alien) processEventVote(currentBlockVotes []Vote, state *state.StateDB, tx *types.Transaction, voter common.Address) []Vote {

	a.lock.RLock()
	stake := state.GetBalance(voter)
	a.lock.RUnlock()

	currentBlockVotes = append(currentBlockVotes, Vote{
		Voter:     voter,
		Candidate: *tx.To(),
		Stake:     stake,
	})

	return currentBlockVotes
}

func (a *Alien) processEventConfirm(currentBlockConfirmations []Confirmation, chain consensus.ChainReader, txDataInfo []string, number uint64, tx *types.Transaction, confirmer common.Address, refundHash RefundHash) ([]Confirmation, RefundHash) {
	if len(txDataInfo) > posEventConfirmNumber {
		confirmedBlockNumber := new(big.Int)
		err := confirmedBlockNumber.UnmarshalText([]byte(txDataInfo[posEventConfirmNumber]))
		if err != nil || number-confirmedBlockNumber.Uint64() > a.config.MaxSignerCount || number-confirmedBlockNumber.Uint64() < 0 {
			return currentBlockConfirmations, refundHash
		}
		// check if the voter is in block
		confirmedHeader := chain.GetHeaderByNumber(confirmedBlockNumber.Uint64())
		if confirmedHeader == nil {
			//log.Info("Fail to get confirmedHeader")
			return currentBlockConfirmations, refundHash
		}
		confirmedHeaderExtra := HeaderExtra{}
		if extraVanity+extraSeal > len(confirmedHeader.Extra) {
			return currentBlockConfirmations, refundHash
		}
		err = decodeHeaderExtra(a.config, confirmedBlockNumber, confirmedHeader.Extra[extraVanity:len(confirmedHeader.Extra)-extraSeal], &confirmedHeaderExtra)
		if err != nil {
			log.Info("Fail to decode parent header", "err", err)
			return currentBlockConfirmations, refundHash
		}
		for _, s := range confirmedHeaderExtra.SignerQueue {
			if s == confirmer {
				currentBlockConfirmations = append(currentBlockConfirmations, Confirmation{
					Signer:      confirmer,
					BlockNumber: new(big.Int).Set(confirmedBlockNumber),
				})
				refundHash[tx.Hash()] = RefundPair{confirmer, tx.GasPrice()}
				break
			}
		}
	}

	return currentBlockConfirmations, refundHash
}

func (a *Alien) processPredecessorVoter(modifyPredecessorVotes []Vote, state *state.StateDB, tx *types.Transaction, voter common.Address, snap *Snapshot) []Vote {
	// process normal transaction which relate to voter
	if tx.Value().Cmp(big.NewInt(0)) > 0 && tx.To() != nil {
		if snap.isVoter(voter) {
			a.lock.RLock()
			stake := state.GetBalance(voter)
			a.lock.RUnlock()
			modifyPredecessorVotes = append(modifyPredecessorVotes, Vote{
				Voter:     voter,
				Candidate: common.Address{},
				Stake:     stake,
			})
		}
		if snap.isVoter(*tx.To()) {
			a.lock.RLock()
			stake := state.GetBalance(*tx.To())
			a.lock.RUnlock()
			modifyPredecessorVotes = append(modifyPredecessorVotes, Vote{
				Voter:     *tx.To(),
				Candidate: common.Address{},
				Stake:     stake,
			})
		}

	}
	return modifyPredecessorVotes
}
