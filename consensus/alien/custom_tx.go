// Copyright 2017 The gttc Authors
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
	ufoEventVote          = "vote"
	ufoEventConfirm       = "confirm"
	ufoEventPorposal      = "proposal"
	ufoEventDeclare       = "declare"
	ufoMinSplitLen        = 3
	posPrefix             = 0
	posVersion            = 1
	posCategory           = 2
	posEventVote          = 3
	posEventConfirm       = 3
	posEventProposal      = 3
	posEventDeclare       = 3
	posEventConfirmNumber = 4

	/*
	 * decision type
	 */
	decisionTypeWaitTillEnd = 1
	decisionTypeImmediately = 2

	/*
	 *  proposal type
	 */
	proposalTypeCandidateAdd                  = 1
	proposalTypeCandidateRemove               = 2
	proposalTypeMinerRewardDistributionModify = 3 // count in one thousand

	/*
	 * proposal related
	 */
	defaultValidationCnt = 30000
)

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
	ValidationCnt          uint64         // validation block number length of this proposal from the received block number
	ImplementNumber        *big.Int       // block number to implement modification in this proposal
	DecisionType           uint64         // success if condition fill / success if condition fill and block number reach ValidationCnt
	ProposalType           uint64         // type of proposal 1 - add candidate 2 - remove candidate ...
	Proposer               common.Address //
	Candidate              common.Address
	MinerRewardPerThousand uint64
	Enable                 bool       // Result of proposal
	Declares               []*Declare // Declare this proposal received
}

func (p *Proposal) copy() *Proposal {
	cpy := &Proposal{
		Hash:                   p.Hash,
		ValidationCnt:          p.ValidationCnt,
		ImplementNumber:        new(big.Int).Set(p.ImplementNumber),
		DecisionType:           p.DecisionType,
		ProposalType:           p.ProposalType,
		Proposer:               p.Proposer,
		Candidate:              p.Candidate,
		MinerRewardPerThousand: p.MinerRewardPerThousand,
		Enable:                 p.Enable,
		Declares:               make([]*Declare, len(p.Declares)),
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

// HeaderExtra is the struct of info in header.Extra[extraVanity:len(header.extra)-extraSeal]
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
}

// Calculate Votes from transaction in this block, write into header.Extra
func (a *Alien) processCustomTx(headerExtra HeaderExtra, chain consensus.ChainReader, header *types.Header, state *state.StateDB, txs []*types.Transaction) (HeaderExtra, error) {
	// if predecessor voter make transaction and vote in this block,
	// just process as vote, do it in snapshot.apply
	var (
		snap   *Snapshot
		err    error
		number uint64
	)
	number = header.Number.Uint64()
	if number > 1 {
		snap, err = a.snapshot(chain, number-1, header.ParentHash, nil, nil, defaultLoopCntRecalculateSigners)
		if err != nil {
			return headerExtra, err
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
								if txDataInfo[posEventVote] == ufoEventVote && snap.isCandidate(*tx.To()) {
									headerExtra.CurrentBlockVotes = a.processEventVote(headerExtra.CurrentBlockVotes, state, tx, txSender)
								} else if txDataInfo[posEventConfirm] == ufoEventConfirm {
									headerExtra.CurrentBlockConfirmations = a.processEventConfirm(headerExtra.CurrentBlockConfirmations, chain, txDataInfo, number, tx, txSender)
								} else if txDataInfo[posEventProposal] == ufoEventPorposal && snap.isCandidate(txSender) {
									headerExtra.CurrentBlockProposals = a.processEventProposal(headerExtra.CurrentBlockProposals, txDataInfo, tx, txSender)
								} else if txDataInfo[posEventDeclare] == ufoEventDeclare && snap.isCandidate(txSender) {
									headerExtra.CurrentBlockDeclares = a.processEventDeclare(headerExtra.CurrentBlockDeclares, txDataInfo, tx, txSender)
								}

								// if value is not zero, this vote may influence the balance of tx.To()
								if tx.Value().Cmp(big.NewInt(0)) == 0 {
									continue
								}

							} else {
								// todo : something wrong, leave this transaction to process as normal transaction
							}
						} else if txDataInfo[posCategory] == ufoCategoryLog {
							// todo :
						}
					}
				}
			}
		}

		if number > 1 {
			headerExtra.ModifyPredecessorVotes = a.processPredecessorVoter(headerExtra.ModifyPredecessorVotes, state, tx, txSender, snap)
		}

	}

	return headerExtra, nil
}

func (a *Alien) processEventProposal(currentBlockProposals []Proposal, txDataInfo []string, tx *types.Transaction, proposer common.Address) []Proposal {

	proposal := Proposal{
		Hash:                   tx.Hash(),
		ValidationCnt:          defaultValidationCnt,
		ImplementNumber:        big.NewInt(1),
		DecisionType:           decisionTypeImmediately,
		ProposalType:           proposalTypeCandidateAdd,
		Proposer:               proposer,
		Candidate:              common.Address{},
		MinerRewardPerThousand: defaultMinerRewardPerThousand,
		Enable:                 false,
		Declares:               []*Declare{},
	}

	for i := 0; i < len(txDataInfo[posEventProposal+1:])/2; i++ {
		k, v := txDataInfo[posEventProposal+1+i*2], txDataInfo[posEventProposal+2+i*2]
		switch k {
		case "validation_cnt":
			if validationCnt, err := strconv.Atoi(v); err != nil || validationCnt <= 0 {
				return currentBlockProposals
			} else {
				proposal.ValidationCnt = uint64(validationCnt)
			}
		case "implement_number":
			if implementNumber, err := strconv.Atoi(v); err != nil || implementNumber <= 0 {
				return currentBlockProposals
			} else {
				proposal.ImplementNumber = big.NewInt(int64(implementNumber))
			}
		case "decision_type":
			if decisionType, err := strconv.Atoi(v); err != nil || (decisionType != decisionTypeWaitTillEnd && decisionType != decisionTypeImmediately) {
				return currentBlockProposals
			} else {
				proposal.DecisionType = uint64(decisionType)
			}
		case "proposal_type":
			if proposalType, err := strconv.Atoi(v); err != nil || (proposalType != proposalTypeCandidateAdd && proposalType != proposalTypeCandidateRemove && proposalType != proposalTypeMinerRewardDistributionModify) {
				return currentBlockProposals
			} else {
				proposal.ProposalType = uint64(proposalType)
			}
		case "candidate":
			// not check here
			proposal.Candidate.SetString(v)
		case "mrpt":
			// miner reward per thousand
			if mrpt, err := strconv.Atoi(v); err != nil || mrpt < 0 || mrpt > 1000 {
				return currentBlockProposals
			} else {
				proposal.MinerRewardPerThousand = uint64(mrpt)
			}

		}
	}

	return append(currentBlockProposals, proposal)
}

func (a *Alien) processEventDeclare(currentBlockDeclares []Declare, txDataInfo []string, tx *types.Transaction, declarer common.Address) []Declare {

	declare := Declare{
		ProposalHash: common.Hash{},
		Declarer:     declarer,
		Decision:     true,
	}

	for i := 0; i < len(txDataInfo[posEventProposal+1:])/2; i++ {
		k, v := txDataInfo[posEventProposal+1+i*2], txDataInfo[posEventProposal+2+i*2]
		switch k {
		case "hash":
			declare.ProposalHash.SetString(v)
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
	if state.GetBalance(voter).Cmp(a.config.MinVoterBalance) > 0 {

		a.lock.RLock()
		stake := state.GetBalance(voter)
		a.lock.RUnlock()

		currentBlockVotes = append(currentBlockVotes, Vote{
			Voter:     voter,
			Candidate: *tx.To(),
			Stake:     stake,
		})
	}

	return currentBlockVotes
}

func (a *Alien) processEventConfirm(currentBlockConfirmations []Confirmation, chain consensus.ChainReader, txDataInfo []string, number uint64, tx *types.Transaction, confirmer common.Address) []Confirmation {
	if len(txDataInfo) >= posEventConfirmNumber {
		confirmedBlockNumber, err := strconv.Atoi(txDataInfo[posEventConfirmNumber])
		if err != nil || number-uint64(confirmedBlockNumber) > a.config.MaxSignerCount || number-uint64(confirmedBlockNumber) < 0 {
			return currentBlockConfirmations
		}
		// check if the voter is in block
		confirmedHeader := chain.GetHeaderByNumber(uint64(confirmedBlockNumber))
		if confirmedHeader == nil {
			log.Info("Fail to get confirmedHeader")
			return currentBlockConfirmations
		}
		confirmedHeaderExtra := HeaderExtra{}
		if extraVanity+extraSeal > len(confirmedHeader.Extra) {
			return currentBlockConfirmations
		}
		err = rlp.DecodeBytes(confirmedHeader.Extra[extraVanity:len(confirmedHeader.Extra)-extraSeal], &confirmedHeaderExtra)
		if err != nil {
			log.Info("Fail to decode parent header", "err", err)
			return currentBlockConfirmations
		}
		for _, s := range confirmedHeaderExtra.SignerQueue {
			if s == confirmer {
				currentBlockConfirmations = append(currentBlockConfirmations, Confirmation{
					Signer:      confirmer,
					BlockNumber: big.NewInt(int64(confirmedBlockNumber)),
				})
				break
			}
		}
	}

	return currentBlockConfirmations
}

func (a *Alien) processPredecessorVoter(modifyPredecessorVotes []Vote, state *state.StateDB, tx *types.Transaction, voter common.Address, snap *Snapshot) []Vote {
	// process normal transaction which relate to voter
	if tx.Value().Cmp(big.NewInt(0)) > 0 {
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
