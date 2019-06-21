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

func TestAlien_Penalty(t *testing.T) {
	tests := []struct {
		last    string
		current string
		queue   []string
		number  uint64
		count   uint64
		result  []string // the result of current snapshot
	}{
		{
			/* 	Case 0:
			 *  simple loop order
			 */
			last:    "A",
			current: "B",
			queue:   []string{"A", "B", "C"},
			number:  1,
			count:   7,
			result:  []string{},
		},
		{
			/* 	Case 1:
			 * simple loop order, new loop, no matter which one is current signer
			 */
			last:    "C",
			current: "A",
			queue:   []string{"A", "B", "C"},
			number:  7,
			count:   7,
			result:  []string{},
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
		missing := getSignerMissing(accounts.address(tt.last), accounts.address(tt.current), extra, tt.number, tt.count)

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
