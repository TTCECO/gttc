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

// Vote represents a single vote that an authorized signer made to modify the
// list of authorizations.
type Vote struct {
	Signer    common.Address `json:"signer"`    // Authorized signer that cast this vote
	Block     uint64         `json:"block"`     // Block number the vote was cast in (expire old votes)
	Address   common.Address `json:"address"`   // Account being voted on to change its authorization
	Authorize bool           `json:"authorize"` // Whether to authorize or deauthorize the voted account
}

// Tally is a simple vote tally to keep the current score of votes. Votes that
// go against the proposal aren't counted since it's equivalent to not voting.
type Tally struct {
	Authorize bool `json:"authorize"` // Whether the vote is about authorizing or kicking someone
	Votes     int  `json:"votes"`     // Number of votes until now wanting to pass the proposal
}

// Snapshot is the state of the authorization voting at a given point in time.
type Snapshot struct {
	config   *params.AlienConfig // Consensus engine parameters to fine tune behavior
	sigcache *lru.ARCCache        // Cache of recent block signatures to speed up ecrecover

	Number  uint64                      `json:"number"`  // Block number where the snapshot was created
	Hash    common.Hash                 `json:"hash"`    // Block hash where the snapshot was created

	////////////////
	XXXSigners map[int] common.Address 		`json:"xxxsigners"`
	XXXVotes []*TVote						`json:"xxxvotes"`
	XXXTally map[common.Address] *big.Int	`json:"xxxtally"`

	HeaderTime uint64		`json:"headerTime"`
	LoopStartTime uint64  	`json:"loopStartTime"`

}

// newSnapshot creates a new snapshot with the specified startup parameters. This
// method does not initialize the set of recent signers, so only ever use if for
// the genesis block.
func newSnapshot(config *params.AlienConfig, sigcache *lru.ARCCache, number uint64, hash common.Hash, signers []common.Address, tvotes []*TVote, headerTime uint64) *Snapshot {
	snap := &Snapshot{
		config:   config,
		sigcache: sigcache,
		Number:   number,
		Hash:     hash,

		XXXSigners:make(map[int] common.Address),
		XXXVotes: tvotes,
		XXXTally: make(map[common.Address] *big.Int),
		HeaderTime:headerTime,
		LoopStartTime:headerTime,

	}

	for _, vote := range tvotes {
		_, ok := snap.XXXTally[vote.Candidate]
		if !ok{
			snap.XXXTally[vote.Candidate] = big.NewInt(0)
		}

		snap.XXXTally[vote.Candidate].Add(snap.XXXTally[vote.Candidate], &vote.Stake)

	}


	fill_loop := false
	for tmp_index := 0; tmp_index < int(config.MaxSignerCount) ; {
		for  candidate, _ := range snap.XXXTally{

			snap.XXXSigners[tmp_index] = candidate
			tmp_index += 1
			if tmp_index == int(config.MaxSignerCount)  {
				fill_loop = true
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


		XXXSigners:make(map[int] common.Address),
		XXXVotes: make([]*TVote, len(s.XXXVotes)),
		XXXTally: make(map[common.Address] *big.Int),

		HeaderTime:s.HeaderTime,
		LoopStartTime:s.LoopStartTime,

	}

	for index := range s.XXXSigners {
		cpy.XXXSigners[index] = s.XXXSigners[index]
	}
	copy(cpy.XXXVotes, s.XXXVotes)

	for address, tally := range s.XXXTally {
		cpy.XXXTally[address] = tally
	}
	return cpy
}

// validVote returns whether it makes sense to cast the specified vote in the
// given snapshot context (e.g. don't try to add an already authorized signer).
func (s *Snapshot) validVote(address common.Address, authorize bool) bool {
	return true
}




// cast adds a new vote into the tally.
func (s *Snapshot) xxxcast(candidate common.Address, stake big.Int) bool {

	s.XXXTally[candidate].Add(s.XXXTally[candidate], &stake)

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
		if loop_signer, ok := snap.XXXSigners[loop_index]; !ok {

			return nil, errUnauthorized
		}else{
			// todo : check if this signer should seal this block by timestamp in header
			if loop_signer != signer{
				return nil, errUnauthorized
			}

		}


		for _,tvote := range headerExtra.CurrentBlockVotes{
			if snap.xxxcast(tvote.Candidate,tvote.Stake) {

				snap.XXXVotes = append(snap.XXXVotes, &TVote{
					Voter:tvote.Voter,
					Candidate:tvote.Candidate,
					Stake:tvote.Stake,
				})
			}

		}

		if number % s.config.MaxSignerCount == 0{

			snap.LoopStartTime = snap.HeaderTime
			// change the signers and random the

			fill_loop := false
			for tmp_index := 0; tmp_index < int(s.config.MaxSignerCount) ; {
				for  candidate, _ := range s.XXXTally{

					s.XXXSigners[tmp_index] = candidate
					tmp_index += 1
					if tmp_index == int(s.config.MaxSignerCount)  {
						fill_loop = true
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

	for index, _ :=range s.XXXSigners{
		signersMap[s.XXXSigners[index]] = struct{}{}
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
	if loop_signer, ok := s.XXXSigners[loop_index]; !ok {
		return false
	}else{

		// todo : check if this signer should seal this block by timestamp in header
		if loop_signer != signer{
			return false
		}
	}

	return true

}
