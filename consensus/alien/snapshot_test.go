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
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/TTCECO/gttc/common"
	"github.com/TTCECO/gttc/core"
	"github.com/TTCECO/gttc/core/rawdb"
	"github.com/TTCECO/gttc/core/types"
	"github.com/TTCECO/gttc/crypto"
	"github.com/TTCECO/gttc/ethdb"
	"github.com/TTCECO/gttc/params"
	"github.com/TTCECO/gttc/rlp"
)

type testerTransaction struct {
	from    string // name of from address
	to      string // name of to address
	balance int    // balance address in snap.voter
	isVote  bool   // is msg in data is "ufo:1:event:vote"
}

type testerSingleHeader struct {
	txs []testerTransaction
}

type testerSelfVoter struct {
	voter   string // name of self voter address in genesis block
	balance int    // balance
}

type testerVote struct {
	voter     string
	candidate string
	stake     int
}

type testerSnapshot struct {
	Signers []string
	Votes   map[string]*testerVote
	Tally   map[string]int
	Voters  map[string]int
}

// testerAccountPool is a pool to maintain currently active tester accounts,
// mapped from textual names used in the tests below to actual Ethereum private
// keys capable of signing transactions.
type testerAccountPool struct {
	accounts map[string]*ecdsa.PrivateKey
}

func newTesterAccountPool() *testerAccountPool {
	return &testerAccountPool{
		accounts: make(map[string]*ecdsa.PrivateKey),
	}
}

func (ap *testerAccountPool) sign(header *types.Header, signer string) {
	// Ensure we have a persistent key for the signer
	if ap.accounts[signer] == nil {
		ap.accounts[signer], _ = crypto.GenerateKey()
	}
	// Sign the header and embed the signature in extra data
	sig, _ := crypto.Sign(sigHash(header).Bytes(), ap.accounts[signer])
	copy(header.Extra[len(header.Extra)-65:], sig)
}

func (ap *testerAccountPool) address(account string) common.Address {
	// Ensure we have a persistent key for the account
	if ap.accounts[account] == nil {
		ap.accounts[account], _ = crypto.GenerateKey()
	}
	// Resolve and return the Ethereum address
	return crypto.PubkeyToAddress(ap.accounts[account].PublicKey)
}

func (ap *testerAccountPool) name(address common.Address) string {
	for name, v := range ap.accounts {
		if crypto.PubkeyToAddress(v.PublicKey) == address {
			return name
		}
	}
	return ""
}

// testerChainReader implements consensus.ChainReader to access the genesis
// block. All other methods and requests will panic.
type testerChainReader struct {
	db ethdb.Database
}

func (r *testerChainReader) Config() *params.ChainConfig                 { return params.AllAlienProtocolChanges }
func (r *testerChainReader) CurrentHeader() *types.Header                { panic("not supported") }
func (r *testerChainReader) GetHeader(common.Hash, uint64) *types.Header { panic("not supported") }
func (r *testerChainReader) GetBlock(common.Hash, uint64) *types.Block   { panic("not supported") }
func (r *testerChainReader) GetHeaderByHash(common.Hash) *types.Header   { panic("not supported") }
func (r *testerChainReader) GetHeaderByNumber(number uint64) *types.Header {
	if number == 0 {
		return rawdb.ReadHeader(r.db, rawdb.ReadCanonicalHash(r.db, 0), 0)
	}
	panic("not supported")
}

// Tests that voting is evaluated correctly for various simple and complex scenarios.
func TestVoting(t *testing.T) {
	// Define the various voting scenarios to test
	tests := []struct {
		addrNames        []string             // accounts used in this case
		period           uint64               // default 3
		epoch            uint64               // default 30000
		maxSignerCount   uint64               // default 5 for test
		minVoterBalance  int                  // default 50
		genesisTimestamp uint64               // default time.now() - period + 1
		selfVoters       []testerSelfVoter    //
		txHeaders        []testerSingleHeader //
		result           testerSnapshot       // the result of current snapshot
	}{
		{
			/* 	Case 0:
			*	Just two self vote address A B in genesis
			*  	No votes or transactions through blocks
			 */
			addrNames:        []string{"A", "B"},
			period:           uint64(3),
			epoch:            uint64(31),
			maxSignerCount:   uint64(5),
			minVoterBalance:  50,
			genesisTimestamp: uint64(0),
			selfVoters:       []testerSelfVoter{{"A", 100}, {"B", 200}},
			txHeaders: []testerSingleHeader{
				{[]testerTransaction{}},
				{[]testerTransaction{}},
				{[]testerTransaction{}},
				{[]testerTransaction{}},
				{[]testerTransaction{}},
				{[]testerTransaction{}},
			},
			result: testerSnapshot{
				Signers: []string{"A", "B"},
				Tally:   map[string]int{"A": 100, "B": 200},
				Voters:  map[string]int{"A": 0, "B": 0},
				Votes: map[string]*testerVote{
					"A": {"A", "A", 100},
					"B": {"B", "B", 200},
				},
			},
		},
		{
			/*	Case 1:
			*	Two self vote address A B in  genesis
			* 	C vote D to be signer in block 3
			* 	But current loop do not finish, so D is not signer,
			* 	the vote info already in Tally, Voters and Votes
			 */
			addrNames:        []string{"A", "B", "C", "D"},
			period:           uint64(3),
			epoch:            uint64(31),
			maxSignerCount:   uint64(7),
			minVoterBalance:  50,
			genesisTimestamp: uint64(0),
			selfVoters:       []testerSelfVoter{{"A", 100}, {"B", 200}},
			txHeaders: []testerSingleHeader{
				{[]testerTransaction{}},
				{[]testerTransaction{}},
				{[]testerTransaction{{from: "C", to: "D", balance: 200, isVote: true}}},
				{[]testerTransaction{}},
			},
			result: testerSnapshot{
				Signers: []string{"A", "B"},
				Tally:   map[string]int{"A": 100, "B": 200, "D": 200},
				Voters:  map[string]int{"A": 0, "B": 0, "C": 3},
				Votes: map[string]*testerVote{
					"A": {"A", "A", 100},
					"B": {"B", "B", 200},
					"C": {"C", "D", 200},
				},
			},
		},
		{
			/*	Case 2:
			*	Two self vote address in  genesis
			* 	C vote D to be signer in block 2
			* 	But balance of C is lower than minVoterBalance,
			*   so this vote not processed, D is not signer
			* 	the vote info is dropped .
			 */
			addrNames:        []string{"A", "B", "C", "D"},
			period:           uint64(3),
			epoch:            uint64(31),
			maxSignerCount:   uint64(5),
			minVoterBalance:  50,
			genesisTimestamp: uint64(0),
			selfVoters:       []testerSelfVoter{{"A", 100}, {"B", 200}},
			txHeaders: []testerSingleHeader{
				{[]testerTransaction{}},
				{[]testerTransaction{{from: "C", to: "D", balance: 20, isVote: true}}},
				{[]testerTransaction{}},
				{[]testerTransaction{}},
				{[]testerTransaction{}},
				{[]testerTransaction{}},
			},
			result: testerSnapshot{
				Signers: []string{"A", "B"},
				Tally:   map[string]int{"A": 100, "B": 200},
				Voters:  map[string]int{"A": 0, "B": 0},
				Votes: map[string]*testerVote{
					"A": {"A", "A", 100},
					"B": {"B", "B", 200},
				},
			},
		},
		{
			/*	Case 3:
			*	Two self vote address A B in  genesis
			* 	C vote D to be signer in block 3
			* 	balance of C is higher than minVoterBalance
			* 	D is signer in next loop
			 */
			addrNames:        []string{"A", "B", "C", "D"},
			period:           uint64(3),
			epoch:            uint64(31),
			maxSignerCount:   uint64(5),
			minVoterBalance:  50,
			genesisTimestamp: uint64(0),
			selfVoters:       []testerSelfVoter{{"A", 100}, {"B", 200}},
			txHeaders: []testerSingleHeader{
				{[]testerTransaction{}},
				{[]testerTransaction{}},
				{[]testerTransaction{{from: "C", to: "D", balance: 200, isVote: true}}},
				{[]testerTransaction{}},
				{[]testerTransaction{}},
				{[]testerTransaction{}},
				{[]testerTransaction{}},
			},
			result: testerSnapshot{
				Signers: []string{"A", "B", "D"},
				Tally:   map[string]int{"A": 100, "B": 200, "D": 200},
				Voters:  map[string]int{"A": 0, "B": 0, "C": 3},
				Votes: map[string]*testerVote{
					"A": {"A", "A", 100},
					"B": {"B", "B", 200},
					"C": {"C", "D", 200},
				},
			},
		},

		{
			/*	Case 4:
			*	Two self vote address A B in  genesis
			* 	C vote D to be signer in block 2
			*  	C vote B to be signer in block 3
			* 	balance of C is higher minVoterBalance
			* 	the first vote from C is dropped
			* 	the signers are still A and B
			 */
			addrNames:        []string{"A", "B", "C", "D"},
			period:           uint64(3),
			epoch:            uint64(31),
			maxSignerCount:   uint64(5),
			minVoterBalance:  50,
			genesisTimestamp: uint64(0),
			selfVoters:       []testerSelfVoter{{"A", 100}, {"B", 200}},
			txHeaders: []testerSingleHeader{
				{[]testerTransaction{}},
				{[]testerTransaction{{from: "C", to: "D", balance: 200, isVote: true}}},
				{[]testerTransaction{{from: "C", to: "B", balance: 180, isVote: true}}},
				{[]testerTransaction{}},
				{[]testerTransaction{}},
				{[]testerTransaction{}},
			},
			result: testerSnapshot{
				Signers: []string{"A", "B"},
				Tally:   map[string]int{"A": 100, "B": 380},
				Voters:  map[string]int{"A": 0, "B": 0, "C": 3},
				Votes: map[string]*testerVote{
					"A": {"A", "A", 100},
					"B": {"B", "B", 200},
					"C": {"C", "B", 180},
				},
			},
		},
		{
			/*	Case 5:
			*	Two self vote address A B in  genesis
			* 	C vote D to be signer in block 2
			*  	C transaction to E 20 in block 3
			*	In Voters, the vote block number of C is still 2, not 4
			 */
			addrNames:        []string{"A", "B", "C", "D", "E"},
			period:           uint64(3),
			epoch:            uint64(31),
			maxSignerCount:   uint64(5),
			minVoterBalance:  50,
			genesisTimestamp: uint64(0),
			selfVoters:       []testerSelfVoter{{"A", 100}, {"B", 200}},
			txHeaders: []testerSingleHeader{
				{[]testerTransaction{}},
				{[]testerTransaction{{from: "C", to: "D", balance: 100, isVote: true}}},
				{[]testerTransaction{}},
				{[]testerTransaction{{from: "C", to: "E", balance: 20, isVote: false}}}, // when C transaction to E, the balance of C is 20
				{[]testerTransaction{}},
				{[]testerTransaction{}},
			},
			result: testerSnapshot{
				Signers: []string{"A", "B", "D"},
				Tally:   map[string]int{"A": 100, "B": 200, "D": 20},
				Voters:  map[string]int{"A": 0, "B": 0, "C": 2},
				Votes: map[string]*testerVote{
					"A": {"A", "A", 100},
					"B": {"B", "B", 200},
					"C": {"C", "D", 20},
				},
			},
		},
		{
			/*	Case 6:
			*	Two self vote address A B in  genesis
			* 	C vote D , J vote K, H vote I  to be signer in block 2
			*   E vote F in block 3
			* 	The signers in the next loop is A,B,D,F,I but not K
			*	K is not top 5(maxsigercount) in Tally
			 */
			addrNames:        []string{"A", "B", "C", "D", "E", "F", "H", "I", "J", "K"},
			period:           uint64(3),
			epoch:            uint64(31),
			maxSignerCount:   uint64(5),
			minVoterBalance:  50,
			genesisTimestamp: uint64(0),
			selfVoters:       []testerSelfVoter{{"A", 100}, {"B", 200}},
			txHeaders: []testerSingleHeader{
				{[]testerTransaction{}},
				{[]testerTransaction{{from: "C", to: "D", balance: 110, isVote: true}, {from: "J", to: "K", balance: 80, isVote: true}, {from: "H", to: "I", balance: 160, isVote: true}}},
				{[]testerTransaction{{from: "E", to: "F", balance: 130, isVote: true}}},
				{[]testerTransaction{}},
				{[]testerTransaction{}},
				{[]testerTransaction{}},
				{[]testerTransaction{}},
			},
			result: testerSnapshot{
				Signers: []string{"A", "B", "D", "F", "I"},
				Tally:   map[string]int{"A": 100, "B": 200, "D": 110, "I": 160, "F": 130, "K": 80},
				Voters:  map[string]int{"A": 0, "B": 0, "C": 2, "H": 2, "J": 2, "E": 3},
				Votes: map[string]*testerVote{
					"A": {"A", "A", 100},
					"B": {"B", "B", 200},
					"C": {"C", "D", 110},
					"J": {"J", "K", 80},
					"H": {"H", "I", 160},
					"E": {"E", "F", 130},
				},
			},
		},
		{
			/*	Case 7:
			*	one self vote address A in  genesis
			* 	C vote D , J vote K, H vote I  to be signer in block 3
			*   E vote F in block 4
			* 	B vote B in block 5
			* 	The signers in the next loop is B, D,F,I,K
			*	current number - The block number of vote for A > epoch AND tally > maxSignerCount
			 */
			addrNames:        []string{"A", "B", "C", "D", "E", "F", "H", "I", "J", "K"},
			period:           uint64(3),
			epoch:            uint64(8),
			maxSignerCount:   uint64(5),
			minVoterBalance:  50,
			genesisTimestamp: uint64(0),
			selfVoters:       []testerSelfVoter{{"A", 100}},
			txHeaders: []testerSingleHeader{
				{[]testerTransaction{}},
				{[]testerTransaction{}},
				{[]testerTransaction{{from: "C", to: "D", balance: 110, isVote: true}, {from: "J", to: "K", balance: 80, isVote: true}, {from: "H", to: "I", balance: 160, isVote: true}}},
				{[]testerTransaction{{from: "E", to: "F", balance: 130, isVote: true}}},
				{[]testerTransaction{{from: "B", to: "B", balance: 200, isVote: true}}},
				{[]testerTransaction{}},
				{[]testerTransaction{}},
				{[]testerTransaction{}},
				{[]testerTransaction{}},
				{[]testerTransaction{}},
				{[]testerTransaction{}},
				{[]testerTransaction{}},
			},
			result: testerSnapshot{
				Signers: []string{"B", "D", "F", "I", "K"},
				Tally:   map[string]int{"B": 200, "D": 110, "I": 160, "F": 130, "K": 80},
				Voters:  map[string]int{"B": 5, "C": 3, "H": 3, "J": 3, "E": 4},
				Votes: map[string]*testerVote{
					"B": {"B", "B", 200},
					"C": {"C", "D", 110},
					"J": {"J", "K", 80},
					"H": {"H", "I", 160},
					"E": {"E", "F", 130},
				},
			},
		},
		{
			/*	Case 8:
			*	Two self vote address A,B in  genesis
			* 	C vote D , D vote C to be signer in block 3
			 */
			addrNames:        []string{"A", "B", "C", "D", "E"},
			period:           uint64(3),
			epoch:            uint64(31),
			maxSignerCount:   uint64(5),
			minVoterBalance:  50,
			genesisTimestamp: uint64(0),
			selfVoters:       []testerSelfVoter{{"A", 100}, {"B", 200}},
			txHeaders: []testerSingleHeader{
				{[]testerTransaction{}},
				{[]testerTransaction{}},
				{[]testerTransaction{{from: "C", to: "D", balance: 110, isVote: true}, {from: "D", to: "C", balance: 80, isVote: true}, {from: "C", to: "E", balance: 110, isVote: false}}},
				{[]testerTransaction{}},
				{[]testerTransaction{}},
				{[]testerTransaction{}},
			},
			result: testerSnapshot{
				Signers: []string{"B", "A", "C", "D"},
				Tally:   map[string]int{"B": 200, "D": 110, "A": 100, "C": 80},
				Voters:  map[string]int{"B": 0, "C": 3, "D": 3, "A": 0},
				Votes: map[string]*testerVote{
					"B": {"B", "B", 200},
					"A": {"A", "A", 100},
					"C": {"C", "D", 110},
					"D": {"D", "C", 80},
				},
			},
		},
	}
	// Run through the scenarios and test them
	for i, tt := range tests {
		// Create the account pool and generate the initial set of all address in addrNames
		accounts := newTesterAccountPool()
		addrNames := make([]common.Address, len(tt.addrNames))
		for j, signer := range tt.addrNames {
			addrNames[j] = accounts.address(signer)
		}
		for j := 0; j < len(addrNames); j++ {
			for k := j + 1; k < len(addrNames); k++ {
				if bytes.Compare(addrNames[j][:], addrNames[k][:]) > 0 {
					addrNames[j], addrNames[k] = addrNames[k], addrNames[j]
				}
			}
		}

		// Prepare data for the genesis block
		var genesisVotes []*Vote             // for create the new snapshot of genesis block
		var selfVoteSigners []common.Address // for header extra
		alreadyVote := make(map[common.Address]struct{})
		for _, voter := range tt.selfVoters {
			if _, ok := alreadyVote[accounts.address(voter.voter)]; !ok {
				genesisVotes = append(genesisVotes, &Vote{
					Voter:     accounts.address(voter.voter),
					Candidate: accounts.address(voter.voter),
					Stake:     big.NewInt(int64(voter.balance)),
				})
				selfVoteSigners = append(selfVoteSigners, accounts.address(voter.voter))
				alreadyVote[accounts.address(voter.voter)] = struct{}{}
			}
		}

		// extend length of extra, so address of CoinBase can keep signature .
		genesis := &core.Genesis{
			ExtraData: make([]byte, extraVanity+extraSeal),
		}

		// Create a pristine blockchain with the genesis injected
		db := ethdb.NewMemDatabase()
		genesis.Commit(db)

		// Create new alien
		alien := New(&params.AlienConfig{
			Period:          tt.period,
			Epoch:           tt.epoch,
			MinVoterBalance: big.NewInt(int64(tt.minVoterBalance)),
			MaxSignerCount:  tt.maxSignerCount,
			SelfVoteSigners: selfVoteSigners,
		}, db)

		// Assemble a chain of headers from the cast votes
		headers := make([]*types.Header, len(tt.txHeaders))
		for j, header := range tt.txHeaders {

			var currentBlockVotes []Vote
			var modifyPredecessorVotes []Vote
			for _, trans := range header.txs {
				if trans.isVote {
					if trans.balance >= tt.minVoterBalance {
						// vote event
						currentBlockVotes = append(currentBlockVotes, Vote{
							Voter:     accounts.address(trans.from),
							Candidate: accounts.address(trans.to),
							Stake:     big.NewInt(int64(trans.balance)),
						})
					}
				} else {
					// modify balance
					// modifyPredecessorVotes
					// only consider the voter
					modifyPredecessorVotes = append(modifyPredecessorVotes, Vote{
						Voter: accounts.address(trans.from),
						Stake: big.NewInt(int64(trans.balance)),
					})
				}
			}
			currentHeaderExtra := HeaderExtra{}
			signer := common.Address{}

			// (j==0) means (header.Number==1)
			if j == 0 {
				for k := 0; k < int(tt.maxSignerCount); k++ {
					currentHeaderExtra.SignerQueue = append(currentHeaderExtra.SignerQueue, selfVoteSigners[k%len(selfVoteSigners)])
				}
				currentHeaderExtra.LoopStartTime = tt.genesisTimestamp // here should be parent genesisTimestamp
				signer = selfVoteSigners[0]

			} else {
				// decode parent header.extra
				rlp.DecodeBytes(headers[j-1].Extra[extraVanity:len(headers[j-1].Extra)-extraSeal], &currentHeaderExtra)
				signer = currentHeaderExtra.SignerQueue[uint64(j)%tt.maxSignerCount]
				// means header.Number % tt.maxSignerCount == 0
				if (j+1)%int(tt.maxSignerCount) == 0 {
					snap, err := alien.snapshot(&testerChainReader{db: db}, headers[j-1].Number.Uint64(), headers[j-1].Hash(), headers, nil)
					if err != nil {
						t.Errorf("test %d: failed to create voting snapshot: %v", i, err)
						continue
					}
					currentHeaderExtra.SignerQueue = []common.Address{}
					newSignerQueue := snap.getSignerQueue()
					for k := 0; k < int(tt.maxSignerCount); k++ {
						currentHeaderExtra.SignerQueue = append(currentHeaderExtra.SignerQueue, newSignerQueue[k%len(newSignerQueue)])
					}
					currentHeaderExtra.LoopStartTime = currentHeaderExtra.LoopStartTime + tt.period*tt.maxSignerCount
				} else {
				}
			}

			currentHeaderExtra.CurrentBlockVotes = currentBlockVotes
			currentHeaderExtra.ModifyPredecessorVotes = modifyPredecessorVotes
			currentHeaderExtraEnc, err := rlp.EncodeToBytes(currentHeaderExtra)
			if err != nil {
				t.Errorf("test %d: failed to rlp encode to bytes: %v", i, err)
				continue
			}
			// Create the genesis block with the initial set of signers
			ExtraData := make([]byte, extraVanity+len(currentHeaderExtraEnc)+extraSeal)
			copy(ExtraData[extraVanity:], currentHeaderExtraEnc)

			headers[j] = &types.Header{
				Number:   big.NewInt(int64(j) + 1),
				Time:     big.NewInt((int64(j)+1)*int64(defaultBlockPeriod) - 1),
				Coinbase: signer,
				Extra:    ExtraData,
			}
			if j > 0 {
				headers[j].ParentHash = headers[j-1].Hash()
			}
			accounts.sign(headers[j], accounts.name(signer))

			// Pass all the headers through alien and ensure tallying succeeds
			_, err = alien.snapshot(&testerChainReader{db: db}, headers[j].Number.Uint64(), headers[j].Hash(), headers[:j+1], genesisVotes)
			genesisVotes = []*Vote{}
			if err != nil {
				t.Errorf("test %d: failed to create voting snapshot: %v", i, err)
				continue
			}
		}

		// verify the result in test case
		head := headers[len(headers)-1]
		snap, err := alien.snapshot(&testerChainReader{db: db}, head.Number.Uint64(), head.Hash(), headers, nil)
		//
		if err != nil {
			t.Errorf("test %d: failed to create voting snapshot: %v", i, err)
			continue
		}
		// check signers
		signers := map[common.Address]int{}
		for _, signer := range snap.Signers {
			signers[*signer] = 1

		}
		for _, signer := range tt.result.Signers {
			signers[accounts.address(signer)] += 2
		}

		for address, cnt := range signers {
			if cnt != 3 {
				t.Errorf("test %d: signer %v address: %v not in result signers %d", i, accounts.name(address), address, cnt)
				continue
			}
		}
		// check tally
		if len(tt.result.Tally) != len(snap.Tally) {
			t.Errorf("test %d: tally length result %d, snap %d dismatch", i, len(tt.result.Tally), len(snap.Tally))
		}
		for name, tally := range tt.result.Tally {
			if big.NewInt(int64(tally)).Cmp(snap.Tally[accounts.address(name)]) != 0 {
				t.Errorf("test %d: tally %v address: %v, tally:%v ,result: %v", i, name, accounts.address(name), snap.Tally[accounts.address(name)], big.NewInt(int64(tally)))
				continue
			}
		}
		// check voters
		if len(tt.result.Voters) != len(snap.Voters) {
			t.Errorf("test %d: voter length result %d, snap %d dismatch", i, len(tt.result.Voters), len(snap.Voters))
		}
		for name, number := range tt.result.Voters {
			if snap.Voters[accounts.address(name)].Cmp(big.NewInt(int64(number))) != 0 {
				t.Errorf("test %d: voter %v address: %v, number:%v ,result: %v", i, name, accounts.address(name), snap.Voters[accounts.address(name)], big.NewInt(int64(number)))
				continue
			}
		}
		// check votes

		if len(tt.result.Votes) != len(snap.Votes) {
			t.Errorf("test %d: votes length result %d, snap %d dismatch", i, len(tt.result.Votes), len(snap.Votes))
		}
		for name, vote := range tt.result.Votes {
			snapVote, ok := snap.Votes[accounts.address(name)]
			if !ok {
				t.Errorf("test %d: votes %v address: %v can not found", i, name, accounts.address(name))

			}
			if snapVote.Voter != accounts.address(vote.voter) {
				t.Errorf("test %d: votes voter dismatch %v address: %v  , show in snap is %v address: %v", i, vote.voter, accounts.address(vote.voter), accounts.name(snapVote.Voter), snapVote.Voter)
			}
			if snapVote.Candidate != accounts.address(vote.candidate) {
				t.Errorf("test %d: votes candidate dismatch %v address: %v , show in snap is %v address: %v ", i, vote.candidate, accounts.address(vote.candidate), accounts.name(snapVote.Candidate), snapVote.Candidate)
			}
			if snapVote.Stake.Cmp(big.NewInt(int64(vote.stake))) != 0 {
				t.Errorf("test %d: votes stake dismatch %v ,show in snap is %v ", i, vote.stake, snapVote.Stake)
			}
		}

	}
}
