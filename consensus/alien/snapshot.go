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

	"sort"
	"math/big"
	"math/rand"
	"encoding/json"

	"github.com/TTCECO/gttc/common"
	"github.com/TTCECO/gttc/core/types"
	"github.com/TTCECO/gttc/ethdb"
	"github.com/TTCECO/gttc/params"
	"github.com/hashicorp/golang-lru"
	"github.com/TTCECO/gttc/rlp"

)


// Snapshot is the state of the authorization voting at a given point in time.
type Snapshot struct {
	config   *params.AlienConfig // Consensus engine parameters to fine tune behavior
	sigcache *lru.ARCCache        // Cache of recent block signatures to speed up ecrecover

	Number  uint64                      `json:"number"`  // Block number where the snapshot was created
	Hash    common.Hash                 `json:"hash"`    // Block hash where the snapshot was created

	Signers map[int] common.Address 	`json:"signers"`	// Signers queue in current header
															// The signer validate should judge by last snapshot
	Votes map[common.Address] *Vote		`json:"votes"`		// All validate votes from genesis block
	Tally map[common.Address] *big.Int	`json:"tally"`		// Stake for each candidate address
	Voters map[common.Address] *big.Int  `json:"voters"`		// block number for each voter address

	HeaderTime uint64		`json:"headerTime"`				// Time of the current header
	LoopStartTime uint64  	`json:"loopStartTime"`			// Start Time of the current loop

}

// newSnapshot creates a new snapshot with the specified startup parameters. only ever use if for
// the genesis block.
func newSnapshot(config *params.AlienConfig, sigcache *lru.ARCCache,  hash common.Hash, votes []*Vote) *Snapshot {
	snap := &Snapshot{
		config:   config,
		sigcache: sigcache,
		Number:   0,
		Hash:     hash,
		Signers:make(map[int] common.Address),
		Votes: make(map[common.Address] *Vote),
		Tally: make(map[common.Address] *big.Int),
		Voters:make(map[common.Address] *big.Int),
		HeaderTime:config.GenesisTimestamp - 1, //
		LoopStartTime:config.GenesisTimestamp,
	}

	for _, vote := range votes {
		// init Votes from each vote
		snap.Votes[vote.Voter] = vote

		// init Tally
		_, ok := snap.Tally[vote.Candidate]
		if !ok{
			snap.Tally[vote.Candidate] = big.NewInt(0)
		}
		snap.Tally[vote.Candidate].Add(snap.Tally[vote.Candidate], vote.Stake)

		// init Voters
		snap.Voters[vote.Voter] = big.NewInt(0) // block number is 0 , vote in genesis block

	}

	for i := 0; i < int(config.MaxSignerCount); i++{
		snap.Signers[i] = config.SelfVoteSigners[i % len(config.SelfVoteSigners)]
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
		config:   s.config,
		sigcache: s.sigcache,
		Number:   s.Number,
		Hash:     s.Hash,

		Signers:make(map[int] common.Address ),
		Votes: make(map[common.Address] *Vote ),
		Tally: make(map[common.Address] *big.Int),
		Voters: make(map[common.Address] *big.Int),

		HeaderTime:s.HeaderTime,
		LoopStartTime:s.LoopStartTime,

	}

	for index, address := range s.Signers {
		cpy.Signers[index] = address
	}
	for voter, vote := range s.Votes {
		cpy.Votes[voter] = vote
	}
	for candidate, tally := range s.Tally {
		cpy.Tally[candidate] = tally
	}
	for voter, number := range s.Voters {
		cpy.Voters[voter] = number
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

		snap.HeaderTime = header.Time.Uint64()

		headerExtra := HeaderExtra{}
		rlp.DecodeBytes(header.Extra[extraVanity:len(header.Extra)-extraSeal],&headerExtra)
		snap.LoopStartTime = headerExtra.LoopStartTime
		snap.Signers  = make(map[int] common.Address)
		for i,sig := range headerExtra.SignerQueue{
			snap.Signers[i] = sig
		}
		// deal the new vote from voter
		for _, vote := range headerExtra.CurrentBlockVotes{
			// update Votes, Tally, Voters data
			if lastVote, ok := snap.Votes[vote.Voter]; ok{
				snap.Tally[lastVote.Candidate].Sub(snap.Tally[lastVote.Candidate], lastVote.Stake)
			}
			if _,ok := snap.Tally[vote.Candidate]; ok{

				snap.Tally[vote.Candidate].Add(snap.Tally[vote.Candidate], vote.Stake)
			}else{
				snap.Tally[vote.Candidate] = vote.Stake
			}

			snap.Votes[vote.Voter] = &vote
			snap.Voters[vote.Voter] = header.Number
		}
		// deal the voter which balance modified
		for _, txVote := range headerExtra.ModifyPredecessorVotes{
			if lastVote, ok := snap.Votes[txVote.Voter]; ok{
				snap.Tally[lastVote.Candidate].Sub(snap.Tally[lastVote.Candidate], lastVote.Stake)
				snap.Tally[lastVote.Candidate].Add(snap.Tally[lastVote.Candidate], txVote.Stake )
				snap.Votes[txVote.Voter] = &Vote{Voter:txVote.Voter,Candidate:lastVote.Candidate,Stake:txVote.Stake}
				// do not modify header number of snap.Voters
			}
		}
	}
	snap.Number += uint64(len(headers))
	snap.Hash = headers[len(headers)-1].Hash()

	// deal the expired vote,
	if len(snap.Voters) > int(s.config.MaxSignerCount) {
		for voterAddress, voteNumber := range snap.Voters {
			if snap.Number - voteNumber.Uint64() > s.config.Epoch {
				// clear the vote
				if expiredVote, ok := snap.Votes[voterAddress]; ok{
					snap.Tally[expiredVote.Candidate].Sub(snap.Tally[expiredVote.Candidate], expiredVote.Stake)
					delete(snap.Votes, expiredVote.Voter)
					delete(snap.Voters, expiredVote.Voter)
				}
			}
		}
	}
	return snap, nil
}


// inturn returns if a signer at a given block height is in-turn or not.
func (s *Snapshot) inturn(signer common.Address,  headerTime uint64) bool {

	// if all node stop more than period of one loop
	loopIndex := int((headerTime - s.LoopStartTime) / s.config.Period) % len(s.Signers)
	if currentSigner, ok := s.Signers[loopIndex]; !ok {
		return false
	}else{
		if currentSigner != signer{
			return false
		}
	}
	return true
}



type BigIntSlice []*big.Int

func (s BigIntSlice) Len() int { return len(s) }
func (s BigIntSlice) Swap(i, j int){ s[i], s[j] = s[j], s[i] }
func (s BigIntSlice) Less(i, j int) bool { return s[i].Cmp(s[j]) < 0 }


// get signer queue when one loop finished
func (s *Snapshot) getSignerQueue() []common.Address {

	var stakeList []*big.Int
	var topStakeAddress []common.Address

	for _, stake := range s.Tally {
		stakeList = append(stakeList, stake)
	}

	sort.Sort(BigIntSlice(stakeList))
	minStakeForCandidate := s.config.MinVoterBalance

	if len(stakeList) >= int(s.config.MaxSignerCount) {
		minStakeForCandidate = stakeList[s.config.MaxSignerCount - 1]
	}
	for address, stake := range s.Tally{
		if len(topStakeAddress) == int(s.config.MaxSignerCount) {
			break
		}
		if stake.Cmp(minStakeForCandidate) >= 0 {
			topStakeAddress = append(topStakeAddress, address)
		}
	}
	// Set the top candidates in random order
	for i:= 0;i< len(topStakeAddress);i ++{
		newPos := rand.Int() % len(topStakeAddress)
		topStakeAddress[i],topStakeAddress[newPos] = topStakeAddress[newPos],topStakeAddress[i]
	}

	return topStakeAddress

}

// check if address belong to voter
func (s *Snapshot) isVoter(address common.Address) bool{
	if _, ok := s.Voters[address]; ok{
		return true
	}
	return false
}
