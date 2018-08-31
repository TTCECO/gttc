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
	"bytes"
	"encoding/json"
	"math/big"
	"sort"
	"time"

	"github.com/TTCECO/gttc/common"
	"github.com/TTCECO/gttc/core/types"
	"github.com/TTCECO/gttc/ethdb"
	"github.com/TTCECO/gttc/params"
	"github.com/TTCECO/gttc/rlp"
	"github.com/hashicorp/golang-lru"
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
	candidateAdding   = 0
	candidateNormal   = 1
	candidateRemoving = 2
)

// Snapshot is the state of the authorization voting at a given point in time.
type Snapshot struct {
	config   *params.AlienConfig // Consensus engine parameters to fine tune behavior
	sigcache *lru.ARCCache       // Cache of recent block signatures to speed up ecrecover
	LCRS     uint64              // Loop count to recreate signers from top tally

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
}

// newSnapshot creates a new snapshot with the specified startup parameters. only ever use if for
// the genesis block.
func newSnapshot(config *params.AlienConfig, sigcache *lru.ARCCache, hash common.Hash, votes []*Vote, lcrs uint64) *Snapshot {

	snap := &Snapshot{
		config:          config,
		sigcache:        sigcache,
		LCRS:            lcrs,
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
		snap.Candidates[vote.Voter] = candidateNormal
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

		HeaderTime:    s.HeaderTime,
		LoopStartTime: s.LoopStartTime,
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
		cpy.Tally[candidate] = tally
	}
	for voter, number := range s.Voters {
		cpy.Voters[voter] = number
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
		_, err := ecrecover(header, s.sigcache)
		if err != nil {
			return nil, err
		}

		headerExtra := HeaderExtra{}
		err = rlp.DecodeBytes(header.Extra[extraVanity:len(header.Extra)-extraSeal], &headerExtra)
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

		//
		if len(snap.HistoryHash) >= int(s.config.MaxSignerCount)*2 {
			snap.HistoryHash = snap.HistoryHash[:int(s.config.MaxSignerCount)*2-1]
		}
		snap.HistoryHash = append(snap.HistoryHash, header.Hash())

		// deal the new confirmation in this block
		for _, confirmation := range headerExtra.CurrentBlockConfirmations {
			_, ok := snap.Confirmations[confirmation.BlockNumber.Uint64()]
			if !ok {
				snap.Confirmations[confirmation.BlockNumber.Uint64()] = []*common.Address{}
			}
			addConfirmation := true
			for _, address := range snap.Confirmations[confirmation.BlockNumber.Uint64()] {
				if confirmation.Signer.Str() == address.Str() {
					addConfirmation = false
					break
				}
			}
			if addConfirmation == true {
				var confirmSigner common.Address
				confirmSigner.Set(confirmation.Signer)
				snap.Confirmations[confirmation.BlockNumber.Uint64()] = append(snap.Confirmations[confirmation.BlockNumber.Uint64()], &confirmSigner)
			}
		}

		// deal the new vote from voter
		for _, vote := range headerExtra.CurrentBlockVotes {
			// update Votes, Tally, Voters data
			if lastVote, ok := snap.Votes[vote.Voter]; ok {
				snap.Tally[lastVote.Candidate].Sub(snap.Tally[lastVote.Candidate], lastVote.Stake)
			}
			if _, ok := snap.Tally[vote.Candidate]; ok {

				snap.Tally[vote.Candidate].Add(snap.Tally[vote.Candidate], vote.Stake)
			} else {
				snap.Tally[vote.Candidate] = vote.Stake
			}

			snap.Votes[vote.Voter] = &Vote{vote.Voter, vote.Candidate, vote.Stake}
			snap.Voters[vote.Voter] = header.Number
		}
		// deal the voter which balance modified
		for _, txVote := range headerExtra.ModifyPredecessorVotes {

			if lastVote, ok := snap.Votes[txVote.Voter]; ok {
				snap.Tally[lastVote.Candidate].Sub(snap.Tally[lastVote.Candidate], lastVote.Stake)
				snap.Tally[lastVote.Candidate].Add(snap.Tally[lastVote.Candidate], txVote.Stake)
				snap.Votes[txVote.Voter] = &Vote{Voter: txVote.Voter, Candidate: lastVote.Candidate, Stake: txVote.Stake}
				// do not modify header number of snap.Voters
			}
		}
		// set punished count to half of origin in Epoch
		if header.Number.Uint64()%snap.config.Epoch == 0 {
			for bePublished := range snap.Punished {
				if count := snap.Punished[bePublished] / 2; count > 0 {
					snap.Punished[bePublished] = count
				} else {
					delete(snap.Punished, bePublished)
				}
			}
		}
		// punish the missing signer
		for _, signerMissing := range headerExtra.SignerMissing {
			if _, ok := snap.Punished[signerMissing]; ok {
				snap.Punished[signerMissing] += missingPublishCredit
			} else {
				snap.Punished[signerMissing] = missingPublishCredit
			}
		}
		// reduce the punish of sign signer
		if _, ok := snap.Punished[header.Coinbase]; ok {

			if snap.Punished[header.Coinbase] > signRewardCredit {
				snap.Punished[header.Coinbase] -= signRewardCredit
			} else {
				delete(snap.Punished, header.Coinbase)
			}
		}
		// reduce the punish for all punished
		for signerEach := range snap.Punished {
			snap.Punished[signerEach] -= autoRewardCredit
		}
		// deal proposals
		for _, proposal := range headerExtra.CurrentBlockProposals {
			snap.Proposals[proposal.Hash] = &proposal
		}
		// deal declares
		for _, declare := range headerExtra.CurrentBlockDeclares {
			if proposal, ok := snap.Proposals[declare.ProposalHash]; ok {
				// todo : check the proposal enable status and valid block number
				alreadyDeclare := false
				for _, v := range proposal.Declares {
					if v.Declarer.Hex() == declare.Declarer.Hex() {
						// this declarer already declare for this proposal
						alreadyDeclare = true
						break
					}
				}
				if !alreadyDeclare {
					snap.Proposals[declare.ProposalHash].Declares = append(snap.Proposals[declare.ProposalHash].Declares, &declare)
				}
				// todo: check if this proposal is enabled
			}
		}

	}
	snap.Number += uint64(len(headers))
	snap.Hash = headers[len(headers)-1].Hash()

	// deal the expired vote
	for voterAddress, voteNumber := range snap.Voters {
		if len(snap.Voters) <= int(s.config.MaxSignerCount) || len(snap.Tally) <= int(s.config.MaxSignerCount) {
			break
		}
		if snap.Number-voteNumber.Uint64() > s.config.Epoch {
			// clear the vote
			if expiredVote, ok := snap.Votes[voterAddress]; ok {
				snap.Tally[expiredVote.Candidate].Sub(snap.Tally[expiredVote.Candidate], expiredVote.Stake)
				if snap.Tally[expiredVote.Candidate].Cmp(big.NewInt(0)) == 0 {
					delete(snap.Tally, expiredVote.Candidate)
				}
				delete(snap.Votes, expiredVote.Voter)
				delete(snap.Voters, expiredVote.Voter)
			}
		}
	}
	// deal the expired confirmation
	for blockNumber := range snap.Confirmations {
		if snap.Number-blockNumber > snap.config.MaxSignerCount {
			delete(snap.Confirmations, blockNumber)
		}
	}

	// remove 0 stake tally
	for address, tally := range snap.Tally {
		if tally.Cmp(big.NewInt(0)) <= 0 {
			delete(snap.Tally, address)
		}
	}

	return snap, nil
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

type TallyItem struct {
	addr  common.Address
	stake *big.Int
}
type TallySlice []TallyItem

func (s TallySlice) Len() int      { return len(s) }
func (s TallySlice) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s TallySlice) Less(i, j int) bool {
	//we need sort reverse, so ...
	isLess := s[i].stake.Cmp(s[j].stake)
	if isLess > 0 {
		return true

	} else if isLess < 0 {
		return false
	}
	// if the stake equal
	return bytes.Compare(s[i].addr.Bytes(), s[j].addr.Bytes()) > 0
}

type SignerItem struct {
	addr common.Address
	hash common.Hash
}
type SignerSlice []SignerItem

func (s SignerSlice) Len() int      { return len(s) }
func (s SignerSlice) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s SignerSlice) Less(i, j int) bool {
	return bytes.Compare(s[i].hash.Bytes(), s[j].hash.Bytes()) > 0
}

// verify the SignerQueue base on block hash
func (s *Snapshot) verifySignerQueue(signerQueue []common.Address) error {

	if len(signerQueue) > int(s.config.MaxSignerCount) {
		return errInvalidSignerQueue
	}
	sq, err := s.createSignerQueue()
	if err != nil {
		return err
	}
	if len(sq) == 0 || len(sq) != len(signerQueue) {
		return errInvalidSignerQueue
	}
	for i, signer := range signerQueue {
		if signer != sq[i] {
			return errInvalidSignerQueue
		}
	}

	return nil
}

func (s *Snapshot) createSignerQueue() ([]common.Address, error) {

	if (s.Number+1)%s.config.MaxSignerCount != 0 || s.Hash != s.HistoryHash[len(s.HistoryHash)-1] {
		return nil, errCreateSignerQueueNotAllowed
	}

	var signerSlice SignerSlice
	var topStakeAddress []common.Address

	if (s.Number+1)%(s.config.MaxSignerCount*s.LCRS) == 0 {
		// before recalculate the signers, clear the candidate is not in snap.Candidates

		// only recalculate signers from to tally per 10 loop,
		// other loop end just reset the order of signers by block hash (nearly random)
		var tallySlice TallySlice
		for address, stake := range s.Tally {
			if s.isCandidate(address) {
				if _, ok := s.Punished[address]; ok {
					var creditWeight uint64
					if s.Punished[address] > defaultFullCredit-minCalSignerQueueCredit {
						creditWeight = minCalSignerQueueCredit
					} else {
						creditWeight = defaultFullCredit - s.Punished[address]
					}
					tallySlice = append(tallySlice, TallyItem{address, new(big.Int).Mul(stake, big.NewInt(int64(creditWeight)))})
				} else {
					tallySlice = append(tallySlice, TallyItem{address, new(big.Int).Mul(stake, big.NewInt(defaultFullCredit))})
				}
			}
		}

		sort.Sort(TallySlice(tallySlice))
		queueLength := int(s.config.MaxSignerCount)
		if queueLength > len(tallySlice) {
			queueLength = len(tallySlice)
		}

		if queueLength == defaultOfficialMaxSignerCount && len(tallySlice) > defaultOfficialThirdLevelCount {
			for i, tallyItem := range tallySlice[:defaultOfficialFirstLevelCount] {
				signerSlice = append(signerSlice, SignerItem{tallyItem.addr, s.HistoryHash[len(s.HistoryHash)-1-i]})
			}
			var signerSecondLevelSlice, signerThirdLevelSlice, signerLastLevelSlice SignerSlice
			// 60%
			for i, tallyItem := range tallySlice[defaultOfficialFirstLevelCount:defaultOfficialSecondLevelCount] {
				signerSecondLevelSlice = append(signerSecondLevelSlice, SignerItem{tallyItem.addr, s.HistoryHash[len(s.HistoryHash)-1-i]})
			}
			sort.Sort(SignerSlice(signerSecondLevelSlice))
			signerSlice = append(signerSlice, signerSecondLevelSlice[:6]...)
			// 40%
			for i, tallyItem := range tallySlice[defaultOfficialSecondLevelCount:defaultOfficialThirdLevelCount] {
				signerThirdLevelSlice = append(signerThirdLevelSlice, SignerItem{tallyItem.addr, s.HistoryHash[len(s.HistoryHash)-1-i]})
			}
			sort.Sort(SignerSlice(signerThirdLevelSlice))
			signerSlice = append(signerSlice, signerThirdLevelSlice[:4]...)
			// choose 1 from last
			for i, tallyItem := range tallySlice[defaultOfficialThirdLevelCount:] {
				signerLastLevelSlice = append(signerLastLevelSlice, SignerItem{tallyItem.addr, s.HistoryHash[len(s.HistoryHash)-1-i]})
			}
			sort.Sort(SignerSlice(signerLastLevelSlice))
			signerSlice = append(signerSlice, signerLastLevelSlice[0])

		} else {
			for i, tallyItem := range tallySlice[:queueLength] {
				signerSlice = append(signerSlice, SignerItem{tallyItem.addr, s.HistoryHash[len(s.HistoryHash)-1-i]})
			}

		}

	} else {
		for i, signer := range s.Signers {
			signerSlice = append(signerSlice, SignerItem{*signer, s.HistoryHash[len(s.HistoryHash)-1-i]})
		}
	}

	sort.Sort(SignerSlice(signerSlice))
	// Set the top candidates in random order base on block hash
	for i := 0; i < int(s.config.MaxSignerCount); i++ {
		topStakeAddress = append(topStakeAddress, signerSlice[i%len(signerSlice)].addr)
	}

	return topStakeAddress, nil

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
	if state, ok := s.Candidates[address]; ok {
		// in adding procedure, the candidate not being valid by enough signer (delegate stake accurately)
		if state != candidateAdding {
			return true
		}
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

func (s *Snapshot) calculateReward(coinbase common.Address, votersReward *big.Int) map[common.Address]*big.Int {

	rewards := make(map[common.Address]*big.Int)
	allStake := big.NewInt(0)
	for voter, vote := range s.Votes {
		if vote.Candidate.Hex() == coinbase.Hex() {
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
