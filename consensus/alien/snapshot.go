// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package alien

import (
	"encoding/json"

	"github.com/TTCECO/gttc/common"
	"github.com/TTCECO/gttc/core/types"
	"github.com/TTCECO/gttc/ethdb"
	"github.com/TTCECO/gttc/params"
	lru "github.com/hashicorp/golang-lru"

	"github.com/TTCECO/gttc/rlp"
	"math/big"

)


// Snapshot is the state of the authorization voting at a given point in time.
type Snapshot struct {
	config   *params.AlienConfig // Consensus engine parameters to fine tune behavior
	sigcache *lru.ARCCache        // Cache of recent block signatures to speed up ecrecover

	Number  uint64                      `json:"number"`  // Block number where the snapshot was created
	Hash    common.Hash                 `json:"hash"`    // Block hash where the snapshot was created

	Signers map[int] common.Address 		`json:"signers"`
	Votes []*Vote						`json:"votes"`
	Tally map[common.Address] *big.Int	`json:"tally"`

	HeaderTime uint64		`json:"headerTime"`
	LoopStartTime uint64  	`json:"loopStartTime"`

}

// newSnapshot creates a new snapshot with the specified startup parameters. This
// method does not initialize the set of recent signers, so only ever use if for
// the genesis block.
func newSnapshot(config *params.AlienConfig, sigcache *lru.ARCCache, number uint64, hash common.Hash, signers []common.Address, votes []*Vote, headerTime uint64) *Snapshot {
	snap := &Snapshot{
		config:   config,
		sigcache: sigcache,
		Number:   number,
		Hash:     hash,

		Signers:make(map[int] common.Address),
		Votes: votes,
		Tally: make(map[common.Address] *big.Int),
		HeaderTime:headerTime,
		LoopStartTime:headerTime,

	}

	for _, vote := range votes {
		_, ok := snap.Tally[vote.Candidate]
		if !ok{
			snap.Tally[vote.Candidate] = big.NewInt(0)
		}

		snap.Tally[vote.Candidate].Add(snap.Tally[vote.Candidate], &vote.Stake)

	}


	fill_loop := false
	for tmp_index := 0; tmp_index < int(config.MaxSignerCount) ; {
		for  candidate, _ := range snap.Tally{

			snap.Signers[tmp_index] = candidate
			tmp_index += 1
			if tmp_index == int(config.MaxSignerCount) {
				fill_loop = true
				break
			}

		}
		if fill_loop == true {
			break
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


		Signers:make(map[int] common.Address),
		Votes: make([]*Vote, len(s.Votes)),
		Tally: make(map[common.Address] *big.Int),

		HeaderTime:s.HeaderTime,
		LoopStartTime:s.LoopStartTime,

	}

	for index := range s.Signers {
		cpy.Signers[index] = s.Signers[index]
	}
	copy(cpy.Votes, s.Votes)

	for address, tally := range s.Tally {
		cpy.Tally[address] = tally
	}
	return cpy
}

// validVote returns whether it makes sense to cast the specified vote in the
// given snapshot context (e.g. don't try to add an already authorized signer).
func (s *Snapshot) validVote(address common.Address, authorize bool) bool {
	return true
}




// cast adds a new vote into the tally.
func (s *Snapshot) cast(candidate common.Address, stake big.Int) bool {

	s.Tally[candidate].Add(s.Tally[candidate], &stake)

	return true
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
		number := header.Number.Uint64()

		// Resolve the authorization key and check against signers
		signer, err := ecrecover(header, s.sigcache)
		if err != nil {
			return nil, err
		}

		snap.HeaderTime = header.Time.Uint64()

		headerExtra := HeaderExtra{}
		rlp.DecodeBytes(header.Extra[extraVanity:len(header.Extra)-extraSeal],&headerExtra)

		// todo : from the timestamp in header calculate the index of signer address
		loop_index := int((header.Time.Uint64() - headerExtra.LoopStartTime) /  s.config.Period)
		if loop_signer, ok := snap.Signers[loop_index]; !ok {

			return nil, errUnauthorized
		}else{
			// todo : check if this signer should seal this block by timestamp in header
			if loop_signer != signer{
				return nil, errUnauthorized
			}

		}


		for _, vote := range headerExtra.CurrentBlockVotes{
			if snap.cast(vote.Candidate, vote.Stake) {

				snap.Votes = append(snap.Votes, &Vote{
					Voter: vote.Voter,
					Candidate: vote.Candidate,
					Stake: vote.Stake,
				})
			}

		}

		if number % s.config.MaxSignerCount == 0{

			snap.LoopStartTime = snap.HeaderTime
			// change the signers and random the

			fill_loop := false
			for tmp_index := 0; tmp_index < int(s.config.MaxSignerCount) ; {
				for  candidate, _ := range s.Tally{

					s.Signers[tmp_index] = candidate
					tmp_index += 1
					if tmp_index == int(s.config.MaxSignerCount) {
						fill_loop = true
						break
					}

				}
				if fill_loop == true {
					break
				}
			}
		}
	}
	snap.Number += uint64(len(headers))
	snap.Hash = headers[len(headers)-1].Hash()

	return snap, nil
}

// signers retrieves the list of authorized signers in ascending order.
func (s *Snapshot) signers() []common.Address {
	signersMap := make(map[common.Address]struct{})

	for index, _ :=range s.Signers{
		signersMap[s.Signers[index]] = struct{}{}
	}

	var signers []common.Address
	for signer, _:= range signersMap {
		signers = append(signers,signer)
	}

	return signers
}

// inturn returns if a signer at a given block height is in-turn or not.
func (s *Snapshot) inturn(signer common.Address, loopStartTime uint64, headerTime uint64) bool {

	loop_index := int((headerTime- loopStartTime) /  s.config.Period)
	if loop_signer, ok := s.Signers[loop_index]; !ok {
		return false
	}else{

		// todo : check if this signer should seal this block by timestamp in header
		if loop_signer != signer{
			return false
		}
	}

	return true

}
