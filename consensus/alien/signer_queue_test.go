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
	"bytes"
	"math/big"
	"testing"

	"github.com/TTCECO/gttc/common"
	"github.com/TTCECO/gttc/params"
)

// Tests that voting is evaluated correctly for various simple and complex scenarios.
func TestQueue(t *testing.T) {
	// Define the various voting scenarios to test
	tests := []struct {
		addrNames      []string // accounts used in this case
		signers        []string
		number         uint64
		maxSignerCount uint64
		historyHash    []string
		tally          map[string]uint64
		punished       map[string]uint64
		result         []string // the result of current snapshot
	}{
		{
			/* 	Case 0:
			*   new loop signer queue is create at blocknumber 2,
			*   the new signerQueue is selected by tally, and random order depends on history hash
			*   step 1: the top 3(maxSignerCount) is selected order by tally -> A, B, C
			*   step 2: the top 3 signer is map to historyHash A->c, B->b, C->a
			*   step 3: the result order is set by historyHash decrease -> A, B, C
			 */
			addrNames:      []string{"A", "B", "C", "D"},
			signers:        []string{"A", "B", "C", "D"},
			number:         2,
			maxSignerCount: 3,
			historyHash:    []string{"a", "b", "c"},
			tally:          map[string]uint64{"D": 5, "A": 30, "B": 20, "C": 10},
			punished:       map[string]uint64{},
			result:         []string{"A", "B", "C"},
		},
		{
			/* 	Case 1:
			*   follow test case 0. the tally is same but history hash is x,b,c
			*   step 1: the top 3(maxSignerCount) is selected order by tally -> A, B, C
			*   step 2: the top 3 signer is map to historyHash A->c, B->b, C->x
			*   step 3: the result order is set by historyHash decrease -> C, A, B
			*
			 */
			addrNames:      []string{"A", "B", "C"},
			signers:        []string{"A", "B", "C"},
			number:         2,
			maxSignerCount: 3,
			historyHash:    []string{"x", "b", "c"},
			tally:          map[string]uint64{"A": 30, "B": 20, "C": 10},
			punished:       map[string]uint64{},
			result:         []string{"C", "A", "B"},
		},
		{
			/* 	Case 2:
			*   fllow test case 0. the tally is same but history hash is a,x,c
			*   step 1: the top 3(maxSignerCount) is selected order by tally -> A, B, C
			*   step 2: the top 3 signer is map to historyHash A->c, B->x, C->a
			*   step 3: the result order is set by historyHash decrease -> B, A, C
			*
			 */
			addrNames:      []string{"A", "B", "C"},
			signers:        []string{"A", "B", "C"},
			number:         2,
			maxSignerCount: 3,
			historyHash:    []string{"a", "x", "c"},
			tally:          map[string]uint64{"A": 30, "B": 20, "C": 10},
			punished:       map[string]uint64{},
			result:         []string{"B", "A", "C"},
		},
		{
			/* 	Case 3:
			*   fllow test case 0. the tally is changed and history hash is a,b,c
			*   step 1: the top 3(maxSignerCount) is selected order by tally -> B, A, C
			*   step 2: the top 3 signer is map to historyHash B->c, A->b, C->a
			*   step 3: the result order is set by historyHash decrease -> B, A, C
			*
			 */
			addrNames:      []string{"A", "B", "C"},
			signers:        []string{"A", "B", "C"},
			number:         2,
			maxSignerCount: 3,
			historyHash:    []string{"a", "b", "c"},
			tally:          map[string]uint64{"A": 30, "B": 40, "C": 10},
			punished:       map[string]uint64{},
			result:         []string{"B", "A", "C"},
		},
		{
			/* 	Case 4:
			*   fllow test case 0. the tally is changed and history hash is x,b,c
			*   step 1: the top 3(maxSignerCount) is selected order by tally -> B, A, C
			*   step 2: the top 3 signer is map to historyHash B->c, A->b, C->x
			*   step 3: the result order is set by historyHash decrease -> C, B, A
			*
			 */
			addrNames:      []string{"A", "B", "C"},
			signers:        []string{"A", "B", "C"},
			number:         2,
			maxSignerCount: 3,
			historyHash:    []string{"x", "b", "c"},
			tally:          map[string]uint64{"A": 30, "B": 40, "C": 10},
			punished:       map[string]uint64{},
			result:         []string{"C", "B", "A"},
		},
		{
			/* 	Case 5:
			*   fllow test case 0. the tally is changed and history hash is a,b,c
			*   step 1: the top 3(maxSignerCount) is selected order by tally -> but same t ally
			*   step 2: order by address (NOT A, B, C , address is [20]byte) desc, different by each test.
			*   step 3: the top 3 signer is map to historyHash
			*   step 4: the result order is set by historyHash decrease
			*   The result will be checked if order by address desc if result if empty
			 */
			addrNames:      []string{"A", "B", "C"},
			signers:        []string{"A", "B", "C"},
			number:         2,
			maxSignerCount: 3,
			historyHash:    []string{"a", "b", "c"},
			tally:          map[string]uint64{"A": 30, "B": 30, "C": 30},
			punished:       map[string]uint64{},
			result:         []string{}, // If tally(punished include) is same, then result order by their address
		},
		{
			/* 	Case 6:
			*   fllow test case 0. the tally is changed and history hash is x,b,c
			*   step 1: the top 3(maxSignerCount) is selected order by tally -> same tally
			*   step 2: consider the punished for each account, result order -> A, B, C
			*   step 3: the top 3 signer is map to historyHash A->c, B->b, C->x
			*   step 4: the result order is set by historyHash decrease -> C, A, B
			*
			 */
			addrNames:      []string{"A", "B", "C"},
			signers:        []string{"A", "B", "C"},
			number:         2,
			maxSignerCount: 3,
			historyHash:    []string{"x", "b", "c"},
			tally:          map[string]uint64{"A": 300, "B": 300, "C": 300},
			punished:       map[string]uint64{"A": 500, "B": 600, "C": 800},
			result:         []string{"C", "A", "B"},
		},
	}

	// Run through the scenarios and test them
	for i, tt := range tests {
		candidateNeedPD = false
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

		snap := &Snapshot{
			config:   &params.AlienConfig{MaxSignerCount: tt.maxSignerCount},
			Number:   tt.number,
			LCRS:     1,
			Tally:    make(map[common.Address]*big.Int),
			Punished: make(map[common.Address]uint64),
		}

		snap.Hash.SetString(tt.historyHash[len(tt.historyHash)-1])
		for _, hash := range tt.historyHash {
			var hh common.Hash
			hh.SetString(hash)
			snap.HistoryHash = append(snap.HistoryHash, hh)
		}

		for signer, tally := range tt.tally {
			snap.Tally[accounts.address(signer)] = big.NewInt(int64(tally))
		}

		for signer, punish := range tt.punished {
			snap.Punished[accounts.address(signer)] = punish
		}

		signerQueue, err := snap.createSignerQueue()
		if err != nil {
			t.Errorf("test %d: create signer queue fail , err = %s", i, err)
			continue
		}
		if len(tt.result) == 0 {
			for j, signer := range signerQueue {
				if j >= 1 && signerQueue[j-1].Str() < signerQueue[j].Str() {
					t.Errorf("test %d: result is not correct, signerQueue(%d) %s larger than signerQueue(%d) %s ", i, j, signer.Hex(), j-1, signerQueue[j-1].Hex())
				}
			}
		} else {
			if len(signerQueue) != len(tt.result) {
				t.Errorf("test %d: length of result is not correct. len(signerQueue) is %d, but len(tt.result) is %d", i, len(signerQueue), len(tt.result))
				continue
			}
			for j, signer := range signerQueue {
				if signer.Hex() != accounts.address(tt.result[j]).Hex() {
					t.Errorf("test %d: result is not correct signerQueue(%d) is %s, but result(%d) is %s", i, j, signer.Hex(), j, accounts.address(tt.result[j]).Hex())
				}
			}
		}

	}
}
