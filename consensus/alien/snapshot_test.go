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
	from string 		// name of from address
	to 	 string 		// name of to address
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
	Signers []string
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
		period 	uint64					// default 3
		epoch  	uint64					// default 30000
		maxSignerCount uint64			// default 5 for test
		minVoterBalance int				// default 50
		genesisTimestamp uint64			// default time.now() - period + 1
		selfVoters []testerSelfVoter	//
		txHeaders   []testerSingleHeader //
		result testerSnapshot			// the result of current snapshot
	}{
		{
			addrNames : []string{"A","B","C","D"},
			period:  uint64(3),
			epoch: uint64(31),
			maxSignerCount: uint64(15),
			minVoterBalance: 50,
			genesisTimestamp: uint64(0),
			selfVoters : []testerSelfVoter{{"A",100},{"B",200}},
			txHeaders: 	[]testerSingleHeader{
								{"A",[]testerTransaction{{from: "C", to: "D", balance: 200, isVote:true},},},
								},
			result: 	testerSnapshot{
								Signers:[]string{0:"A",1:"B"},
								Tally:map[string]int {"A":100,"B":200,"D":200},
								Voters:map[string]int {"A":0,"B":0,"C":1},
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
		var genesisVotes  []*Vote				// for create the new snapshot of genesis block
		var selfVoteSigners []common.Address	// for header extra
		alreadyVote := make(map[common.Address] struct{})
		for _, voter := range tt.selfVoters {
			if _, ok := alreadyVote[accounts.address(voter.voter)]; !ok {
				genesisVotes = append(genesisVotes, &Vote{
					Voter: accounts.address(voter.voter),
					Candidate: accounts.address(voter.voter),
					Stake: big.NewInt(int64(voter.balance)),
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
								Period: tt.period,
								Epoch: tt.epoch ,
								MinVoterBalance: big.NewInt(int64(tt.minVoterBalance)),
								MaxSignerCount: tt.maxSignerCount ,
								SelfVoteSigners: selfVoteSigners,
								}, db)


		// Assemble a chain of headers from the cast votes
		headers := make([]*types.Header, len(tt.txHeaders))
		for j, header := range tt.txHeaders {
			var currentBlockVotes []Vote
			var modifyPredecessorVotes []Vote
			for _,trans := range header.txs {
				if trans.isVote{
					// vote event
					currentBlockVotes = append(currentBlockVotes, Vote{
						Voter: accounts.address(trans.from),
						Candidate: accounts.address(trans.to),
						Stake: big.NewInt(int64(trans.balance)),
					})
				}else {
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
			// (j==0) means (header.Number==1)
			if j == 0 {
				for k := 0; k < int(tt.maxSignerCount); k++{
					currentHeaderExtra.SignerQueue = append(currentHeaderExtra.SignerQueue, selfVoteSigners[k % len(selfVoteSigners)])
				}
				currentHeaderExtra.LoopStartTime = tt.genesisTimestamp // here should be parent genesisTimestamp

			}else {
				// decode parent header.extra
				currentHeaderExtra := HeaderExtra{}
				rlp.DecodeBytes(headers[j-1].Extra[extraVanity:len(headers[j-1].Extra)-extraSeal],&currentHeaderExtra)

				// means header.Number % tt.maxSignerCount == 0
				if (j + 1 )% int(tt.maxSignerCount) == 0{
					snap, err :=alien.snapshot(&testerChainReader{db: db}, headers[j-1].Number.Uint64(), headers[j-1].Hash(), headers, nil)
					if err != nil {
						t.Errorf("test %d: failed to create voting snapshot: %v", i, err)
						continue
					}
					currentHeaderExtra.SignerQueue = snap.getSignerQueue()
					currentHeaderExtra.LoopStartTime = currentHeaderExtra.LoopStartTime + tt.period * tt.maxSignerCount
				}else{}
			}

			currentHeaderExtra.CurrentBlockVotes = currentBlockVotes
			currentHeaderExtra.ModifyPredecessorVotes = modifyPredecessorVotes
			currentHeaderExtraEnc,err := rlp.EncodeToBytes(currentHeaderExtra)
			if err != nil {
				t.Errorf("test %d: failed to rlp encode to bytes: %v", currentHeaderExtra, err)
				continue
			}
			// Create the genesis block with the initial set of signers
			ExtraData := make([]byte, extraVanity+len(currentHeaderExtraEnc)+extraSeal)
			copy(ExtraData[extraVanity:], currentHeaderExtraEnc)

			headers[j] = &types.Header{
				Number:   big.NewInt(int64(j) + 1),
				Time:     big.NewInt((int64(j) + 1) * int64(defaultBlockPeriod) - 1) ,
				Coinbase: accounts.address(header.signer),
				Extra:    ExtraData,
			}
			if j > 0 {
				headers[j].ParentHash = headers[j-1].Hash()
			}
			accounts.sign(headers[j], header.signer)


			// Pass all the headers through alien and ensure tallying succeeds
			_, err =alien.snapshot(&testerChainReader{db: db}, headers[j].Number.Uint64() , headers[j].Hash(), headers, genesisVotes)
			genesisVotes = []*Vote{}
			if err != nil {
				t.Errorf("test %d: failed to create voting snapshot: %v", i, err)
				continue
			}
		}

		// verify the result in test case
		head := headers[len(headers) - 1]
		snap, err :=alien.snapshot(&testerChainReader{db: db}, head.Number.Uint64(), head.Hash(), headers, nil)
		//
		if err != nil {
			t.Errorf("test %d: failed to create voting snapshot: %v", i, err)
			continue
		}
		// check signers
		signers := map[common.Address]int{}
		for _,signer := range snap.Signers{
			signers[signer] = 1
		}
		for _,signer := range tt.result.Signers{
			signers[accounts.address(signer)] += 1
		}
		for address ,cnt := range signers{
			if cnt != 2{
				t.Errorf("test %d: signer address: %v not in result signers", i, address)
				continue
			}
		}
		// check tally
		for name,tally := range tt.result.Tally{
			if big.NewInt(int64(tally)).Cmp(snap.Tally[accounts.address(name)]) != 0 {
				t.Errorf("test %d: tally %v address: %v, tally:%v ,result: %v", i, name, accounts.address(name), snap.Tally[accounts.address(name)],big.NewInt(int64(tally)))
				continue
			}
		}
		// check voters
		for name,number := range tt.result.Voters{
			if snap.Voters[accounts.address(name)].Cmp( big.NewInt(int64(number))) != 0{
				t.Errorf("test %d: voter %v address: %v, number:%v ,result: %v", i, name, accounts.address(name), snap.Voters[accounts.address(name)],big.NewInt(int64(number)))
				continue
			}
		}
		// check votes
		for name,vote := range tt.result.Votes{
			snapVote,ok := snap.Votes[accounts.address(name)]
			if !ok {
				t.Errorf("test %d: votes %v address: %v can not found", i, name, accounts.address(name))

			}
			if snapVote.Voter != accounts.address(vote.voter){
				t.Errorf("test %d: votes voter dismatch %v address: %v  , show in snap is %v", i, vote.voter, accounts.address(vote.voter), snapVote.Voter)
			}
			if snapVote.Candidate != accounts.address(vote.candidate){
				t.Errorf("test %d: votes candidate dismatch %v address: %v , show in snap is %v ", i, vote.candidate, accounts.address(vote.candidate), snapVote.Candidate)
			}
			if snapVote.Stake.Cmp(big.NewInt(int64(vote.stake))) != 0 {
				t.Errorf("test %d: votes stake dismatch %v ,show in snap is %v ", i, vote.stake, snapVote.Stake)
			}
		}


	}
}
