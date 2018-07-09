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
	"time"
)

type testerTransaction struct {
	from string 		// name of from address
	to 	 string 		// name of to address
	value int 			// value
	balance int    		// balance address in snap.voter
	isVote bool			// is msg in data is "ufo:1:event:vote"
}



type testerSingleHeader struct {
	signer string		// signer of current block
	txs []testerTransaction
}

type testerSelfVoter struct {
	voter string		// name of self voter address in genesis block
	balance int 	// balance
}

type testerVote struct {
	voter string
	candidate string
	stake int
}

type testerSnapshot struct {
	Signers map[int] string
	Votes map[string] *testerVote
	Tally map[string] int
	Voters map[string] int
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
		addrNames []string 				// accounts used in this case
		epoch  	uint64					// default 30000
		maxSignerCount uint64			// default 5 for test
		minVoterBalance int				// default 50
		selfVoters []testerSelfVoter	//
		txHeaders   []testerSingleHeader //
		result testerSnapshot			// the result of current snapshot
	}{
		{
			addrNames : []string{"A","B","C","D"},
			selfVoters : []testerSelfVoter{{voter:"A",balance:100},{voter:"B",balance:200}},
			txHeaders: 	[]testerSingleHeader{
								{"A",[]testerTransaction{{from: "C", to: "D", balance: 200, value:0, isVote:true},},},
								},
			result: 	testerSnapshot{
								Signers:map[int]string{0:"A",1:"B"},
								Tally:map[string]int {"A":100,"B":200,"D":200},
								Voters:map[string]int {"A":0,"B":0,"D":3},
								Votes:map[string]*testerVote{
											"A":{"A","A",100},
											"B":{"B","B",200},
											"C":{"C","D",200},

											},
										},

		},
	}
	// Run through the scenarios and test them
	for i, tt := range tests {
		// Create the account pool and generate the initial set of signers
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

		genesisVotes := []*Vote{}
		alreadyVote := make(map[common.Address] struct{})
		for _, voter := range tt.selfVoters {

			if _, ok := alreadyVote[accounts.address(voter.voter)]; !ok {
				genesisVotes = append(genesisVotes, &Vote{
					Voter: accounts.address(voter.voter),
					Candidate: accounts.address(voter.voter),
					Stake: big.NewInt(int64(voter.balance)),
				})
				alreadyVote[accounts.address(voter.voter)] = struct{}{}
			}
		}
		currentHeaderExtra := HeaderExtra{}
		currentHeaderExtra.CurrentBlockVotes = []Vote{}
		currentHeaderExtra.ModifyPredecessorVotes = []Vote{}
		currentHeaderExtra.LoopStartTime = uint64(time.Now().Unix())
		currentHeaderExtra.SignerQueue = []common.Address{}

		currentHeaderExtraEnc,err := rlp.EncodeToBytes(currentHeaderExtra)
		if err != nil {
			t.Errorf("test %d: failed to rlp encode to bytes: %v", currentHeaderExtra, err)
			continue
		}
		// Create the genesis block with the initial set of signers
		genesis := &core.Genesis{
			ExtraData: make([]byte, extraVanity+len(currentHeaderExtraEnc)+extraSeal),
		}
		// Create a pristine blockchain with the genesis injected
		db := ethdb.NewMemDatabase()
		genesis.Commit(db)

		// Assemble a chain of headers from the cast votes
		headers := make([]*types.Header, len(tt.txHeaders))
		for j, header := range tt.txHeaders {
			for _,trans := range header.txs {
				currentBlockVotes := []Vote{}
				if trans.isVote{
					currentBlockVotes = append(currentBlockVotes, Vote{
						Voter: accounts.address(trans.from),
						Candidate: accounts.address(trans.to),
						Stake: big.NewInt(int64(trans.balance)),
					})
				}else {

				}
			}
			headers[j] = &types.Header{
				Number:   big.NewInt(int64(j) + 1),
				Time:     big.NewInt(int64(j) * int64(defaultBlockPeriod)),
				Coinbase: accounts.address(header.signer),
				Extra:    make([]byte, extraVanity+extraSeal),
			}
			if j > 0 {
				headers[j].ParentHash = headers[j-1].Hash()
			}
			accounts.sign(headers[j], header.signer)
		}
		// Pass all the headers through alien and ensure tallying succeeds
		head := headers[len(headers)-1]

		selfVoteSigners := []common.Address{}
		selfVoteSigners = append(selfVoteSigners, accounts.address("A"))
		selfVoteSigners = append(selfVoteSigners, accounts.address("B"))

		snap, err := New(&params.AlienConfig{Epoch: tt.epoch ,MinVoterBalance: big.NewInt(100), MaxSignerCount: 21 , SelfVoteSigners: selfVoteSigners},
			db).snapshot(&testerChainReader{db: db}, head.Number.Uint64(), head.Hash(), headers, genesisVotes)


		if err != nil {
			t.Errorf("test %d: failed to create voting snapshot: %v", i, err)
			continue
		}
		result := snap.getSignerQueue()
		if result[0] !=  accounts.address(tt.result.Signers[0]){
			t.Errorf("test mismatch")
			continue
		}



	}
}
