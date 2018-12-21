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
	"strconv"
	"strings"
	"time"
)

const (
	defaultFullCredit               = 1000 // no punished
	missingPublishCredit            = 100  // punished for missing one block seal
	signRewardCredit                = 10   // seal one block
	autoRewardCredit                = 1    // credit auto recover for each block
	minCalSignerQueueCredit         = 300  // when calculate the signerQueue
	defaultOfficialMaxSignerCount   = 21   // official max signer count
	defaultOfficialFirstLevelCount  = 10   // official first level , 100% in signer queue
	defaultOfficialSecondLevelCount = 20   // official second level, 60% in signer queue
	defaultOfficialThirdLevelCount  = 30   // official third level, 40% in signer queue
	// the credit of one signer is at least minCalSignerQueueCredit
	candidateStateNormal = 1
	candidateMaxLen      = 500 // if candidateNeedPD is false and candidate is more than candidateMaxLen, then minimum tickets candidates will be remove in each LCRS*loop
	// reward for side chain
	scRewardDelayLoopCount   = 2                          //
	scRewardExpiredLoopCount = scRewardDelayLoopCount + 4 //
	scMaxCountPerPeriod      = 6
)

var (
	errIncorrectTallyCount = errors.New("incorrect tally count")
)

// SCCurrentBlockReward is base on scMaxCountPerPeriod = 6
var SCCurrentBlockReward = map[uint64]map[uint64]uint64{1: {1: 100},
	2: {1: 30, 2: 70},
	3: {1: 15, 2: 30, 3: 55},
	4: {1: 5, 2: 15, 3: 30, 4: 50},
	5: {1: 5, 2: 10, 3: 15, 4: 25, 5: 45},
	6: {1: 1, 2: 4, 3: 10, 4: 15, 5: 25, 6: 45}}

// SCReward
type SCReward = map[uint64]map[common.Address]uint64 //sum(this value) in one period == 100

// SCRecord is the state record for side chain
type SCRecord struct {
	Record              map[uint64][]*SCConfirmation `json:"record"`              // Confirmation Record of one side chain
	LastConfirmedNumber uint64                       `json:"lastConfirmedNumber"` // Last confirmed header number of one side chain
	MaxHeaderNumber     uint64                       `json:"maxHeaderNumber"`     // max header number of one side chain
	CountPerPeriod      uint64                       `json:"countPerPeriod"`      // block sealed per period on this side chain
	RewardPerPeriod     uint64                       `json:"rewardPerPeriod"`     // full reward per period
}

// Snapshot is the state of the authorization voting at a given point in time.
type Snapshot struct {
	config   *params.AlienConfig // Consensus engine parameters to fine tune behavior
	sigcache *lru.ARCCache       // Cache of recent block signatures to speed up ecrecover
	LCRS     uint64              // Loop count to recreate signers from top tally

	Period          uint64                       `json:"period"`          // Period of seal each block
	Number          uint64                       `json:"number"`          // Block number where the snapshot was created
	ConfirmedNumber uint64                       `json:"confirmedNumber"` // Block number confirmed when the snapshot was created
	Hash            common.Hash                  `json:"hash"`            // Block hash where the snapshot was created
	HistoryHash     []common.Hash                `json:"historyHash"`     // Block hash list for two recent loop
	Signers         []*common.Address            `json:"signers"`         // Signers queue in current header
	Votes           map[common.Address]*Vote     `json:"votes"`           // All validate votes from genesis block
	Tally           map[common.Address]*big.Int  `json:"tally"`           // Stake for each candidate address
	Voters          map[common.Address]*big.Int  `json:"voters"`          // Block number for each voter address
	Candidates      map[common.Address]uint64    `json:"candidates"`      // Candidates for Signers (0- adding procedure 1- normal 2- removing procedure)
	Punished        map[common.Address]uint64    `json:"punished"`        // The signer be punished count cause of missing seal
	Confirmations   map[uint64][]*common.Address `json:"confirms"`        // The signer confirm given block number
	Proposals       map[common.Hash]*Proposal    `json:"proposals"`       // The Proposals going or success (failed proposal will be removed)
	HeaderTime      uint64                       `json:"headerTime"`      // Time of the current header
	LoopStartTime   uint64                       `json:"loopStartTime"`   // Start Time of the current loop

	SCCoinbase     map[common.Address]map[common.Hash]common.Address `json:"sideChainCoinbase"`     // Coinbase of side chain setting
	SCConfirmation map[common.Hash]*SCRecord                         `json:"sideChainConfirmation"` // Confirmation of side chain setting
	SCAllReward    map[common.Hash]SCReward                          `json:"sideChainReward"`       // Side Chain Reward
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
		SCConfirmation:  make(map[common.Hash]*SCRecord),
		SCAllReward:     make(map[common.Hash]SCReward),
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

	for i := 0; i < int(config.MaxSignerCount); i++ {
		snap.Signers = append(snap.Signers, &config.SelfVoteSigners[i%len(config.SelfVoteSigners)])
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
		SCConfirmation: make(map[common.Hash]*SCRecord),
		SCAllReward:    make(map[common.Hash]SCReward),
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
	for hash, scc := range s.SCConfirmation {
		cpy.SCConfirmation[hash] = &SCRecord{
			LastConfirmedNumber: scc.LastConfirmedNumber,
			MaxHeaderNumber:     scc.MaxHeaderNumber,
			CountPerPeriod:      scc.CountPerPeriod,
			RewardPerPeriod:     scc.RewardPerPeriod,
			Record:              make(map[uint64][]*SCConfirmation),
		}
		for number, scConfirmation := range scc.Record {
			cpy.SCConfirmation[hash].Record[number] = make([]*SCConfirmation, len(scConfirmation))
			copy(cpy.SCConfirmation[hash].Record[number], scConfirmation)
		}
	}

	for hash, sca := range s.SCAllReward {
		cpy.SCAllReward[hash] = make(map[uint64]map[common.Address]uint64)
		for number, reward := range sca {
			cpy.SCAllReward[hash][number] = make(map[common.Address]uint64)
			for addr, count := range reward {
				cpy.SCAllReward[hash][number][addr] = count
			}
		}
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
		if coinbase.Str() != header.Coinbase.Str() {
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
		if snap.Period == 0 && snap.config.IsTrantor(header.Number) {
			snap.Period = snap.config.Period
		}

		// deal setcoinbase for side chain
		snap.updateSnapshotBySetSCCoinbase(headerExtra.SideChainSetCoinbases)

		// deal confirmation for side chain
		snap.updateSnapshotBySCConfirm(headerExtra.SideChainConfirmations, header.Number)

		// calculate proposal result
		snap.calculateProposalResult(header.Number)

		// check the len of candidate if not candidateNeedPD
		if !candidateNeedPD && (snap.Number+1)%(snap.config.MaxSignerCount*snap.LCRS) == 0 && len(snap.Candidates) > candidateMaxLen {
			snap.removeExtraCandidate()
		}

	}
	snap.Number += uint64(len(headers))
	snap.Hash = headers[len(headers)-1].Hash()

	snap.updateSnapshotForExpired()
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

func (s *Snapshot) isSideChainCoinbase(sc common.Hash, address common.Address) bool {
	// check is side chain coinbase
	// is use the coinbase of main chain as coinbase of side chain , return false
	// the main chain cloud seal block, but not recommend for send confirm tx usually fail
	for _, coinbaseMap := range s.SCCoinbase {
		if coinbase, ok := coinbaseMap[sc]; ok && coinbase == address {
			return true
		}
	}
	return false
}

func (s *Snapshot) updateSnapshotBySCConfirm(scConfirmations []SCConfirmation, headerNumber *big.Int) {
	// todo ,if diff side chain coinbase send confirm for the same side chain , same number ...
	for _, scc := range scConfirmations {
		// new confirmation header number must larger than last confirmed number of this side chain
		if s.isSideChainCoinbase(scc.Hash, scc.Coinbase) {
			if _, ok := s.SCConfirmation[scc.Hash]; ok && scc.Number > s.SCConfirmation[scc.Hash].LastConfirmedNumber {
				s.SCConfirmation[scc.Hash].Record[scc.Number] = append(s.SCConfirmation[scc.Hash].Record[scc.Number], scc.copy())
				if scc.Number > s.SCConfirmation[scc.Hash].MaxHeaderNumber {
					s.SCConfirmation[scc.Hash].MaxHeaderNumber = scc.Number
				}
			}
		}
	}
	// calculate the side chain reward in each loop
	if (headerNumber.Uint64()+1)%s.config.MaxSignerCount == 0 {
		s.updateSCConfirmation(headerNumber)
	}
}

func (s *Snapshot) calculateConfirmedNumber(record *SCRecord, minConfirmedSignerCount int) (uint64, map[uint64]common.Address) {
	// todo : add params scHash, so can check if the address in SCRecord is belong to this side chain

	confirmedNumber := record.LastConfirmedNumber
	confirmedRecordMap := make(map[string]map[common.Address]bool)
	confirmedCoinbase := make(map[uint64]common.Address)
	sep := ":"
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
							headerNum, err := strconv.Atoi(scConfirm.LoopInfo[len(scConfirm.LoopInfo)-2])
							if err == nil && uint64(headerNum) > confirmedNumber {
								confirmedNumber = uint64(headerNum)
							}
						}
					}
				}
			}
		}
	}

	for info, count := range confirmedRecordMap {
		if len(count) >= minConfirmedSignerCount {
			infos := strings.Split(info, sep)
			for i := 0; i+1 < len(infos); i += 2 {
				number, err := strconv.Atoi(infos[i])
				if err != nil {
					continue
				}
				confirmedCoinbase[uint64(number)] = common.HexToAddress(infos[i+1])
			}
		}
	}

	return confirmedNumber, confirmedCoinbase
}

func (s *Snapshot) calcuateCurrentBlockReward(currentCount uint64, periodCount uint64) uint64 {
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
	for scHash, record := range s.SCConfirmation {
		if _, ok := s.SCAllReward[scHash]; !ok {
			s.SCAllReward[scHash] = make(map[uint64]map[common.Address]uint64)
		}
		if _, ok := s.SCAllReward[scHash][headerNumber.Uint64()]; !ok {
			s.SCAllReward[scHash][headerNumber.Uint64()] = make(map[common.Address]uint64)
		}
		confirmedNumber, confirmedCoinbase := s.calculateConfirmedNumber(record, minConfirmedSignerCount)
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

					if _, ok := s.SCAllReward[scHash][headerNumber.Uint64()][scCoinbase]; !ok {
						s.SCAllReward[scHash][headerNumber.Uint64()][scCoinbase] = s.calcuateCurrentBlockReward(currentSCCoinbaseCount, record.CountPerPeriod)
					} else {
						s.SCAllReward[scHash][headerNumber.Uint64()][scCoinbase] += s.calcuateCurrentBlockReward(currentSCCoinbaseCount, record.CountPerPeriod)
					}

					// update lastSCCoinbase
					lastSCCoinbase = scCoinbase
				}
			}

			for i := record.LastConfirmedNumber + 1; i <= confirmedNumber; i++ {
				if _, ok := s.SCConfirmation[scHash].Record[i]; ok {
					delete(s.SCConfirmation[scHash].Record, i)
				}
			}
			s.SCConfirmation[scHash].LastConfirmedNumber = confirmedNumber
		}
		// clear empty block number for side chain
		if len(s.SCAllReward[scHash][headerNumber.Uint64()]) == 0 {
			delete(s.SCAllReward[scHash], headerNumber.Uint64())
		}
	}

	for scHash, _ := range s.SCAllReward {
		// clear expired side chain reward record
		for number, _ := range s.SCAllReward[scHash] {
			if number < headerNumber.Uint64()-scRewardExpiredLoopCount*s.config.MaxSignerCount {
				delete(s.SCAllReward[scHash], number)
			}
		}
		// clear this side chain if reward is empty
		if len(s.SCAllReward[scHash]) == 0 {
			delete(s.SCAllReward, scHash)
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

	for hashKey, proposal := range s.Proposals {
		// the result will be calculate at receiverdNumber + vlcnt + 1
		if proposal.ReceivedNumber.Uint64()+proposal.ValidationLoopCnt*s.config.MaxSignerCount+1 == headerNumber.Uint64() {
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
						s.Candidates[proposal.Candidate] = candidateStateNormal
					}
				case proposalTypeCandidateRemove:
					if _, ok := s.Candidates[proposal.Candidate]; ok && candidateNeedPD {
						delete(s.Candidates, proposal.Candidate)
					}
				case proposalTypeMinerRewardDistributionModify:
					minerRewardPerThousand = s.Proposals[hashKey].MinerRewardPerThousand

				case proposalTypeSideChainAdd:
					if _, ok := s.SCConfirmation[proposal.SCHash]; !ok {
						s.SCConfirmation[proposal.SCHash] = &SCRecord{make(map[uint64][]*SCConfirmation), 0, 0, proposal.SCBlockCountPerPeriod, proposal.SCBlockRewardPerPeriod}
					}
				case proposalTypeSideChainRemove:
					if _, ok := s.SCConfirmation[proposal.SCHash]; ok {
						delete(s.SCConfirmation, proposal.SCHash)
					}
				}
			} else {
				// reach the target header number, but not success
				// remove the fail proposal
				delete(s.Proposals, hashKey)
			}

		}

	}

}

func (s *Snapshot) updateSnapshotByProposals(proposals []Proposal, headerNumber *big.Int) {
	for _, proposal := range proposals {
		proposal.ReceivedNumber = new(big.Int).Set(headerNumber)
		s.Proposals[proposal.Hash] = &proposal
	}
}

func (s *Snapshot) updateSnapshotForExpired() {

	// deal the expired vote
	var expiredVotes []*Vote
	for voterAddress, voteNumber := range s.Voters {
		if s.Number-voteNumber.Uint64() > s.config.Epoch {
			// clear the vote
			if expiredVote, ok := s.Votes[voterAddress]; ok {
				expiredVotes = append(expiredVotes, expiredVote)
			}
		}
	}
	// remove expiredVotes only enough voters left
	if uint64(len(s.Voters)-len(expiredVotes)) >= s.config.MaxSignerCount {
		for _, expiredVote := range expiredVotes {
			s.Tally[expiredVote.Candidate].Sub(s.Tally[expiredVote.Candidate], expiredVote.Stake)
			if s.Tally[expiredVote.Candidate].Cmp(big.NewInt(0)) == 0 {
				delete(s.Tally, expiredVote.Candidate)
			}
			delete(s.Votes, expiredVote.Voter)
			delete(s.Voters, expiredVote.Voter)
		}
	}

	// deal the expired confirmation
	for blockNumber := range s.Confirmations {
		if s.Number-blockNumber > s.config.MaxSignerCount {
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
			s.Tally[vote.Candidate] = vote.Stake
			if !candidateNeedPD {
				s.Candidates[vote.Candidate] = candidateStateNormal
			}
		}

		s.Votes[vote.Voter] = &Vote{vote.Voter, vote.Candidate, vote.Stake}
		s.Voters[vote.Voter] = headerNumber
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
	for _, signerMissing := range signerMissing {
		if _, ok := s.Punished[signerMissing]; ok {
			s.Punished[signerMissing] += missingPublishCredit
		} else {
			s.Punished[signerMissing] = missingPublishCredit
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
}

// inturn returns if a signer at a given block height is in-turn or not.
func (s *Snapshot) inturn(signer common.Address, headerTime uint64) bool {

	// if all node stop more than period of one loop
	loopIndex := int((headerTime-s.LoopStartTime)/s.config.Period) % len(s.Signers)
	if loopIndex >= len(s.Signers) {
		return false
	} else if *s.Signers[loopIndex] != signer {
		return false

	}
	return true
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

func (s *Snapshot) calculateVoteReward(coinbase common.Address, votersReward *big.Int) map[common.Address]*big.Int {
	rewards := make(map[common.Address]*big.Int)
	allStake := big.NewInt(0)
	for voter, vote := range s.Votes {
		if vote.Candidate.Str() == coinbase.Str() {
			allStake.Add(allStake, vote.Stake)
			rewards[voter] = new(big.Int).Set(vote.Stake)
		}
	}
	for _, stake := range rewards {
		stake.Mul(stake, votersReward)
		stake.Div(stake, allStake)
	}

	return rewards
}

func (s *Snapshot) calculateSCReward() map[common.Address]*big.Int {
	// rewards for side chain
	if s.config.IsTrantor(new(big.Int).SetUint64(s.Number)) {
		rewards := make(map[common.Address]*big.Int)
		for scHash, scReward := range s.SCAllReward {
			// check reward for the block number is exist
			if reward, ok := scReward[s.Number-scRewardDelayLoopCount*s.config.MaxSignerCount]; ok {
				// check confirm is exist, to get countPerPeriod and rewardPerPeriod
				if confirmation, ok := s.SCConfirmation[scHash]; ok {

					// todo : need calculate the side chain reward base on RewardPerPeriod(/100) and record.RewardPerPeriod
					// todo : need to deal with sum of record.RewardPerPeriod for all side chain is larger than 100% situation
					for addr, scre := range reward {
						if _, ok := rewards[addr]; ok {
							rewards[addr].Add(rewards[addr], new(big.Int).SetUint64(scre*confirmation.RewardPerPeriod))
						} else {
							rewards[addr] = new(big.Int).SetUint64(scre * confirmation.RewardPerPeriod)
						}
					}
				}

			}

		}
		return rewards

	}
	return nil
}
