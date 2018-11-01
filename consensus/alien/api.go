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
	"github.com/TTCECO/gttc/common"
	"github.com/TTCECO/gttc/consensus"
	"github.com/TTCECO/gttc/core/types"
	"github.com/TTCECO/gttc/rpc"
)

// API is a user facing RPC API to allow controlling the signer and voting
// mechanisms of the delegated-proof-of-stake scheme.
type API struct {
	chain consensus.ChainReader
	alien *Alien
}

// GetSnapshot retrieves the state snapshot at a given block.
func (api *API) GetSnapshot(number *rpc.BlockNumber) (*Snapshot, error) {
	// Retrieve the requested block number (or current if none requested)
	var header *types.Header
	if number == nil || *number == rpc.LatestBlockNumber {
		header = api.chain.CurrentHeader()
	} else {
		header = api.chain.GetHeaderByNumber(uint64(number.Int64()))
	}
	// Ensure we have an actually valid block and return its snapshot
	if header == nil {
		return nil, errUnknownBlock
	}
	return api.alien.snapshot(api.chain, header.Number.Uint64(), header.Hash(), nil, nil, defaultLoopCntRecalculateSigners)

}

// GetSnapshotAtHash retrieves the state snapshot at a given block.
func (api *API) GetSnapshotAtHash(hash common.Hash) (*Snapshot, error) {
	header := api.chain.GetHeaderByHash(hash)
	if header == nil {
		return nil, errUnknownBlock
	}
	return api.alien.snapshot(api.chain, header.Number.Uint64(), header.Hash(), nil, nil, defaultLoopCntRecalculateSigners)
}

// GetSnapshotAtNumber retrieves the state snapshot at a given block.
func (api *API) GetSnapshotAtNumber(number uint64) (*Snapshot, error) {
	header := api.chain.GetHeaderByNumber(number)
	if header == nil {
		return nil, errUnknownBlock
	}
	return api.alien.snapshot(api.chain, header.Number.Uint64(), header.Hash(), nil, nil, defaultLoopCntRecalculateSigners)
}

// GetSnapshotByHeaderTime retrieves the state snapshot by timestamp of header.
// snapshot.header.time <= targetTime < snapshot.header.time + period
func (api *API) GetSnapshotByHeaderTime(targetTime uint64) (*Snapshot, error) {
	period := api.chain.Config().Alien.Period
	header := api.chain.CurrentHeader()
	if header == nil || targetTime > header.Time.Uint64()+period {
		return nil, errUnknownBlock
	}
	minN := uint64(0)
	maxN := header.Number.Uint64()
	for {
		if targetTime >= header.Time.Uint64() && targetTime < header.Time.Uint64()+period {
			return api.alien.snapshot(api.chain, header.Number.Uint64(), header.Hash(), nil, nil, defaultLoopCntRecalculateSigners)
		} else {
			if maxN == minN || maxN == minN+1 {
				break
			}
			// calculate next number
			nextN := uint64(int64(header.Number.Uint64()) + (int64(targetTime)-int64(header.Time.Uint64()))/int64(period))
			if nextN >= maxN || nextN <= minN {
				nextN = (maxN + minN) / 2
			}
			// get new header
			header = api.chain.GetHeaderByNumber(nextN)
			if header == nil {
				break
			}
			// update maxN & minN
			if header.Time.Uint64() >= targetTime {
				if header.Number.Uint64() < maxN {
					maxN = header.Number.Uint64()
				}
			} else if header.Time.Uint64() <= targetTime {
				if header.Number.Uint64() > minN {
					minN = header.Number.Uint64()
				}
			}
		}
	}
	return nil, errUnknownBlock
}
