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
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/TTCECO/gttc/common"
	"github.com/TTCECO/gttc/common/hexutil"
	"github.com/TTCECO/gttc/consensus"
	"github.com/TTCECO/gttc/core/types"
	"github.com/TTCECO/gttc/rlp"
)

const (
	mainchainRPCTimeout = 300 // Number of millisecond mainchain rpc connect timeout
)

var (
	// errNotSideChain is returned if main chain try to get main chain client
	errNotSideChain = errors.New("not side chain")

	// errMCRPCCLientEmpty is returned if Side chain not have main chain rpc client
	errMCRPCClientEmpty = errors.New("main chain rpc client empty")

	// errMCPeriodMissing is returned if period from main chain snapshot is zero
	errMCPeriodMissing = errors.New("main chain period is missing")

	// errMCGasChargingInvalid is returned if gas charging info on main chain and side chain header are different
	errMCGasChargingInvalid = errors.New("gas charging info is invalid")
)

// getMainChainSnapshotByTime return snapshot by header time of side chain
// the rpc api will return the snapshot with the same header time (not loopStartTime)
func (a *Alien) getMainChainSnapshotByTime(chain consensus.ChainReader, headerTime uint64, scHash common.Hash) (*Snapshot, error) {
	if !chain.Config().Alien.SideChain {
		return nil, errNotSideChain
	}
	if chain.Config().Alien.MCRPCClient == nil {
		return nil, errMCRPCClientEmpty
	}
	ctx, cancel := context.WithTimeout(context.Background(), mainchainRPCTimeout*time.Millisecond)
	defer cancel()

	var ms *Snapshot
	if err := chain.Config().Alien.MCRPCClient.CallContext(ctx, &ms, "alien_getSnapshotByHeaderTime", headerTime, scHash); err != nil {
		return nil, err
	} else if ms.Period == 0 {
		return nil, errMCPeriodMissing
	}
	return ms, nil
}

// sendTransactionToMainChain
// transaction send to main chain by rpc api, usually is the transaction for notify or confirm seal new block.
func (a *Alien) sendTransactionToMainChain(chain consensus.ChainReader, tx *types.Transaction) (common.Hash, error) {
	if !chain.Config().Alien.SideChain {
		return common.Hash{}, errNotSideChain
	}
	if chain.Config().Alien.MCRPCClient == nil {
		return common.Hash{}, errMCRPCClientEmpty
	}
	ctx, cancel := context.WithTimeout(context.Background(), mainchainRPCTimeout*time.Millisecond)
	defer cancel()

	data, err := rlp.EncodeToBytes(tx)
	if err != nil {
		return common.Hash{}, err
	}
	var hash common.Hash
	if err := chain.Config().Alien.MCRPCClient.CallContext(ctx, &hash, "eth_sendRawTransaction", common.ToHex(data)); err != nil {
		return common.Hash{}, err
	}
	return hash, nil
}

// getTransactionCountFromMainChain
// get nonce from main chain for sendTransactionToMainChain
func (a *Alien) getTransactionCountFromMainChain(chain consensus.ChainReader, account common.Address) (uint64, error) {
	if !chain.Config().Alien.SideChain {
		return 0, errNotSideChain
	}
	if chain.Config().Alien.MCRPCClient == nil {
		return 0, errMCRPCClientEmpty
	}
	ctx, cancel := context.WithTimeout(context.Background(), mainchainRPCTimeout*time.Millisecond)
	defer cancel()

	var result hexutil.Uint64
	if err := chain.Config().Alien.MCRPCClient.CallContext(ctx, &result, "eth_getTransactionCount", account.Hex(), "latest"); err != nil {
		return 0, err
	}
	return uint64(result), nil
}

// getNetVersionFromMainChain
// get network id
func (a *Alien) getNetVersionFromMainChain(chain consensus.ChainReader) (uint64, error) {
	if !chain.Config().Alien.SideChain {
		return 0, errNotSideChain
	}
	if chain.Config().Alien.MCRPCClient == nil {
		return 0, errMCRPCClientEmpty
	}
	ctx, cancel := context.WithTimeout(context.Background(), mainchainRPCTimeout*time.Millisecond)
	defer cancel()

	var result string
	if err := chain.Config().Alien.MCRPCClient.CallContext(ctx, &result, "net_version", "latest"); err != nil {
		return 0, err
	}

	netVersion := new(big.Int)
	err := netVersion.UnmarshalText([]byte(result))
	if err != nil {
		return 0, err
	}
	return netVersion.Uint64(), nil
}
