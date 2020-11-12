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
	"encoding/json"
	"errors"
	"github.com/TTCECO/gttc/common"
	"github.com/TTCECO/gttc/core/types"
	"github.com/TTCECO/gttc/ethdb"
	"github.com/TTCECO/gttc/params"
	"github.com/hashicorp/golang-lru"
	"math/big"
	"sort"
	"strings"
	"time"
)

const (
	defaultFullCredit               = 28800 // no punished
	missingPublishCredit            = 100   // punished for missing one block seal
	signRewardCredit                = 10    // seal one block
	autoRewardCredit                = 1     // credit auto recover for each block
	minCalSignerQueueCredit         = 10000 // when calculate the signerQueue
	defaultOfficialMaxSignerCount   = 21    // official max signer count
	defaultOfficialFirstLevelCount  = 10    // official first level , 100% in signer queue
	defaultOfficialSecondLevelCount = 20    // official second level, 60% in signer queue
	defaultOfficialThirdLevelCount  = 30    // official third level, 40% in signer queue
	defaultOfficialMaxValidCount    = 50    // official max valid candidate count, sort by vote

	maxUncheckBalanceVoteCount = 10000 // not check current balance when calculate expired
	// the credit of one signer is at least minCalSignerQueueCredit
	candidateStateNormal = 1
	candidateMaxLen      = 500 // if candidateNeedPD is false and candidate is more than candidateMaxLen, then minimum tickets candidates will be remove in each LCRS*loop
	// reward for side chain
	scRewardDelayLoopCount     = 0                          //
	scRewardExpiredLoopCount   = scRewardDelayLoopCount + 4 //
	scMaxCountPerPeriod        = 6
	scMaxConfirmedRecordLength = defaultOfficialMaxSignerCount * 50 // max record length for each side chain
	// proposal refund
	proposalRefundDelayLoopCount   = 0
	proposalRefundExpiredLoopCount = proposalRefundDelayLoopCount + 2
	// notice
	mcNoticeClearDelayLoopCount = 4 // this count can be hundreds times
	scNoticeClearDelayLoopCount = mcNoticeClearDelayLoopCount * scMaxCountPerPeriod * 2
	scGasChargingDelayLoopCount = 1 // 1 is always enough
	// bug fix
	bugFixBlockNumber = 14456164   // fix bug for header
)

var (
	errIncorrectTallyCount = errors.New("incorrect tally count")
	errAllStakeMissing     = errors.New("all stake for this signer is zero")
)

// SCCurrentBlockReward is base on scMaxCountPerPeriod = 6
var SCCurrentBlockReward = map[uint64]map[uint64]uint64{1: {1: 100},
	2: {1: 30, 2: 70},
	3: {1: 15, 2: 30, 3: 55},
	4: {1: 5, 2: 15, 3: 30, 4: 50},
	5: {1: 5, 2: 10, 3: 15, 4: 25, 5: 45},
	6: {1: 1, 2: 4, 3: 10, 4: 15, 5: 25, 6: 45}}

// Score to calculate at one main chain block, for calculate the side chain reward
type SCBlockReward struct {
	RewardScoreMap map[common.Address]uint64 //sum(this value) in one period == 100
}

// Record for one side chain
type SCReward struct {
	SCBlockRewardMap map[uint64]*SCBlockReward
}

type SCRentInfo struct {
	RentPerPeriod   *big.Int `json:"rentPerPeriod"`
	MaxRewardNumber *big.Int `json:"maxRewardNumber"`
}

// SCRecord is the state record for side chain
type SCRecord struct {
	Record              map[uint64][]*SCConfirmation `json:"record"`              // Confirmation Record of one side chain
	LastConfirmedNumber uint64                       `json:"lastConfirmedNumber"` // Last confirmed header number of one side chain
	MaxHeaderNumber     uint64                       `json:"maxHeaderNumber"`     // max header number of one side chain
	CountPerPeriod      uint64                       `json:"countPerPeriod"`      // block sealed per period on this side chain
	RewardPerPeriod     uint64                       `json:"rewardPerPeriod"`     // full reward per period, number per thousand
	RentReward          map[common.Hash]*SCRentInfo  `json:"rentReward"`          // reward info by rent
}

type NoticeCR struct {
	NRecord map[common.Address]bool `json:"noticeConfirmRecord"`
	Number  uint64                  `json:"firstReceivedNumber"` // this number will fill when there are more than 2/3+1 maxSignerCnt
	Type    uint64                  `json:"noticeType"`
	Success bool                    `json:"success"`
}

// CCNotice (cross chain notice) contain the information main chain need to notify given side chain
//
type CCNotice struct {
	CurrentCharging map[common.Hash]GasCharging `json:"currentCharging"` // common.Hash here is the proposal txHash not the hash of side chain
	ConfirmReceived map[common.Hash]NoticeCR    `json:"confirmReceived"` // record the confirm address
}

// Snapshot is the state of the authorization voting at a given point in time.
type Snapshot struct {
	config   *params.AlienConfig // Consensus engine parameters to fine tune behavior
	sigcache *lru.ARCCache       // Cache of recent block signatures to speed up ecrecover
	LCRS     uint64              // Loop count to recreate signers from top tally

	Period          uint64                                            `json:"period"`            // Period of seal each block
	Number          uint64                                            `json:"number"`            // Block number where the snapshot was created
	ConfirmedNumber uint64                                            `json:"confirmedNumber"`   // Block number confirmed when the snapshot was created
	Hash            common.Hash                                       `json:"hash"`              // Block hash where the snapshot was created
	HistoryHash     []common.Hash                                     `json:"historyHash"`       // Block hash list for two recent loop
	Signers         []*common.Address                                 `json:"signers"`           // Signers queue in current header
	Votes           map[common.Address]*Vote                          `json:"votes"`             // All validate votes from genesis block
	Tally           map[common.Address]*big.Int                       `json:"tally"`             // Stake for each candidate address
	Voters          map[common.Address]*big.Int                       `json:"voters"`            // Block number for each voter address
	Candidates      map[common.Address]uint64                         `json:"candidates"`        // Candidates for Signers (0- adding procedure 1- normal 2- removing procedure)
	Punished        map[common.Address]uint64                         `json:"punished"`          // The signer be punished count cause of missing seal
	Confirmations   map[uint64][]*common.Address                      `json:"confirms"`          // The signer confirm given block number
	Proposals       map[common.Hash]*Proposal                         `json:"proposals"`         // The Proposals going or success (failed proposal will be removed)
	HeaderTime      uint64                                            `json:"headerTime"`        // Time of the current header
	LoopStartTime   uint64                                            `json:"loopStartTime"`     // Start Time of the current loop
	ProposalRefund  map[uint64]map[common.Address]*big.Int            `json:"proposalRefund"`    // Refund proposal deposit
	SCCoinbase      map[common.Address]map[common.Hash]common.Address `json:"sideChainCoinbase"` // main chain set Coinbase of side chain setting
	SCRecordMap     map[common.Hash]*SCRecord                         `json:"sideChainRecord"`   // main chain record Confirmation of side chain setting
	SCRewardMap     map[common.Hash]*SCReward                         `json:"sideChainReward"`   // main chain record Side Chain Reward
	SCNoticeMap     map[common.Hash]*CCNotice                         `json:"sideChainNotice"`   // main chain record Notification to side chain
	LocalNotice     *CCNotice                                         `json:"localNotice"`       // side chain record Notification
	MinerReward     uint64                                            `json:"minerReward"`       // miner reward per thousand
	MinVB           *big.Int                                          `json:"minVoterBalance"`   // min voter balance
}

// newSnapshot creates a new snapshot with the specified startup parameters. only ever use if for
// the genesis block.
func newSnapshot(config *params.AlienConfig, sigcache *lru.ARCCache, hash common.Hash, votes []*Vote, lcrs uint64) *Snapshot {

	snap := &Snapshot{
		config:          config,
		sigcache:        sigcache,
		LCRS:            lcrs,
		Period:          config.Period,
		Number:          0,
		ConfirmedNumber: 0,
		Hash:            hash,
		HistoryHash:     []common.Hash{},
		Signers:         []*common.Address{},
		Votes:           make(map[common.Address]*Vote),
		Tally:           make(map[common.Address]*big.Int),
		Voters:          make(map[common.Address]*big.Int),
		Punished:        make(map[common.Address]uint64),
		Candidates:      make(map[common.Address]uint64),
		Confirmations:   make(map[uint64][]*common.Address),
		Proposals:       make(map[common.Hash]*Proposal),
		HeaderTime:      uint64(time.Now().Unix()) - 1,
		LoopStartTime:   config.GenesisTimestamp,
		SCCoinbase:      make(map[common.Address]map[common.Hash]common.Address),
		SCRecordMap:     make(map[common.Hash]*SCRecord),
		SCRewardMap:     make(map[common.Hash]*SCReward),
		SCNoticeMap:     make(map[common.Hash]*CCNotice),
		LocalNotice:     &CCNotice{CurrentCharging: make(map[common.Hash]GasCharging), ConfirmReceived: make(map[common.Hash]NoticeCR)},
		ProposalRefund:  make(map[uint64]map[common.Address]*big.Int),
		MinerReward:     minerRewardPerThousand,
		MinVB:           config.MinVoterBalance,
	}
	snap.HistoryHash = append(snap.HistoryHash, hash)

	for _, vote := range votes {
		// init Votes from each vote
		snap.Votes[vote.Voter] = vote
		// init Tally
		_, ok := snap.Tally[vote.Candidate]
		if !ok {
			snap.Tally[vote.Candidate] = big.NewInt(0)
		}
		snap.Tally[vote.Candidate].Add(snap.Tally[vote.Candidate], vote.Stake)
		// init Voters
		snap.Voters[vote.Voter] = big.NewInt(0) // block number is 0 , vote in genesis block
		// init Candidates
		snap.Candidates[vote.Voter] = candidateStateNormal
	}

	if len(config.SelfVoteSigners) > 0 {
		var prefixSelfVoteSigners []common.Address
		for _, unPrefixSelfVoteSigners := range config.SelfVoteSigners {
			prefixSelfVoteSigners = append(prefixSelfVoteSigners, common.Address(unPrefixSelfVoteSigners))
		}
		for i := 0; i < int(config.MaxSignerCount); i++ {
			snap.Signers = append(snap.Signers, &prefixSelfVoteSigners[i%len(prefixSelfVoteSigners)])
		}
	}

	return snap
}

// loadSnapshot loads an existing snapshot from the database.
func loadSnapshot(config *params.AlienConfig, sigcache *lru.ARCCache, db ethdb.Database, hash common.Hash) (*Snapshot, error) {
	blob, err := db.Get(append([]byte("alien-"), hash[:]...))
	if err != nil {
		return nil, err
	}
	snap := new(Snapshot)
	if err := json.Unmarshal(blob, snap); err != nil {
		return nil, err
	}
	snap.config = config
	snap.sigcache = sigcache

	// miner reward per thousand proposal must larger than 0
	// so minerReward is zeron only when update the program
	if snap.MinerReward == 0 {
		snap.MinerReward = minerRewardPerThousand
	}
	if snap.MinVB == nil {
		snap.MinVB = new(big.Int).Set(minVoterBalance)
	}
	return snap, nil
}

// store inserts the snapshot into the database.
func (s *Snapshot) store(db ethdb.Database) error {
	blob, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return db.Put(append([]byte("alien-"), s.Hash[:]...), blob)
}

// copy creates a deep copy of the snapshot, though not the individual votes.
func (s *Snapshot) copy() *Snapshot {
	cpy := &Snapshot{
		config:          s.config,
		sigcache:        s.sigcache,
		LCRS:            s.LCRS,
		Period:          s.Period,
		Number:          s.Number,
		ConfirmedNumber: s.ConfirmedNumber,
		Hash:            s.Hash,
		HistoryHash:     make([]common.Hash, len(s.HistoryHash)),

		Signers:       make([]*common.Address, len(s.Signers)),
		Votes:         make(map[common.Address]*Vote),
		Tally:         make(map[common.Address]*big.Int),
		Voters:        make(map[common.Address]*big.Int),
		Candidates:    make(map[common.Address]uint64),
		Punished:      make(map[common.Address]uint64),
		Proposals:     make(map[common.Hash]*Proposal),
		Confirmations: make(map[uint64][]*common.Address),

		HeaderTime:     s.HeaderTime,
		LoopStartTime:  s.LoopStartTime,
		SCCoinbase:     make(map[common.Address]map[common.Hash]common.Address),
		SCRecordMap:    make(map[common.Hash]*SCRecord),
		SCRewardMap:    make(map[common.Hash]*SCReward),
		SCNoticeMap:    make(map[common.Hash]*CCNotice),
		LocalNotice:    &CCNotice{CurrentCharging: make(map[common.Hash]GasCharging), ConfirmReceived: make(map[common.Hash]NoticeCR)},
		ProposalRefund: make(map[uint64]map[common.Address]*big.Int),

		MinerReward: s.MinerReward,
		MinVB:       nil,
	}
	copy(cpy.HistoryHash, s.HistoryHash)
	copy(cpy.Signers, s.Signers)
	for voter, vote := range s.Votes {
		cpy.Votes[voter] = &Vote{
			Voter:     vote.Voter,
			Candidate: vote.Candidate,
			Stake:     new(big.Int).Set(vote.Stake),
		}
	}
	for candidate, tally := range s.Tally {
		cpy.Tally[candidate] = new(big.Int).Set(tally)
	}
	for voter, number := range s.Voters {
		cpy.Voters[voter] = new(big.Int).Set(number)
	}
	for candidate, state := range s.Candidates {
		cpy.Candidates[candidate] = state
	}
	for signer, cnt := range s.Punished {
		cpy.Punished[signer] = cnt
	}
	for blockNumber, confirmers := range s.Confirmations {
		cpy.Confirmations[blockNumber] = make([]*common.Address, len(confirmers))
		copy(cpy.Confirmations[blockNumber], confirmers)
	}
	for txHash, proposal := range s.Proposals {
		cpy.Proposals[txHash] = proposal.copy()
	}
	for signer, sc := range s.SCCoinbase {
		cpy.SCCoinbase[signer] = make(map[common.Hash]common.Address)
		for hash, addr := range sc {
			cpy.SCCoinbase[signer][hash] = addr
		}
	}
	for hash, scc := range s.SCRecordMap {
		cpy.SCRecordMap[hash] = &SCRecord{
			LastConfirmedNumber: scc.LastConfirmedNumber,
			MaxHeaderNumber:     scc.MaxHeaderNumber,
			CountPerPeriod:      scc.CountPerPeriod,
			RewardPerPeriod:     scc.RewardPerPeriod,
			Record:              make(map[uint64][]*SCConfirmation),
			RentReward:          make(map[common.Hash]*SCRentInfo),
		}
		for number, scConfirmation := range scc.Record {
			cpy.SCRecordMap[hash].Record[number] = make([]*SCConfirmation, len(scConfirmation))
			copy(cpy.SCRecordMap[hash].Record[number], scConfirmation)
		}
		for rentHash, scRentInfo := range scc.RentReward {
			cpy.SCRecordMap[hash].RentReward[rentHash] = &SCRentInfo{new(big.Int).Set(scRentInfo.RentPerPeriod), new(big.Int).Set(scRentInfo.MaxRewardNumber)}
		}
	}

	for hash, sca := range s.SCRewardMap {
		cpy.SCRewardMap[hash] = &SCReward{
			SCBlockRewardMap: make(map[uint64]*SCBlockReward),
		}
		for number, blockReward := range sca.SCBlockRewardMap {
			cpy.SCRewardMap[hash].SCBlockRewardMap[number] = &SCBlockReward{
				RewardScoreMap: make(map[common.Address]uint64),
			}
			for addr, score := range blockReward.RewardScoreMap {
				cpy.SCRewardMap[hash].SCBlockRewardMap[number].RewardScoreMap[addr] = score
			}
		}
	}

	for hash, scn := range s.SCNoticeMap {
		cpy.SCNoticeMap[hash] = &CCNotice{
			CurrentCharging: make(map[common.Hash]GasCharging),
			ConfirmReceived: make(map[common.Hash]NoticeCR),
		}
		for txHash, charge := range scn.CurrentCharging {
			cpy.SCNoticeMap[hash].CurrentCharging[txHash] = GasCharging{charge.Target, charge.Volume, charge.Hash}
		}
		for txHash, confirm := range scn.ConfirmReceived {
			cpy.SCNoticeMap[hash].ConfirmReceived[txHash] = NoticeCR{make(map[common.Address]bool), confirm.Number, confirm.Type, confirm.Success}
			for addr, b := range confirm.NRecord {
				cpy.SCNoticeMap[hash].ConfirmReceived[txHash].NRecord[addr] = b
			}
		}
	}

	for txHash, charge := range s.LocalNotice.CurrentCharging {
		cpy.LocalNotice.CurrentCharging[txHash] = GasCharging{charge.Target, charge.Volume, charge.Hash}
	}
	for txHash, confirm := range s.LocalNotice.ConfirmReceived {
		cpy.LocalNotice.ConfirmReceived[txHash] = NoticeCR{make(map[common.Address]bool), confirm.Number, confirm.Type, confirm.Success}
		for addr, b := range confirm.NRecord {
			cpy.LocalNotice.ConfirmReceived[txHash].NRecord[addr] = b
		}
	}

	for number, refund := range s.ProposalRefund {
		cpy.ProposalRefund[number] = make(map[common.Address]*big.Int)
		for proposer, deposit := range refund {
			cpy.ProposalRefund[number][proposer] = new(big.Int).Set(deposit)
		}
	}
	// miner reward per thousand proposal must larger than 0
	// so minerReward is zeron only when update the program
	if s.MinerReward == 0 {
		cpy.MinerReward = minerRewardPerThousand
	}
	if s.MinVB == nil {
		cpy.MinVB = new(big.Int).Set(minVoterBalance)
	} else {
		cpy.MinVB = new(big.Int).Set(s.MinVB)
	}

	return cpy
}

// apply creates a new authorization snapshot by applying the given headers to
// the original one.
func (s *Snapshot) apply(headers []*types.Header) (*Snapshot, error) {
	// Allow passing in no headers for cleaner code
	if len(headers) == 0 {
		return s, nil
	}
	// Sanity check that the headers can be applied
	for i := 0; i < len(headers)-1; i++ {
		if headers[i+1].Number.Uint64() != headers[i].Number.Uint64()+1 {
			return nil, errInvalidVotingChain
		}
	}
	if headers[0].Number.Uint64() != s.Number+1 {
		return nil, errInvalidVotingChain
	}
	// Iterate through the headers and create a new snapshot
	snap := s.copy()

	for _, header := range headers {
		// Resolve the authorization key and check against signers
		coinbase, err := ecrecover(header, s.sigcache)
		if err != nil {
			return nil, err
		}
		if coinbase.Str() != header.Coinbase.Str() && header.Number.Cmp(big.NewInt(bugFixBlockNumber)) != 0{
			return nil, errUnauthorized
		}

		headerExtra := HeaderExtra{}
		err = decodeHeaderExtra(s.config, header.Number, header.Extra[extraVanity:len(header.Extra)-extraSeal], &headerExtra)
		if err != nil {
			return nil, err
		}
		snap.HeaderTime = header.Time.Uint64()
		snap.LoopStartTime = headerExtra.LoopStartTime
		snap.Signers = nil
		for i := range headerExtra.SignerQueue {
			snap.Signers = append(snap.Signers, &headerExtra.SignerQueue[i])
		}

		snap.ConfirmedNumber = headerExtra.ConfirmedBlockNumber

		if len(snap.HistoryHash) >= int(s.config.MaxSignerCount)*2 {
			snap.HistoryHash = snap.HistoryHash[1 : int(s.config.MaxSignerCount)*2]
		}
		snap.HistoryHash = append(snap.HistoryHash, header.Hash())

		// deal the new confirmation in this block
		snap.updateSnapshotByConfirmations(headerExtra.CurrentBlockConfirmations)

		// deal the new vote from voter
		snap.updateSnapshotByVotes(headerExtra.CurrentBlockVotes, header.Number)

		// deal the voter which balance modified
		snap.updateSnapshotByMPVotes(headerExtra.ModifyPredecessorVotes)

		// deal the snap related with punished
		snap.updateSnapshotForPunish(headerExtra.SignerMissing, header.Number, header.Coinbase)

		// deal proposals
		snap.updateSnapshotByProposals(headerExtra.CurrentBlockProposals, header.Number)

		// deal declares
		snap.updateSnapshotByDeclares(headerExtra.CurrentBlockDeclares, header.Number)

		// deal trantor upgrade
		if snap.Period == 0 {
			snap.Period = snap.config.Period
		}

		// deal setcoinbase for side chain
		snap.updateSnapshotBySetSCCoinbase(headerExtra.SideChainSetCoinbases)

		// deal confirmation for side chain
		snap.updateSnapshotBySCConfirm(headerExtra.SideChainConfirmations, header.Number)

		// deal notice confirmation
		snap.updateSnapshotByNoticeConfirm(headerExtra.SideChainNoticeConfirmed, header.Number)

		// calculate proposal result
		snap.calculateProposalResult(header.Number)

		// check the len of candidate if not candidateNeedPD
		if !candidateNeedPD && (snap.Number+1)%(snap.config.MaxSignerCount*snap.LCRS) == 0 && len(snap.Candidates) > candidateMaxLen {
			snap.removeExtraCandidate()
		}

		/*
		 * follow methods only work on side chain !!!! not like above method
		 */

		// deal the notice from main chain
		snap.updateSnapshotBySCCharging(headerExtra.SideChainCharging, header.Number, header.Coinbase)

		snap.updateSnapshotForExpired(header.Number)
	}
	snap.Number += uint64(len(headers))
	snap.Hash = headers[len(headers)-1].Hash()

	err := snap.verifyTallyCnt()
	if err != nil {
		return nil, err
	}
	return snap, nil
}

func (s *Snapshot) removeExtraCandidate() {
	// remove minimum tickets tally beyond candidateMaxLen
	tallySlice := s.buildTallySlice()
	sort.Sort(TallySlice(tallySlice))
	if len(tallySlice) > candidateMaxLen {
		removeNeedTally := tallySlice[candidateMaxLen:]
		for _, tallySlice := range removeNeedTally {
			if _, ok := s.SCCoinbase[tallySlice.addr]; ok {
				delete(s.SCCoinbase, tallySlice.addr)
			}
			delete(s.Candidates, tallySlice.addr)
		}
	}
}

func (s *Snapshot) verifyTallyCnt() error {

	tallyTarget := make(map[common.Address]*big.Int)
	for _, v := range s.Votes {
		if _, ok := tallyTarget[v.Candidate]; ok {
			tallyTarget[v.Candidate].Add(tallyTarget[v.Candidate], v.Stake)
		} else {
			tallyTarget[v.Candidate] = new(big.Int).Set(v.Stake)
		}
	}

	for address, tally := range s.Tally {
		if targetTally, ok := tallyTarget[address]; ok && targetTally.Cmp(tally) == 0 {
			continue
		} else {
			return errIncorrectTallyCount
		}
	}

	return nil
}

func (s *Snapshot) updateSnapshotBySetSCCoinbase(scCoinbases []SCSetCoinbase) {
	for _, scc := range scCoinbases {
		if _, ok := s.SCCoinbase[scc.Signer]; !ok {
			s.SCCoinbase[scc.Signer] = make(map[common.Hash]common.Address)
		}
		s.SCCoinbase[scc.Signer][scc.Hash] = scc.Coinbase
	}
}

func (s *Snapshot) isSideChainCoinbase(sc common.Hash, address common.Address, realtime bool) bool {
	// check is side chain coinbase
	// is use the coinbase of main chain as coinbase of side chain , return false
	// the main chain cloud seal block, but not recommend for send confirm tx usually fail
	if realtime {
		for _, signer := range s.Signers {
			if _, ok := s.SCCoinbase[*signer]; ok {
				if coinbase, ok := s.SCCoinbase[*signer][sc]; ok && coinbase == address {
					return true
				}
			}
		}
	} else {
		for _, coinbaseMap := range s.SCCoinbase {
			if coinbase, ok := coinbaseMap[sc]; ok && coinbase == address {
				return true
			}
		}

	}
	return false
}

func (s *Snapshot) updateSnapshotBySCConfirm(scConfirmations []SCConfirmation, headerNumber *big.Int) {
	// todo ,if diff side chain coinbase send confirm for the same side chain , same number ...
	for _, scc := range scConfirmations {
		// new confirmation header number must larger than last confirmed number of this side chain
		if s.isSideChainCoinbase(scc.Hash, scc.Coinbase, false) {
			if _, ok := s.SCRecordMap[scc.Hash]; ok && scc.Number > s.SCRecordMap[scc.Hash].LastConfirmedNumber {
				s.SCRecordMap[scc.Hash].Record[scc.Number] = append(s.SCRecordMap[scc.Hash].Record[scc.Number], scc.copy())
				if scc.Number > s.SCRecordMap[scc.Hash].MaxHeaderNumber {
					s.SCRecordMap[scc.Hash].MaxHeaderNumber = scc.Number
				}
			}
		}
	}
	// calculate the side chain reward in each loop
	if (headerNumber.Uint64()+1)%s.config.MaxSignerCount == 0 {
		s.checkSCConfirmation(headerNumber)
		s.updateSCConfirmation(headerNumber)
	}
}

func (s *Snapshot) updateSnapshotByNoticeConfirm(scNoticeConfirmed []SCConfirmation, headerNumber *big.Int) {
	// record the confirmed info into Notice, and remove notice if there are enough confirm
	// may be receive confirmed more than 2/3+1 and the remove will delay a reasonable loop count (4)
	for _, noticeConfirm := range scNoticeConfirmed {
		// check if the coinbase of this side chain
		// todo check if the current coinbase of this side chain.
		if !s.isSideChainCoinbase(noticeConfirm.Hash, noticeConfirm.Coinbase, true) {
			continue
		}
		// noticeConfirm.Hash is the hash of side chain
		if _, ok := s.SCNoticeMap[noticeConfirm.Hash]; ok {
			for _, strHash := range noticeConfirm.LoopInfo {
				// check the charging current exist
				noticeHash := common.HexToHash(strHash)
				if _, ok := s.SCNoticeMap[noticeConfirm.Hash].CurrentCharging[noticeHash]; ok {
					//noticeType = noticeTypeGasCharging
					if _, ok := s.SCNoticeMap[noticeConfirm.Hash].ConfirmReceived[noticeHash]; !ok {
						s.SCNoticeMap[noticeConfirm.Hash].ConfirmReceived[noticeHash] = NoticeCR{make(map[common.Address]bool), 0, noticeTypeGasCharging, false}
					}
					s.SCNoticeMap[noticeConfirm.Hash].ConfirmReceived[noticeHash].NRecord[noticeConfirm.Coinbase] = true
				}
			}
		}
	}

	// check notice confirm number
	if (headerNumber.Uint64()+1)%s.config.MaxSignerCount == 0 {
		// todo : check if the enough coinbase is the side chain coinbase which main chain coinbase is in the signers
		// todo : if checked ,then update the number in noticeConfirmed
		// todo : remove the notice , delete(notice,hash) to stop the broadcast to side chain

		for chainHash, scNotice := range s.SCNoticeMap {
			// check each side chain
			for noticeHash, noticeRecord := range scNotice.ConfirmReceived {
				if len(noticeRecord.NRecord) >= int(2*s.config.MaxSignerCount/3+1) && !noticeRecord.Success {
					s.SCNoticeMap[chainHash].ConfirmReceived[noticeHash] = NoticeCR{noticeRecord.NRecord, headerNumber.Uint64(), noticeRecord.Type, true}
				}

				if noticeRecord.Success && noticeRecord.Number < headerNumber.Uint64()-s.config.MaxSignerCount*mcNoticeClearDelayLoopCount {
					delete(s.SCNoticeMap[chainHash].CurrentCharging, noticeHash)
					delete(s.SCNoticeMap[chainHash].ConfirmReceived, noticeHash)
				}
			}
		}
	}

}

func (s *Snapshot) updateSnapshotBySCCharging(scCharging []GasCharging, headerNumber *big.Int, coinbase common.Address) {
	for _, charge := range scCharging {
		if _, ok := s.LocalNotice.CurrentCharging[charge.Hash]; !ok {
			s.LocalNotice.CurrentCharging[charge.Hash] = GasCharging{charge.Target, charge.Volume, charge.Hash}
			s.LocalNotice.ConfirmReceived[charge.Hash] = NoticeCR{make(map[common.Address]bool), 0, noticeTypeGasCharging, false}

		}
		s.LocalNotice.ConfirmReceived[charge.Hash].NRecord[coinbase] = true
	}

	if (headerNumber.Uint64()+1)%s.config.MaxSignerCount == 0 {
		for hash, noticeRecord := range s.LocalNotice.ConfirmReceived {
			if len(noticeRecord.NRecord) >= int(2*s.config.MaxSignerCount/3+1) && !noticeRecord.Success {
				s.LocalNotice.ConfirmReceived[hash] = NoticeCR{noticeRecord.NRecord, headerNumber.Uint64(), noticeTypeGasCharging, true}
				// todo charging the gas fee on set block

			}
			if noticeRecord.Success && noticeRecord.Number < headerNumber.Uint64()-s.config.MaxSignerCount*scNoticeClearDelayLoopCount {
				delete(s.LocalNotice.CurrentCharging, hash)
				delete(s.LocalNotice.ConfirmReceived, hash)
			}
		}
	}

}

func (s *Snapshot) checkSCConfirmation(headerNumber *big.Int) {
	for hash, scRecord := range s.SCRecordMap {
		// check maxRentRewardNumber by headerNumber
		for txHash, scRentInfo := range scRecord.RentReward {
			if scRentInfo.MaxRewardNumber.Uint64() < headerNumber.Uint64()-scRewardExpiredLoopCount*s.config.MaxSignerCount {
				delete(s.SCRecordMap[hash].RentReward, txHash)
			}
		}

		// if size of confirmed record from one side chain larger than scMaxConfirmedRecordLength
		// we reset the record info of this side chain, good enough for now
		if len(scRecord.Record) > scMaxConfirmedRecordLength {
			s.SCRecordMap[hash].Record = make(map[uint64][]*SCConfirmation)
			s.SCRecordMap[hash].LastConfirmedNumber = 0
			s.SCRecordMap[hash].MaxHeaderNumber = 0
			// the rentReward info will be kept, do not delete
		}
	}

}

func (s *Snapshot) calculateSCConfirmedNumber(record *SCRecord, minConfirmedSignerCount int) (uint64, map[uint64]common.Address) {
	// todo : add params scHash, so can check if the address in SCRecord is belong to this side chain

	confirmedNumber := record.LastConfirmedNumber
	confirmedRecordMap := make(map[string]map[common.Address]bool)
	confirmedCoinbase := make(map[uint64]common.Address)
	sep := ":"
	tmpHeaderNum := new(big.Int)
	for i := record.LastConfirmedNumber + 1; i <= record.MaxHeaderNumber; i++ {
		if _, ok := record.Record[i]; ok {
			// during reorged, the side chain loop info may more than one for each side chain block number.
			for _, scConfirm := range record.Record[i] {
				// loopInfo slice contain number and coinbase address of side chain block,
				// so the length of loop info must larger than twice of minConfirmedSignerCount .
				if len(scConfirm.LoopInfo) >= minConfirmedSignerCount*2 {
					key := strings.Join(scConfirm.LoopInfo, sep)
					if _, ok := confirmedRecordMap[key]; !ok {
						confirmedRecordMap[key] = make(map[common.Address]bool)
					}
					// new coinbase for same loop info
					if _, ok := confirmedRecordMap[key][scConfirm.Coinbase]; !ok {
						confirmedRecordMap[key][scConfirm.Coinbase] = true
						if len(confirmedRecordMap[key]) >= minConfirmedSignerCount {
							err := tmpHeaderNum.UnmarshalText([]byte(scConfirm.LoopInfo[len(scConfirm.LoopInfo)-2]))
							if err == nil && tmpHeaderNum.Uint64() > confirmedNumber {
								confirmedNumber = tmpHeaderNum.Uint64()
							}
						}
					}
				}
			}
		}
	}

	for info, confirm := range confirmedRecordMap {
		if len(confirm) >= minConfirmedSignerCount {
			infos := strings.Split(info, sep)
			for i := 0; i+1 < len(infos); i += 2 {
				err := tmpHeaderNum.UnmarshalText([]byte(infos[i]))
				if err != nil {
					continue
				}
				confirmedCoinbase[tmpHeaderNum.Uint64()] = common.HexToAddress(infos[i+1])
			}
		}
	}

	// for calculate side chain reward
	// if the side chain count per period is more than one
	// then the reward should calculate continue till one coinbase finished.
	if record.CountPerPeriod > 1 && confirmedNumber > record.LastConfirmedNumber {
		if lastConfirmedCoinbase, ok := confirmedCoinbase[confirmedNumber]; ok {
			for i := confirmedNumber - 1; i > confirmedNumber-record.CountPerPeriod; i-- {
				if lastConfirmedCoinbase != confirmedCoinbase[i] {
					confirmedNumber = i
					break
				}
			}
			for i := confirmedNumber + 1; i < confirmedNumber+record.CountPerPeriod; i++ {
				if _, ok = confirmedCoinbase[i]; ok {
					delete(confirmedCoinbase, i)
				}
			}
		}
	}

	return confirmedNumber, confirmedCoinbase
}

func (s *Snapshot) calculateCurrentBlockReward(currentCount uint64, periodCount uint64) uint64 {
	currentRewardPercentage := uint64(0)
	if periodCount > uint64(scMaxCountPerPeriod) {
		periodCount = scMaxCountPerPeriod
	}
	if v, ok := SCCurrentBlockReward[periodCount][currentCount]; ok {
		currentRewardPercentage = v
	}
	return currentRewardPercentage
}

func (s *Snapshot) updateSCConfirmation(headerNumber *big.Int) {
	minConfirmedSignerCount := int(2 * s.config.MaxSignerCount / 3)
	for scHash, record := range s.SCRecordMap {
		if _, ok := s.SCRewardMap[scHash]; !ok {
			s.SCRewardMap[scHash] = &SCReward{SCBlockRewardMap: make(map[uint64]*SCBlockReward)}
		}
		currentReward := &SCBlockReward{RewardScoreMap: make(map[common.Address]uint64)}
		confirmedNumber, confirmedCoinbase := s.calculateSCConfirmedNumber(record, minConfirmedSignerCount)
		if confirmedNumber > record.LastConfirmedNumber {
			// todo: map coinbase of side chain to coin base of main chain here
			lastSCCoinbase := common.Address{}
			currentSCCoinbaseCount := uint64(0)
			for n := record.LastConfirmedNumber + 1; n <= confirmedNumber; n++ {
				if scCoinbase, ok := confirmedCoinbase[n]; ok {
					// if scCoinbase not same with lastSCCoinbase recount
					if lastSCCoinbase != scCoinbase {
						currentSCCoinbaseCount = 1
					} else {
						currentSCCoinbaseCount++
					}

					if _, ok := currentReward.RewardScoreMap[scCoinbase]; !ok {
						currentReward.RewardScoreMap[scCoinbase] = s.calculateCurrentBlockReward(currentSCCoinbaseCount, record.CountPerPeriod)
					} else {
						currentReward.RewardScoreMap[scCoinbase] += s.calculateCurrentBlockReward(currentSCCoinbaseCount, record.CountPerPeriod)
					}

					// update lastSCCoinbase
					lastSCCoinbase = scCoinbase
				}
			}

			for i := record.LastConfirmedNumber + 1; i <= confirmedNumber; i++ {
				if _, ok := s.SCRecordMap[scHash].Record[i]; ok {
					delete(s.SCRecordMap[scHash].Record, i)
				}
			}
			s.SCRecordMap[scHash].LastConfirmedNumber = confirmedNumber
		}
		// clear empty block number for side chain
		if len(currentReward.RewardScoreMap) != 0 {
			s.SCRewardMap[scHash].SCBlockRewardMap[headerNumber.Uint64()] = currentReward
		}
	}

	for scHash := range s.SCRewardMap {
		// clear expired side chain reward record
		for number := range s.SCRewardMap[scHash].SCBlockRewardMap {
			if number < headerNumber.Uint64()-scRewardExpiredLoopCount*s.config.MaxSignerCount {
				delete(s.SCRewardMap[scHash].SCBlockRewardMap, number)
			}
		}
		// clear this side chain if reward is empty
		if len(s.SCRewardMap[scHash].SCBlockRewardMap) == 0 {
			delete(s.SCRewardMap, scHash)
		}
	}

}

func (s *Snapshot) updateSnapshotByDeclares(declares []Declare, headerNumber *big.Int) {
	for _, declare := range declares {
		if proposal, ok := s.Proposals[declare.ProposalHash]; ok {
			// check the proposal enable status and valid block number
			if proposal.ReceivedNumber.Uint64()+proposal.ValidationLoopCnt*s.config.MaxSignerCount < headerNumber.Uint64() || !s.isCandidate(declare.Declarer) {
				continue
			}
			// check if this signer already declare on this proposal
			alreadyDeclare := false
			for _, v := range proposal.Declares {
				if v.Declarer.Str() == declare.Declarer.Str() {
					// this declarer already declare for this proposal
					alreadyDeclare = true
					break
				}
			}
			if alreadyDeclare {
				continue
			}
			// add declare to proposal
			s.Proposals[declare.ProposalHash].Declares = append(s.Proposals[declare.ProposalHash].Declares,
				&Declare{declare.ProposalHash, declare.Declarer, declare.Decision})

		}
	}
}

func (s *Snapshot) calculateProposalResult(headerNumber *big.Int) {
	// process the expire proposal refund record
	expiredHeaderNumber := headerNumber.Uint64() - proposalRefundExpiredLoopCount*s.config.MaxSignerCount
	if _, ok := s.ProposalRefund[expiredHeaderNumber]; ok {
		delete(s.ProposalRefund, expiredHeaderNumber)
	}

	for hashKey, proposal := range s.Proposals {
		// the result will be calculate at receiverdNumber + vlcnt + 1
		if proposal.ReceivedNumber.Uint64()+proposal.ValidationLoopCnt*s.config.MaxSignerCount+1 == headerNumber.Uint64() {
			//return deposit for proposal
			if _, ok := s.ProposalRefund[headerNumber.Uint64()]; !ok {
				s.ProposalRefund[headerNumber.Uint64()] = make(map[common.Address]*big.Int)
			}
			if _, ok := s.ProposalRefund[headerNumber.Uint64()][proposal.Proposer]; !ok {
				s.ProposalRefund[headerNumber.Uint64()][proposal.Proposer] = new(big.Int).Set(proposal.CurrentDeposit)
			} else {
				s.ProposalRefund[headerNumber.Uint64()][proposal.Proposer].Add(s.ProposalRefund[headerNumber.Uint64()][proposal.Proposer], proposal.CurrentDeposit)
			}

			// calculate the current stake of this proposal
			judegmentStake := big.NewInt(0)
			for _, tally := range s.Tally {
				judegmentStake.Add(judegmentStake, tally)
			}
			judegmentStake.Mul(judegmentStake, big.NewInt(2))
			judegmentStake.Div(judegmentStake, big.NewInt(3))
			// calculate declare stake
			yesDeclareStake := big.NewInt(0)
			for _, declare := range proposal.Declares {
				if declare.Decision {
					if _, ok := s.Tally[declare.Declarer]; ok {
						yesDeclareStake.Add(yesDeclareStake, s.Tally[declare.Declarer])
					}
				}
			}
			if yesDeclareStake.Cmp(judegmentStake) > 0 {
				// process add candidate
				switch proposal.ProposalType {
				case proposalTypeCandidateAdd:
					if candidateNeedPD {
						s.Candidates[proposal.TargetAddress] = candidateStateNormal
					}
				case proposalTypeCandidateRemove:
					if _, ok := s.Candidates[proposal.TargetAddress]; ok && candidateNeedPD {
						delete(s.Candidates, proposal.TargetAddress)
					}
				case proposalTypeMinerRewardDistributionModify:
					s.MinerReward = s.Proposals[hashKey].MinerRewardPerThousand

				case proposalTypeSideChainAdd:
					if _, ok := s.SCRecordMap[proposal.SCHash]; !ok {
						s.SCRecordMap[proposal.SCHash] = &SCRecord{make(map[uint64][]*SCConfirmation), 0, 0, proposal.SCBlockCountPerPeriod, proposal.SCBlockRewardPerPeriod, make(map[common.Hash]*SCRentInfo)}
					} else {
						s.SCRecordMap[proposal.SCHash].CountPerPeriod = proposal.SCBlockCountPerPeriod
						s.SCRecordMap[proposal.SCHash].RewardPerPeriod = proposal.SCBlockRewardPerPeriod
					}
				case proposalTypeSideChainRemove:
					if _, ok := s.SCRecordMap[proposal.SCHash]; ok {
						delete(s.SCRecordMap, proposal.SCHash)
					}
				case proposalTypeMinVoterBalanceModify:
					s.MinVB = new(big.Int).Mul(new(big.Int).SetUint64(s.Proposals[hashKey].MinVoterBalance), big.NewInt(1e+18))
				case proposalTypeProposalDepositModify:
					//proposalDeposit = new(big.Int).Mul(new(big.Int).SetUint64(s.Proposals[hashKey].ProposalDeposit), big.NewInt(1e+18))
				case proposalTypeRentSideChain:
					// check if buy success
					if _, ok := s.SCRecordMap[proposal.SCHash]; !ok {
						// refund the rent fee if the side chain do not exist now, (exist when proposal)
						refundSCRentFee := new(big.Int).Mul(new(big.Int).SetUint64(s.Proposals[hashKey].SCRentFee), big.NewInt(1e+18))
						s.ProposalRefund[headerNumber.Uint64()][proposal.Proposer].Add(s.ProposalRefund[headerNumber.Uint64()][proposal.Proposer], refundSCRentFee)
					} else {
						// add rent reward info to scConfirmation
						rentFee := new(big.Int).Mul(new(big.Int).SetUint64(proposal.SCRentFee), big.NewInt(1e+18))
						rentPerPeriod := new(big.Int).Div(rentFee, new(big.Int).SetUint64(proposal.SCRentLength))
						maxRewardNumber := new(big.Int).Add(headerNumber, new(big.Int).SetUint64(proposal.SCRentLength))
						s.SCRecordMap[proposal.SCHash].RentReward[proposal.Hash] = &SCRentInfo{
							rentPerPeriod,
							maxRewardNumber,
						}
						if _, ok := s.SCNoticeMap[proposal.SCHash]; !ok {
							s.SCNoticeMap[proposal.SCHash] = &CCNotice{make(map[common.Hash]GasCharging), make(map[common.Hash]NoticeCR)}
						}
						s.SCNoticeMap[proposal.SCHash].CurrentCharging[proposal.Hash] = GasCharging{proposal.TargetAddress, proposal.SCRentFee * proposal.SCRentRate, proposal.Hash}
					}
				default:
					// todo
				}
			} else {
				// reach the target header number, but not success
				switch proposal.ProposalType {
				case proposalTypeRentSideChain:
					// refund the side chain rent fee
					refundSCRentFee := new(big.Int).Mul(new(big.Int).SetUint64(s.Proposals[hashKey].SCRentFee), big.NewInt(1e+18))
					s.ProposalRefund[headerNumber.Uint64()][proposal.Proposer].Add(s.ProposalRefund[headerNumber.Uint64()][proposal.Proposer], refundSCRentFee)
				default:
					// todo

				}
			}

			// remove all proposal
			delete(s.Proposals, hashKey)
		}

	}

}

func (s *Snapshot) updateSnapshotByProposals(proposals []Proposal, headerNumber *big.Int) {
	for _, proposal := range proposals {
		proposal.ReceivedNumber = new(big.Int).Set(headerNumber)
		s.Proposals[proposal.Hash] = &proposal
	}
}

func (s *Snapshot) updateSnapshotForExpired(headerNumber *big.Int) {

	// deal the expired vote
	var expiredVotes []*Vote
	checkBalance := false
	if len(s.Voters) > maxUncheckBalanceVoteCount {
		checkBalance = true
	}

	for voterAddress, voteNumber := range s.Voters {
		// clear the vote
		if expiredVote, ok := s.Votes[voterAddress]; ok {
			if headerNumber.Uint64()-voteNumber.Uint64() > s.config.Epoch || (checkBalance && s.Votes[voterAddress].Stake.Cmp(s.MinVB) < 0) {
				expiredVotes = append(expiredVotes, expiredVote)
			}
		}
	}
	// remove expiredVotes only enough voters left
	if uint64(len(s.Voters)-len(expiredVotes)) >= s.config.MaxSignerCount {
		for _, expiredVote := range expiredVotes {
			if _, ok := s.Tally[expiredVote.Candidate]; ok {
				s.Tally[expiredVote.Candidate].Sub(s.Tally[expiredVote.Candidate], expiredVote.Stake)
				if s.Tally[expiredVote.Candidate].Cmp(big.NewInt(0)) == 0 {
					delete(s.Tally, expiredVote.Candidate)
				}
			}
			delete(s.Votes, expiredVote.Voter)
			delete(s.Voters, expiredVote.Voter)
		}
	}

	// deal the expired confirmation
	for blockNumber := range s.Confirmations {
		if headerNumber.Uint64()-blockNumber > s.config.MaxSignerCount {
			delete(s.Confirmations, blockNumber)
		}
	}

	// remove 0 stake tally
	for address, tally := range s.Tally {
		if tally.Cmp(big.NewInt(0)) <= 0 {
			if _, ok := s.SCCoinbase[address]; ok {
				delete(s.SCCoinbase, address)
			}
			delete(s.Tally, address)
		}
	}
}

func (s *Snapshot) updateSnapshotByConfirmations(confirmations []Confirmation) {
	for _, confirmation := range confirmations {
		_, ok := s.Confirmations[confirmation.BlockNumber.Uint64()]
		if !ok {
			s.Confirmations[confirmation.BlockNumber.Uint64()] = []*common.Address{}
		}
		addConfirmation := true
		for _, address := range s.Confirmations[confirmation.BlockNumber.Uint64()] {
			if confirmation.Signer.Str() == address.Str() {
				addConfirmation = false
				break
			}
		}
		if addConfirmation == true {
			var confirmSigner common.Address
			confirmSigner.Set(confirmation.Signer)
			s.Confirmations[confirmation.BlockNumber.Uint64()] = append(s.Confirmations[confirmation.BlockNumber.Uint64()], &confirmSigner)
		}
	}
}

func (s *Snapshot) updateSnapshotByVotes(votes []Vote, headerNumber *big.Int) {
	for _, vote := range votes {
		// update Votes, Tally, Voters data
		if lastVote, ok := s.Votes[vote.Voter]; ok {
			s.Tally[lastVote.Candidate].Sub(s.Tally[lastVote.Candidate], lastVote.Stake)
		}
		if _, ok := s.Tally[vote.Candidate]; ok {

			s.Tally[vote.Candidate].Add(s.Tally[vote.Candidate], vote.Stake)
		} else {
			s.Tally[vote.Candidate] = new(big.Int).Set(vote.Stake)
			if !candidateNeedPD {
				s.Candidates[vote.Candidate] = candidateStateNormal
			}
		}

		s.Votes[vote.Voter] = &Vote{vote.Voter, vote.Candidate, new(big.Int).Set(vote.Stake)}
		s.Voters[vote.Voter] = new(big.Int).Set(headerNumber)
	}
}

func (s *Snapshot) updateSnapshotByMPVotes(votes []Vote) {
	for _, txVote := range votes {

		if lastVote, ok := s.Votes[txVote.Voter]; ok {
			s.Tally[lastVote.Candidate].Sub(s.Tally[lastVote.Candidate], lastVote.Stake)
			s.Tally[lastVote.Candidate].Add(s.Tally[lastVote.Candidate], txVote.Stake)
			s.Votes[txVote.Voter] = &Vote{Voter: txVote.Voter, Candidate: lastVote.Candidate, Stake: txVote.Stake}
			// do not modify header number of snap.Voters
		}
	}
}

func (s *Snapshot) updateSnapshotForPunish(signerMissing []common.Address, headerNumber *big.Int, coinbase common.Address) {
	// set punished count to half of origin in Epoch
	/*
		if headerNumber.Uint64()%s.config.Epoch == 0 {
			for bePublished := range s.Punished {
				if count := s.Punished[bePublished] / 2; count > 0 {
					s.Punished[bePublished] = count
				} else {
					delete(s.Punished, bePublished)
				}
			}
		}
	*/
	// punish the missing signer
	for _, signerEach := range signerMissing {
		if _, ok := s.Punished[signerEach]; ok {
			// 10 times of defaultFullCredit is big enough for calculate signer order
			if s.Punished[signerEach] <= 10*defaultFullCredit {
				s.Punished[signerEach] += missingPublishCredit
			}
		} else {
			s.Punished[signerEach] = missingPublishCredit
		}
	}
	// reduce the punish of sign signer
	if _, ok := s.Punished[coinbase]; ok {

		if s.Punished[coinbase] > signRewardCredit {
			s.Punished[coinbase] -= signRewardCredit
		} else {
			delete(s.Punished, coinbase)
		}
	}
	// reduce the punish for all punished
	for signerEach := range s.Punished {
		if s.Punished[signerEach] > autoRewardCredit {
			s.Punished[signerEach] -= autoRewardCredit
		} else {
			delete(s.Punished, signerEach)
		}
	}

	// clear all punish score at the beginning of trantor block
	if s.config.IsTrantor(headerNumber) && !s.config.IsTrantor(new(big.Int).Sub(headerNumber, big.NewInt(1))) {
		s.Punished = make(map[common.Address]uint64)
	}

}

// inturn returns if a signer at a given block height is in-turn or not.
func (s *Snapshot) inturn(signer common.Address, headerTime uint64) bool {
	// if all node stop more than period of one loop
	if signersCount := len(s.Signers); signersCount > 0 {
		if loopIndex := ((headerTime - s.LoopStartTime) / s.config.Period) % uint64(signersCount); *s.Signers[loopIndex] == signer {
			return true
		}
	}
	return false

}

// check if side chain is exist (in side chain confirmation)
func (s *Snapshot) isSideChainExist(hash common.Hash) bool {
	if _, ok := s.SCRecordMap[hash]; ok {
		return true
	}
	return false
}

// check if address belong to voter
func (s *Snapshot) isVoter(address common.Address) bool {
	if _, ok := s.Voters[address]; ok {
		return true
	}
	return false
}

// check if address belong to candidate
func (s *Snapshot) isCandidate(address common.Address) bool {
	if _, ok := s.Candidates[address]; ok {
		return true
	}
	return false
}

// get last block number meet the confirm condition
func (s *Snapshot) getLastConfirmedBlockNumber(confirmations []Confirmation) *big.Int {

	cpyConfirmations := make(map[uint64][]*common.Address)
	for blockNumber, confirmers := range s.Confirmations {
		cpyConfirmations[blockNumber] = make([]*common.Address, len(confirmers))
		copy(cpyConfirmations[blockNumber], confirmers)
	}
	// update confirmation into snapshot
	for _, confirmation := range confirmations {
		_, ok := cpyConfirmations[confirmation.BlockNumber.Uint64()]
		if !ok {
			cpyConfirmations[confirmation.BlockNumber.Uint64()] = []*common.Address{}
		}
		addConfirmation := true
		for _, address := range cpyConfirmations[confirmation.BlockNumber.Uint64()] {
			if confirmation.Signer.Str() == address.Str() {
				addConfirmation = false
				break
			}
		}
		if addConfirmation == true {
			var confirmSigner common.Address
			confirmSigner.Set(confirmation.Signer)
			cpyConfirmations[confirmation.BlockNumber.Uint64()] = append(cpyConfirmations[confirmation.BlockNumber.Uint64()], &confirmSigner)
		}
	}

	i := s.Number
	for ; i > s.Number-s.config.MaxSignerCount*2/3+1; i-- {
		if confirmers, ok := cpyConfirmations[i]; ok {
			if len(confirmers) > int(s.config.MaxSignerCount*2/3) {
				return big.NewInt(int64(i))
			}
		}
	}
	return big.NewInt(int64(i))
}

func (s *Snapshot) calculateProposalRefund() map[common.Address]*big.Int {

	if refund, ok := s.ProposalRefund[s.Number-proposalRefundDelayLoopCount*s.config.MaxSignerCount]; ok {
		return refund
	}
	return make(map[common.Address]*big.Int)
}

func (s *Snapshot) calculateVoteReward(coinbase common.Address, votersReward *big.Int) (map[common.Address]*big.Int, error) {
	rewards := make(map[common.Address]*big.Int)
	allStake := big.NewInt(0)

	for voter, vote := range s.Votes {
		if vote.Candidate.Str() == coinbase.Str() && s.Voters[vote.Voter].Uint64() < s.Number-s.config.MaxSignerCount {
			allStake.Add(allStake, vote.Stake)
			rewards[voter] = new(big.Int).Set(vote.Stake)
		}
	}

	if allStake.Cmp(big.NewInt(0)) <= 0 && len(rewards) > 0 {
		return nil, errAllStakeMissing
	}
	for _, stake := range rewards {
		stake.Mul(stake, votersReward)
		stake.Div(stake, allStake)
	}
	return rewards, nil
}

func (s *Snapshot) calculateGasCharging() map[common.Address]*big.Int {
	gasCharge := make(map[common.Address]*big.Int)
	for hash, noticeRecord := range s.LocalNotice.ConfirmReceived {
		if noticeRecord.Success && s.Number == noticeRecord.Number+scGasChargingDelayLoopCount*s.config.MaxSignerCount {
			if charge, ok := s.LocalNotice.CurrentCharging[hash]; ok {
				if _, ok := gasCharge[charge.Target]; !ok {
					gasCharge[charge.Target] = new(big.Int).Mul(big.NewInt(1e+18), new(big.Int).SetUint64(charge.Volume))
				} else {
					gasCharge[charge.Target].Add(gasCharge[charge.Target], new(big.Int).Mul(big.NewInt(1e+18), new(big.Int).SetUint64(charge.Volume)))
				}
			}
		}
	}
	return gasCharge
}

func (s *Snapshot) calculateSCReward(minerReward *big.Int) (map[common.Address]*big.Int, *big.Int) {

	minerLeft := new(big.Int).Set(minerReward)
	scRewardAll := new(big.Int).Set(minerReward)
	scRewards := make(map[common.Address]*big.Int)

	// need to deal with sum of record.RewardPerPeriod for all side chain is larger than 100% situation
	scRewardMilliSum := uint64(0)
	for _, record := range s.SCRecordMap {
		scRewardMilliSum += record.RewardPerPeriod
	}

	if scRewardMilliSum > 0 && scRewardMilliSum < 1000 {
		scRewardAll.Mul(scRewardAll, new(big.Int).SetUint64(scRewardMilliSum))
		scRewardAll.Div(scRewardAll, big.NewInt(1000))
		minerLeft.Sub(minerLeft, scRewardAll)
		scRewardMilliSum = 1000
	} else if scRewardMilliSum >= 1000 {
		minerLeft.SetUint64(0)
	} else {
		scRewardAll.SetUint64(0)
		scRewardMilliSum = 1000
	}

	for scHash := range s.SCRewardMap {
		// check reward for the block number is exist
		if reward, ok := s.SCRewardMap[scHash].SCBlockRewardMap[s.Number-scRewardDelayLoopCount*s.config.MaxSignerCount]; ok {
			// check confirm is exist, to get countPerPeriod and rewardPerPeriod
			if confirmation, ok := s.SCRecordMap[scHash]; ok {
				// calculate the rent still not reach on this side chain
				scRentSumPerPeriod := big.NewInt(0)
				for _, rent := range confirmation.RentReward {
					if rent.MaxRewardNumber.Uint64() >= s.Number-scRewardDelayLoopCount*s.config.MaxSignerCount {
						scRentSumPerPeriod.Add(scRentSumPerPeriod, rent.RentPerPeriod)
					}
				}

				// calculate the side chain reward base on score/100 and record.RewardPerPeriod
				for addr, score := range reward.RewardScoreMap {
					singleReward := new(big.Int).Set(scRewardAll)
					singleReward.Mul(singleReward, new(big.Int).SetUint64(confirmation.RewardPerPeriod))
					singleReward.Div(singleReward, new(big.Int).SetUint64(scRewardMilliSum))
					singleReward.Add(singleReward, scRentSumPerPeriod)
					singleReward.Mul(singleReward, new(big.Int).SetUint64(score))
					singleReward.Div(singleReward, new(big.Int).SetUint64(100)) // for score/100

					if _, ok := scRewards[addr]; ok {
						scRewards[addr].Add(scRewards[addr], singleReward)
					} else {
						scRewards[addr] = singleReward
					}
				}
			}
		}
	}
	return scRewards, minerLeft

}
