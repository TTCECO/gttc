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
	"sort"

	"github.com/TTCECO/gttc/common"
)

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

func (s *Snapshot) buildTallySlice() TallySlice {
	var tallySlice TallySlice
	for address, stake := range s.Tally {
		if !candidateNeedPD || s.isCandidate(address) {
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
	return tallySlice
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
		tallySlice := s.buildTallySlice()
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
			maxValidCount := defaultOfficialMaxValidCount
			if maxValidCount > len(tallySlice) {
				maxValidCount = len(tallySlice)
			}
			for i, tallyItem := range tallySlice[defaultOfficialThirdLevelCount:maxValidCount] {
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
	if len(signerSlice) == 0 {
		return nil, errSignerQueueEmpty
	}
	for i := 0; i < int(s.config.MaxSignerCount); i++ {
		topStakeAddress = append(topStakeAddress, signerSlice[i%len(signerSlice)].addr)
	}

	return topStakeAddress, nil

}
