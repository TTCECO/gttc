// Copyright 2019 The TTC Authors
// This file is part of the TTC library.
//
// The TTC library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The TTC library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the TTC library. If not, see <http://www.gnu.org/licenses/>.

package alien

import (
	"testing"

	"github.com/TTCECO/gttc/common"
)

func TestAlien_PenaltyTrantor(t *testing.T) {
	tests := []struct {
		last    string
		current string
		queue   []string
		lastQ   []string
		result  []string // the result of missing
	}{
		{
			/* 	Case 0:
			 *  simple loop order, miss nothing
			 *  A -> B -> C
			 */
			last:    "A",
			current: "B",
			queue:   []string{"A", "B", "C"},
			lastQ:   []string{},
			result:  []string{},
		},
		{
			/* 	Case 1:
			 *  same loop, missing B
			 *  A -> B -> C
			 */
			last:    "A",
			current: "C",
			queue:   []string{"A", "B", "C"},
			lastQ:   []string{},
			result:  []string{"B"},
		},
		{
			/* 	Case 2:
			 *  same loop, not start from the first one
			 *  C -> A -> B
			 */
			last:    "C",
			current: "B",
			queue:   []string{"A", "B", "C"},
			lastQ:   []string{},
			result:  []string{"A"},
		},
		{
			/* 	Case 3:
			 *  same loop, missing two
			 *  A -> B -> C
			 */
			last:    "C",
			current: "C",
			queue:   []string{"A", "B", "C"},
			lastQ:   []string{},
			result:  []string{"A", "B"},
		},
		{
			/* 	Case 4:
			 *  cross loop
			 *  B -> A -> B -> C -> A
			 */
			last:    "B",
			current: "B",
			queue:   []string{"A", "B", "C"},
			lastQ:   []string{"C", "A", "B"},
			result:  []string{"A"},
		},
		{
			/* 	Case 5:
			 *  cross loop, nothing missing
			 *  A -> C -> A -> B -> C
			 */
			last:    "A",
			current: "C",
			queue:   []string{"A", "B", "C"},
			lastQ:   []string{"C", "A", "B"},
			result:  []string{},
		},
		{
			/* 	Case 6:
			 *  cross loop, two signers missing in last loop
			 *  C -> B -> C -> A
			 */
			last:    "C",
			current: "A",
			queue:   []string{"A", "B", "C"},
			lastQ:   []string{"C", "A", "B"},
			result:  []string{"B", "C"},
		},
	}

	// Run through the test
	for i, tt := range tests {
		// Create the account pool and generate the initial set of all address in addrNames
		accounts := newTesterAccountPool()
		addrQueue := make([]common.Address, len(tt.queue))
		for j, signer := range tt.queue {
			addrQueue[j] = accounts.address(signer)
		}

		extra := HeaderExtra{SignerQueue: addrQueue}
		var lastExtra HeaderExtra
		if len(tt.lastQ) > 0 {
			lastAddrQueue := make([]common.Address, len(tt.lastQ))
			for j, signer := range tt.lastQ {
				lastAddrQueue[j] = accounts.address(signer)
			}
			lastExtra = HeaderExtra{SignerQueue: lastAddrQueue}
		}

		missing := getSignerMissingTrantor(accounts.address(tt.last), accounts.address(tt.current), &extra, &lastExtra)

		signersMissing := make(map[string]bool)
		for _, signer := range missing {
			signersMissing[accounts.name(signer)] = true
		}
		if len(missing) != len(tt.result) {
			t.Errorf("test %d: the length of missing not equal to the length of result, Result is %v not %v  ", i, signersMissing, tt.result)
		}

		for j := 0; j < len(missing); j++ {
			if _, ok := signersMissing[tt.result[j]]; !ok {
				t.Errorf("test %d: the signersMissing is not equal Result is %v not %v ", i, signersMissing, tt.result)
			}
		}

	}
}

func TestAlien_Penalty(t *testing.T) {
	tests := []struct {
		last    string
		current string
		queue   []string
		newLoop bool
		result  []string // the result of current snapshot
	}{
		{
			/* 	Case 0:
			 *  simple loop order
			 */
			last:    "A",
			current: "B",
			queue:   []string{"A", "B", "C"},
			newLoop: false,
			result:  []string{},
		},
		{
			/* 	Case 1:
			 * simple loop order, new loop, no matter which one is current signer
			 */
			last:    "C",
			current: "A",
			queue:   []string{"A", "B", "C"},
			newLoop: true,
			result:  []string{},
		},
		{
			/* 	Case 2:
			 * simple loop order, new loop, no matter which one is current signer
			 */
			last:    "C",
			current: "B",
			queue:   []string{"A", "B", "C"},
			newLoop: true,
			result:  []string{},
		},
		{
			/* 	Case 3:
			 * simple loop order, new loop, missing in last loop
			 */
			last:    "B",
			current: "C",
			queue:   []string{"A", "B", "C"},
			newLoop: true,
			result:  []string{"C"},
		},
		{
			/* 	Case 4:
			 * simple loop order, new loop, two signers missing in last loop
			 */
			last:    "A",
			current: "C",
			queue:   []string{"A", "B", "C"},
			newLoop: true,
			result:  []string{"B", "C"},
		},
	}

	// Run through the test
	for i, tt := range tests {
		// Create the account pool and generate the initial set of all address in addrNames
		accounts := newTesterAccountPool()
		addrQueue := make([]common.Address, len(tt.queue))
		for j, signer := range tt.queue {
			addrQueue[j] = accounts.address(signer)
		}

		extra := HeaderExtra{SignerQueue: addrQueue}
		missing := getSignerMissing(accounts.address(tt.last), accounts.address(tt.current), extra, tt.newLoop)

		signersMissing := make(map[string]bool)
		for _, signer := range missing {
			signersMissing[accounts.name(signer)] = true
		}
		if len(missing) != len(tt.result) {
			t.Errorf("test %d: the length of missing not equal to the length of result, Result is %v not %v  ", i, signersMissing, tt.result)
		}

		for j := 0; j < len(missing); j++ {
			if _, ok := signersMissing[tt.result[j]]; !ok {
				t.Errorf("test %d: the signersMissing is not equal Result is %v not %v ", i, signersMissing, tt.result)
			}
		}

	}
}
