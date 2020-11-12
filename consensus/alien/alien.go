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
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/TTCECO/gttc/accounts"
	"github.com/TTCECO/gttc/common"
	"github.com/TTCECO/gttc/consensus"
	"github.com/TTCECO/gttc/core/state"
	"github.com/TTCECO/gttc/core/types"
	"github.com/TTCECO/gttc/crypto"
	"github.com/TTCECO/gttc/crypto/sha3"
	"github.com/TTCECO/gttc/ethdb"
	"github.com/TTCECO/gttc/log"
	"github.com/TTCECO/gttc/params"
	"github.com/TTCECO/gttc/rlp"
	"github.com/TTCECO/gttc/rpc"
	"github.com/hashicorp/golang-lru"
)

const (
	inMemorySnapshots  = 128             // Number of recent vote snapshots to keep in memory
	inMemorySignatures = 4096            // Number of recent block signatures to keep in memory
	secondsPerYear     = 365 * 24 * 3600 // Number of seconds for one year
	checkpointInterval = 360             // About N hours if config.period is N
	scUnconfirmLoop    = 3               // First count of Loop not send confirm tx to main chain
)

// Alien delegated-proof-of-stake protocol constants.
var (
	totalBlockReward                 = new(big.Int).Mul(big.NewInt(1e+18), big.NewInt(2.5e+8)) // Block reward in wei
	defaultEpochLength               = uint64(201600)                                          // Default number of blocks after which vote's period of validity, About one week if period is 3
	defaultBlockPeriod               = uint64(3)                                               // Default minimum difference between two consecutive block's timestamps
	defaultMaxSignerCount            = uint64(21)                                              //
	minVoterBalance                  = new(big.Int).Mul(big.NewInt(100), big.NewInt(1e+18))
	extraVanity                      = 32                                                    // Fixed number of extra-data prefix bytes reserved for signer vanity
	extraSeal                        = 65                                                    // Fixed number of extra-data suffix bytes reserved for signer seal
	uncleHash                        = types.CalcUncleHash(nil)                              // Always Keccak256(RLP([])) as uncles are meaningless outside of PoW.
	defaultDifficulty                = big.NewInt(1)                                         // Default difficulty
	defaultLoopCntRecalculateSigners = uint64(10)                                            // Default loop count to recreate signers from top tally
	minerRewardPerThousand           = uint64(618)                                           // Default reward for miner in each block from block reward (618/1000)
	candidateNeedPD                  = false                                                 // is new candidate need Proposal & Declare process
	mcNetVersion                     = uint64(0)                                             // the net version of main chain
	mcLoopStartTime                  = uint64(0)                                             // the loopstarttime of main chain
	mcPeriod                         = uint64(0)                                             // the period of main chain
	mcSignerLength                   = uint64(0)                                             // the maxsinger of main chain config
	mcNonce                          = uint64(0)                                             // the current Nonce of coinbase on main chain
	mcTxDefaultGasPrice              = big.NewInt(30000000)                                  // default gas price to build transaction for main chain
	mcTxDefaultGasLimit              = uint64(3000000)                                       // default limit to build transaction for main chain
	proposalDeposit                  = new(big.Int).Mul(big.NewInt(1e+18), big.NewInt(1e+4)) // default current proposalDeposit
	scRentLengthRecommend            = uint64(0)                                             // block number for split each side chain rent fee
)

// Various error messages to mark blocks invalid. These should be private to
// prevent engine specific errors from being referenced in the remainder of the
// codebase, inherently breaking if the engine is swapped out. Please put common
// error types into the consensus package.
var (
	// errUnknownBlock is returned when the list of signers is requested for a block
	// that is not part of the local blockchain.
	errUnknownBlock = errors.New("unknown block")

	// errMissingVanity is returned if a block's extra-data section is shorter than
	// 32 bytes, which is required to store the signer vanity.
	errMissingVanity = errors.New("extra-data 32 byte vanity prefix missing")

	// errMissingSignature is returned if a block's extra-data section doesn't seem
	// to contain a 65 byte secp256k1 signature.
	errMissingSignature = errors.New("extra-data 65 byte suffix signature missing")

	// errInvalidMixDigest is returned if a block's mix digest is non-zero.
	errInvalidMixDigest = errors.New("non-zero mix digest")

	// errInvalidUncleHash is returned if a block contains an non-empty uncle list.
	errInvalidUncleHash = errors.New("non empty uncle hash")

	// ErrInvalidTimestamp is returned if the timestamp of a block is lower than
	// the previous block's timestamp + the minimum block period.
	ErrInvalidTimestamp = errors.New("invalid timestamp")

	// errInvalidVotingChain is returned if an authorization list is attempted to
	// be modified via out-of-range or non-contiguous headers.
	errInvalidVotingChain = errors.New("invalid voting chain")

	// errUnauthorized is returned if a header is signed by a non-authorized entity.
	errUnauthorized = errors.New("unauthorized")

	// errPunishedMissing is returned if a header calculate punished signer is wrong.
	errPunishedMissing = errors.New("punished signer missing")

	// errWaitTransactions is returned if an empty block is attempted to be sealed
	// on an instant chain (0 second period). It's important to refuse these as the
	// block reward is zero, so an empty block just bloats the chain... fast.
	errWaitTransactions = errors.New("waiting for transactions")

	// errUnclesNotAllowed is returned if uncles exists
	errUnclesNotAllowed = errors.New("uncles not allowed")

	// errCreateSignerQueueNotAllowed is returned if called in (block number + 1) % maxSignerCount != 0
	errCreateSignerQueueNotAllowed = errors.New("create signer queue not allowed")

	// errInvalidSignerQueue is returned if verify SignerQueue fail
	errInvalidSignerQueue = errors.New("invalid signer queue")

	// errSignerQueueEmpty is returned if no signer when calculate
	errSignerQueueEmpty = errors.New("signer queue is empty")

	// errGetLastLoopInfoFail is returned if get last loop info fail
	errGetLastLoopInfoFail = errors.New("get last loop info fail")

	// errInvalidNeighborSigner is returned if two neighbor block signed by same miner and time diff less period
	errInvalidNeighborSigner = errors.New("invalid neighbor signer")

	// errMissingGenesisLightConfig is returned only in light syncmode if light config missing
	errMissingGenesisLightConfig = errors.New("light config in genesis is missing")

	// errLastLoopHeaderFail is returned when try to get header of last loop fail
	errLastLoopHeaderFail = errors.New("get last loop header fail")
)

// Alien is the delegated-proof-of-stake consensus engine.
type Alien struct {
	config     *params.AlienConfig // Consensus engine configuration parameters
	db         ethdb.Database      // Database to store and retrieve snapshot checkpoints
	recents    *lru.ARCCache       // Snapshots for recent block to speed up reorgs
	signatures *lru.ARCCache       // Signatures of recent blocks to speed up mining
	signer     common.Address      // Ethereum address of the signing key
	signFn     SignerFn            // Signer function to authorize hashes with
	signTxFn   SignTxFn            // Sign transaction function to sign tx
	lock       sync.RWMutex        // Protects the signer fields
	lcsc       uint64              // Last confirmed side chain
}

// SignerFn is a signer callback function to request a hash to be signed by a
// backing account.
type SignerFn func(accounts.Account, []byte) ([]byte, error)

// SignTxFn is a signTx
type SignTxFn func(accounts.Account, *types.Transaction, *big.Int) (*types.Transaction, error)

// sigHash returns the hash which is used as input for the delegated-proof-of-stake
// signing. It is the hash of the entire header apart from the 65 byte signature
// contained at the end of the extra data.
//
// Note, the method requires the extra data to be at least 65 bytes, otherwise it
// panics. This is done to avoid accidentally using both forms (signature present
// or not), which could be abused to produce different hashes for the same header.
func sigHash(header *types.Header) (hash common.Hash, err error) {
	hasher := sha3.NewKeccak256()
	if err := rlp.Encode(hasher, []interface{}{
		header.ParentHash,
		header.UncleHash,
		header.Coinbase,
		header.Root,
		header.TxHash,
		header.ReceiptHash,
		header.Bloom,
		header.Difficulty,
		header.Number,
		header.GasLimit,
		header.GasUsed,
		header.Time,
		header.Extra[:len(header.Extra)-65], // Yes, this will panic if extra is too short
		header.MixDigest,
		header.Nonce,
	}); err != nil {
		return common.Hash{}, err
	}

	hasher.Sum(hash[:0])
	return hash, nil
}

// ecrecover extracts the Ethereum account address from a signed header.
func ecrecover(header *types.Header, sigcache *lru.ARCCache) (common.Address, error) {
	// If the signature's already cached, return that
	hash := header.Hash()
	if address, known := sigcache.Get(hash); known {
		return address.(common.Address), nil
	}
	// Retrieve the signature from the header extra-data
	if len(header.Extra) < extraSeal {
		return common.Address{}, errMissingSignature
	}
	signature := header.Extra[len(header.Extra)-extraSeal:]

	// Recover the public key and the Ethereum address
	headerSigHash, err := sigHash(header)
	if err != nil {
		return common.Address{}, err
	}
	pubkey, err := crypto.Ecrecover(headerSigHash.Bytes(), signature)
	if err != nil {
		return common.Address{}, err
	}
	var signer common.Address
	copy(signer[:], crypto.Keccak256(pubkey[1:])[12:])

	sigcache.Add(hash, signer)
	return signer, nil
}

// New creates a Alien delegated-proof-of-stake consensus engine with the initial
// signers set to the ones provided by the user.
func New(config *params.AlienConfig, db ethdb.Database) *Alien {
	// Set any missing consensus parameters to their defaults
	conf := *config
	if conf.Epoch == 0 {
		conf.Epoch = defaultEpochLength
	}
	if conf.Period == 0 {
		conf.Period = defaultBlockPeriod
	}
	if conf.MaxSignerCount == 0 {
		conf.MaxSignerCount = defaultMaxSignerCount
	}
	if conf.MinVoterBalance.Uint64() > 0 {
		minVoterBalance = conf.MinVoterBalance
	}

	// Allocate the snapshot caches and create the engine
	recents, _ := lru.NewARC(inMemorySnapshots)
	signatures, _ := lru.NewARC(inMemorySignatures)

	return &Alien{
		config:     &conf,
		db:         db,
		recents:    recents,
		signatures: signatures,
	}
}

// Author implements consensus.Engine, returning the Ethereum address recovered
// from the signature in the header's extra-data section.
func (a *Alien) Author(header *types.Header) (common.Address, error) {
	return ecrecover(header, a.signatures)
}

// VerifyHeader checks whether a header conforms to the consensus rules.
func (a *Alien) VerifyHeader(chain consensus.ChainReader, header *types.Header, seal bool) error {
	return a.verifyHeader(chain, header, nil)
}

// VerifyHeaders is similar to VerifyHeader, but verifies a batch of headers. The
// method returns a quit channel to abort the operations and a results channel to
// retrieve the async verifications (the order is that of the input slice).
func (a *Alien) VerifyHeaders(chain consensus.ChainReader, headers []*types.Header, seals []bool) (chan<- struct{}, <-chan error) {
	abort := make(chan struct{})
	results := make(chan error, len(headers))

	go func() {
		for i, header := range headers {
			err := a.verifyHeader(chain, header, headers[:i])

			select {
			case <-abort:
				return
			case results <- err:
			}
		}
	}()
	return abort, results
}

// verifyHeader checks whether a header conforms to the consensus rules.The
// caller may optionally pass in a batch of parents (ascending order) to avoid
// looking those up from the database. This is useful for concurrently verifying
// a batch of new headers.
func (a *Alien) verifyHeader(chain consensus.ChainReader, header *types.Header, parents []*types.Header) error {
	if header.Number == nil {
		return errUnknownBlock
	}

	// Don't waste time checking blocks from the future
	if header.Time.Cmp(big.NewInt(time.Now().Unix())) > 0 {
		return consensus.ErrFutureBlock
	}

	// Check that the extra-data contains both the vanity and signature
	if len(header.Extra) < extraVanity {
		return errMissingVanity
	}
	if len(header.Extra) < extraVanity+extraSeal {
		return errMissingSignature
	}

	// Ensure that the mix digest is zero as we don't have fork protection currently
	if header.MixDigest != (common.Hash{}) {
		return errInvalidMixDigest
	}
	// Ensure that the block doesn't contain any uncles which are meaningless in PoA
	if header.UncleHash != uncleHash {
		return errInvalidUncleHash
	}

	// All basic checks passed, verify cascading fields
	return a.verifyCascadingFields(chain, header, parents)
}

// verifyCascadingFields verifies all the header fields that are not standalone,
// rather depend on a batch of previous headers. The caller may optionally pass
// in a batch of parents (ascending order) to avoid looking those up from the
// database. This is useful for concurrently verifying a batch of new headers.
func (a *Alien) verifyCascadingFields(chain consensus.ChainReader, header *types.Header, parents []*types.Header) error {
	// The genesis block is the always valid dead-end
	number := header.Number.Uint64()
	if number == 0 {
		return nil
	}
	// Ensure that the block's timestamp isn't too close to it's parent
	var parent *types.Header
	if len(parents) > 0 {
		parent = parents[len(parents)-1]
	} else {
		parent = chain.GetHeader(header.ParentHash, number-1)
	}
	if parent == nil || parent.Number.Uint64() != number-1 || parent.Hash() != header.ParentHash {
		return consensus.ErrUnknownAncestor
	}
	if parent.Time.Uint64() > header.Time.Uint64() {
		return ErrInvalidTimestamp
	}
	// Retrieve the snapshot needed to verify this header and cache it
	_, err := a.snapshot(chain, number-1, header.ParentHash, parents, nil, defaultLoopCntRecalculateSigners)
	if err != nil {
		return err
	}

	// All basic checks passed, verify the seal and return
	return a.verifySeal(chain, header, parents)
}

// snapshot retrieves the authorization snapshot at a given point in time.
func (a *Alien) snapshot(chain consensus.ChainReader, number uint64, hash common.Hash, parents []*types.Header, genesisVotes []*Vote, lcrs uint64) (*Snapshot, error) {
	// Don't keep snapshot for side chain
	//if chain.Config().Alien.SideChain {
	//	return nil, nil
	//}
	// Search for a snapshot in memory or on disk for checkpoints
	var (
		headers []*types.Header
		snap    *Snapshot
	)

	for snap == nil {
		// If an in-memory snapshot was found, use that
		if s, ok := a.recents.Get(hash); ok {
			snap = s.(*Snapshot)
			break
		}
		// If an on-disk checkpoint snapshot can be found, use that
		if number%checkpointInterval == 0 {
			if s, err := loadSnapshot(a.config, a.signatures, a.db, hash); err == nil {
				log.Trace("Loaded voting snapshot from disk", "number", number, "hash", hash)
				snap = s
				break
			}
		}
		// If we're at block zero, make a snapshot
		if number == 0 {
			genesis := chain.GetHeaderByNumber(0)
			if err := a.VerifyHeader(chain, genesis, false); err != nil {
				return nil, err
			}
			a.config.Period = chain.Config().Alien.Period
			snap = newSnapshot(a.config, a.signatures, genesis.Hash(), genesisVotes, lcrs)
			if err := snap.store(a.db); err != nil {
				return nil, err
			}
			log.Trace("Stored genesis voting snapshot to disk")
			break
		}
		// No snapshot for this header, gather the header and move backward
		var header *types.Header
		if len(parents) > 0 {
			// If we have explicit parents, pick from there (enforced)
			header = parents[len(parents)-1]
			if header.Hash() != hash || header.Number.Uint64() != number {
				return nil, consensus.ErrUnknownAncestor
			}
			parents = parents[:len(parents)-1]
		} else {
			// No explicit parents (or no more left), reach out to the database
			header = chain.GetHeader(hash, number)
			if header == nil {
				return nil, consensus.ErrUnknownAncestor
			}
		}
		headers = append(headers, header)
		number, hash = number-1, header.ParentHash
	}
	// Previous snapshot found, apply any pending headers on top of it
	for i := 0; i < len(headers)/2; i++ {
		headers[i], headers[len(headers)-1-i] = headers[len(headers)-1-i], headers[i]
	}

	snap, err := snap.apply(headers)
	if err != nil {
		return nil, err
	}

	a.recents.Add(snap.Hash, snap)

	// If we've generated a new checkpoint snapshot, save to disk
	if snap.Number%checkpointInterval == 0 && len(headers) > 0 {
		if err = snap.store(a.db); err != nil {
			return nil, err
		}
		log.Trace("Stored voting snapshot to disk", "number", snap.Number, "hash", snap.Hash)
	}
	return snap, err
}

// VerifyUncles implements consensus.Engine, always returning an error for any
// uncles as this consensus mechanism doesn't permit uncles.
func (a *Alien) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	if len(block.Uncles()) > 0 {
		return errUnclesNotAllowed
	}
	return nil
}

// VerifySeal implements consensus.Engine, checking whether the signature contained
// in the header satisfies the consensus protocol requirements.
func (a *Alien) VerifySeal(chain consensus.ChainReader, header *types.Header) error {
	return a.verifySeal(chain, header, nil)
}

// verifySeal checks whether the signature contained in the header satisfies the
// consensus protocol requirements. The method accepts an optional list of parent
// headers that aren't yet part of the local blockchain to generate the snapshots
// from.
func (a *Alien) verifySeal(chain consensus.ChainReader, header *types.Header, parents []*types.Header) error {
	// Verifying the genesis block is not supported
	number := header.Number.Uint64()
	if number == 0 {
		return errUnknownBlock
	}
	// Retrieve the snapshot needed to verify this header and cache it
	snap, err := a.snapshot(chain, number-1, header.ParentHash, parents, nil, defaultLoopCntRecalculateSigners)
	if err != nil {
		return err
	}

	// Resolve the authorization key and check against signers
	signer, err := ecrecover(header, a.signatures)
	if err != nil {
		return err
	}

	// check the coinbase == signer
	if header.Number.Cmp(big.NewInt(bugFixBlockNumber)) > 0 {
		if signer != header.Coinbase {
			return errUnauthorized
		}
	}

	if !chain.Config().Alien.SideChain {

		if number > a.config.MaxSignerCount {
			var parent *types.Header
			if len(parents) > 0 {
				parent = parents[len(parents)-1]
			} else {
				parent = chain.GetHeader(header.ParentHash, number-1)
			}
			parentHeaderExtra := HeaderExtra{}
			err = decodeHeaderExtra(a.config, parent.Number, parent.Extra[extraVanity:len(parent.Extra)-extraSeal], &parentHeaderExtra)
			if err != nil {
				log.Info("Fail to decode parent header", "err", err)
				return err
			}
			currentHeaderExtra := HeaderExtra{}
			err = decodeHeaderExtra(a.config, header.Number, header.Extra[extraVanity:len(header.Extra)-extraSeal], &currentHeaderExtra)
			if err != nil {
				log.Info("Fail to decode header", "err", err)
				return err
			}
			// verify signerqueue
			if number%a.config.MaxSignerCount == 0 {
				err := snap.verifySignerQueue(currentHeaderExtra.SignerQueue)
				if err != nil {
					return err
				}

			} else {
				for i := 0; i < int(a.config.MaxSignerCount); i++ {
					if parentHeaderExtra.SignerQueue[i] != currentHeaderExtra.SignerQueue[i] {
						return errInvalidSignerQueue
					}
				}
				if signer == parent.Coinbase && header.Time.Uint64()-parent.Time.Uint64() < chain.Config().Alien.Period {
					return errInvalidNeighborSigner
				}

			}

			// verify missing signer for punish
			var parentSignerMissing []common.Address
			if a.config.IsTrantor(header.Number) {
				var grandParentHeaderExtra HeaderExtra
				if number%a.config.MaxSignerCount == 1 {
					var grandParent *types.Header
					if len(parents) > 1 {
						grandParent = parents[len(parents)-2]
					} else {
						grandParent = chain.GetHeader(parent.ParentHash, number-2)
					}
					if grandParent == nil {
						return errLastLoopHeaderFail
					}
					err := decodeHeaderExtra(a.config, grandParent.Number, grandParent.Extra[extraVanity:len(grandParent.Extra)-extraSeal], &grandParentHeaderExtra)
					if err != nil {
						log.Info("Fail to decode parent header", "err", err)
						return err
					}
				}
				parentSignerMissing = getSignerMissingTrantor(parent.Coinbase, header.Coinbase, &parentHeaderExtra, &grandParentHeaderExtra)
			} else {
				newLoop := false
				if number%a.config.MaxSignerCount == 0 {
					newLoop = true
				}
				parentSignerMissing = getSignerMissing(parent.Coinbase, header.Coinbase, parentHeaderExtra, newLoop)
			}

			if len(parentSignerMissing) != len(currentHeaderExtra.SignerMissing) {
				return errPunishedMissing
			}
			for i, signerMissing := range currentHeaderExtra.SignerMissing {
				if parentSignerMissing[i] != signerMissing {
					return errPunishedMissing
				}
			}
		}

		if !snap.inturn(signer, header.Time.Uint64()) {
			return errUnauthorized
		}
	} else {
		if notice, loopStartTime, period, signerLength, _, err := a.mcSnapshot(chain, signer, header.Time.Uint64()); err != nil {
			return err
		} else {
			mcLoopStartTime = loopStartTime
			mcPeriod = period
			mcSignerLength = signerLength
			// check gas charging
			if notice != nil {
				currentHeaderExtra := HeaderExtra{}
				err = decodeHeaderExtra(a.config, header.Number, header.Extra[extraVanity:len(header.Extra)-extraSeal], &currentHeaderExtra)
				if err != nil {
					return err
				}
				if len(notice.CurrentCharging) != len(currentHeaderExtra.SideChainCharging) {
					return errMCGasChargingInvalid
				} else {
					for _, charge := range currentHeaderExtra.SideChainCharging {
						if v, ok := notice.CurrentCharging[charge.Hash]; !ok {
							return err
						} else {
							if v.Volume != charge.Volume || v.Target != charge.Target {
								return errMCGasChargingInvalid
							}
						}
					}
				}

			}
		}
	}

	return nil
}

// Prepare implements consensus.Engine, preparing all the consensus fields of the
// header for running the transactions on top.
func (a *Alien) Prepare(chain consensus.ChainReader, header *types.Header) error {

	// Set the correct difficulty
	header.Difficulty = new(big.Int).Set(defaultDifficulty)
	number := header.Number.Uint64()
	// Ensure the timestamp has the correct delay
	parent := chain.GetHeader(header.ParentHash, number-1)
	if parent == nil {
		return  consensus.ErrUnknownAncestor
	}
	header.Time = new(big.Int).Add(parent.Time, new(big.Int).SetUint64(a.config.Period))
	if header.Time.Int64() < time.Now().Unix() {
		header.Time = big.NewInt(time.Now().Unix())
	}
	// If now is later than genesis timestamp, skip prepare
	if a.config.GenesisTimestamp < uint64(time.Now().Unix()) {
		return nil
	}
	// Count down for start
	if header.Number.Uint64() == 1 {
		for {
			delay := time.Unix(int64(a.config.GenesisTimestamp-2), 0).Sub(time.Now())
			if delay <= time.Duration(0) {
				log.Info("Ready for seal block", "time", time.Now())
				break
			} else if delay > time.Duration(a.config.Period)*time.Second {
				delay = time.Duration(a.config.Period) * time.Second
			}
			log.Info("Waiting for seal block", "delay", common.PrettyDuration(time.Unix(int64(a.config.GenesisTimestamp-2), 0).Sub(time.Now())))
			select {
			case <-time.After(delay):
				continue
			}
		}
	}

	return nil
}

// get the snapshot info from main chain and check if current signer inturn, if inturn then update the info
func (a *Alien) mcSnapshot(chain consensus.ChainReader, signer common.Address, headerTime uint64) (*CCNotice, uint64, uint64, uint64, uint64, error) {

	if chain.Config().Alien.SideChain {
		chainHash := chain.GetHeaderByNumber(0).ParentHash
		ms, err := a.getMainChainSnapshotByTime(chain, headerTime, chainHash)
		if err != nil {
			return nil, 0, 0, 0, 0, err
		} else if len(ms.Signers) == 0 {
			return nil, 0, 0, 0, 0, errSignerQueueEmpty
		} else if ms.Period == 0 {
			return nil, 0, 0, 0, 0, errMCPeriodMissing
		}

		loopIndex := int((headerTime-ms.LoopStartTime)/ms.Period) % len(ms.Signers)
		if loopIndex >= len(ms.Signers) {
			return nil, 0, 0, 0, 0, errInvalidSignerQueue
		} else if *ms.Signers[loopIndex] != signer {
			return nil, 0, 0, 0, 0, errUnauthorized
		}
		notice := &CCNotice{}
		if mcNotice, ok := ms.SCNoticeMap[chainHash]; ok {
			notice = mcNotice
		}
		return notice, ms.LoopStartTime, ms.Period, uint64(len(ms.Signers)), ms.Number, nil
	}
	return nil, 0, 0, 0, 0, errNotSideChain
}

func (a *Alien) parseNoticeInfo(notice *CCNotice) string {
	// if other notice exist, return string may be more than one
	if notice != nil {
		var charging []string
		for hash := range notice.CurrentCharging {
			charging = append(charging, hash.Hex())
		}
		return strings.Join(charging, "#")
	}
	return ""
}

func (a *Alien) getLastLoopInfo(chain consensus.ChainReader, header *types.Header) (string, error) {
	if chain.Config().Alien.SideChain && mcLoopStartTime != 0 && mcPeriod != 0 && a.config.Period != 0 {
		var loopHeaderInfo []string
		inLastLoop := false
		extraTime := (header.Time.Uint64() - mcLoopStartTime) % (mcPeriod * mcSignerLength)
		for i := uint64(0); i < a.config.MaxSignerCount*2*(mcPeriod/a.config.Period); i++ {
			header = chain.GetHeader(header.ParentHash, header.Number.Uint64()-1)
			if header == nil {
				return "", consensus.ErrUnknownAncestor
			}
			newTime := (header.Time.Uint64() - mcLoopStartTime) % (mcPeriod * mcSignerLength)
			if newTime > extraTime {
				if !inLastLoop {
					inLastLoop = true
				} else {
					break
				}
			}
			extraTime = newTime
			if inLastLoop {
				loopHeaderInfo = append(loopHeaderInfo, fmt.Sprintf("%d#%s", header.Number.Uint64(), header.Coinbase.Hex()))
			}
		}
		if len(loopHeaderInfo) > 0 {
			return strings.Join(loopHeaderInfo, "#"), nil
		}
	}
	return "", errGetLastLoopInfoFail
}

func (a *Alien) mcConfirmBlock(chain consensus.ChainReader, header *types.Header, notice *CCNotice) {

	a.lock.RLock()
	signer, signTxFn := a.signer, a.signTxFn
	a.lock.RUnlock()

	if signer != (common.Address{}) {
		// todo update gaslimit , gasprice ,and get ChainID need to get from mainchain
		if header.Number.Uint64() > a.lcsc && header.Number.Uint64() > a.config.MaxSignerCount*scUnconfirmLoop {
			nonce, err := a.getTransactionCountFromMainChain(chain, signer)
			if err != nil {
				log.Info("Confirm tx sign fail", "err", err)
				return
			}

			lastLoopInfo, err := a.getLastLoopInfo(chain, header)
			if err != nil {
				log.Info("Confirm tx sign fail", "err", err)
				return
			}

			chargingInfo := a.parseNoticeInfo(notice)

			txData := a.buildSCEventConfirmData(chain.GetHeaderByNumber(0).ParentHash, header.Number, header.Time, lastLoopInfo, chargingInfo)
			tx := types.NewTransaction(nonce, header.Coinbase, big.NewInt(0), mcTxDefaultGasLimit, mcTxDefaultGasPrice, txData)

			if mcNetVersion == 0 {
				mcNetVersion, err = a.getNetVersionFromMainChain(chain)
				if err != nil {
					log.Info("Query main chain net version fail", "err", err)
				}
			}

			signedTx, err := signTxFn(accounts.Account{Address: signer}, tx, big.NewInt(int64(mcNetVersion)))
			if err != nil {
				log.Info("Confirm tx sign fail", "err", err)
			}
			txHash, err := a.sendTransactionToMainChain(chain, signedTx)
			if err != nil {
				log.Info("Confirm tx send fail", "err", err)
			} else {
				log.Info("Confirm tx result", "txHash", txHash)
				a.lcsc = header.Number.Uint64()
			}
		}
	}

}

// Finalize implements consensus.Engine, ensuring no uncles are set, nor block
// rewards given, and returns the final block.
func (a *Alien) Finalize(chain consensus.ChainReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header, receipts []*types.Receipt) (*types.Block, error) {


	// Mix digest is reserved for now, set to empty
	header.MixDigest = common.Hash{}
	number := header.Number.Uint64()
	parent := chain.GetHeader(header.ParentHash, number-1)
	if parent == nil {
		return nil, consensus.ErrUnknownAncestor
	}


	// Ensure the extra data has all it's components
	if len(header.Extra) < extraVanity {
		header.Extra = append(header.Extra, bytes.Repeat([]byte{0x00}, extraVanity-len(header.Extra))...)
	}
	header.Extra = header.Extra[:extraVanity]

	// genesisVotes write direct into snapshot, which number is 1
	var genesisVotes []*Vote
	parentHeaderExtra := HeaderExtra{}
	currentHeaderExtra := HeaderExtra{}

	if number == 1 {
		alreadyVote := make(map[common.Address]struct{})
		for _, unPrefixVoter := range a.config.SelfVoteSigners {
			voter := common.Address(unPrefixVoter)
			if _, ok := alreadyVote[voter]; !ok {
				genesisVotes = append(genesisVotes, &Vote{
					Voter:     voter,
					Candidate: voter,
					Stake:     state.GetBalance(voter),
				})
				alreadyVote[voter] = struct{}{}
			}
		}
	} else {
		// decode extra from last header.extra
		err := decodeHeaderExtra(a.config, parent.Number, parent.Extra[extraVanity:len(parent.Extra)-extraSeal], &parentHeaderExtra)
		if err != nil {
			log.Info("Fail to decode parent header", "err", err)
			return nil, err
		}
		currentHeaderExtra.ConfirmedBlockNumber = parentHeaderExtra.ConfirmedBlockNumber
		currentHeaderExtra.SignerQueue = parentHeaderExtra.SignerQueue
		currentHeaderExtra.LoopStartTime = parentHeaderExtra.LoopStartTime

		if a.config.IsTrantor(header.Number) {
			var grandParentHeaderExtra HeaderExtra
			if number%a.config.MaxSignerCount == 1 {
				grandParent := chain.GetHeader(parent.ParentHash, number-2)
				if grandParent == nil {
					return nil, errLastLoopHeaderFail
				}
				err := decodeHeaderExtra(a.config, grandParent.Number, grandParent.Extra[extraVanity:len(grandParent.Extra)-extraSeal], &grandParentHeaderExtra)
				if err != nil {
					log.Info("Fail to decode parent header", "err", err)
					return nil, err
				}
			}
			currentHeaderExtra.SignerMissing = getSignerMissingTrantor(parent.Coinbase, header.Coinbase, &parentHeaderExtra, &grandParentHeaderExtra)
		} else {
			newLoop := false
			if number%a.config.MaxSignerCount == 0 {
				newLoop = true
			}
			currentHeaderExtra.SignerMissing = getSignerMissing(parent.Coinbase, header.Coinbase, parentHeaderExtra, newLoop)
		}

	}

	// Assemble the voting snapshot to check which votes make sense
	snap, err := a.snapshot(chain, number-1, header.ParentHash, nil, genesisVotes, defaultLoopCntRecalculateSigners)
	if err != nil {
		return nil, err
	}
	if !chain.Config().Alien.SideChain {
		// calculate votes write into header.extra
		mcCurrentHeaderExtra, refundGas, err := a.processCustomTx(currentHeaderExtra, chain, header, state, txs, receipts)
		if err != nil {
			return nil, err
		}
		currentHeaderExtra = mcCurrentHeaderExtra
		currentHeaderExtra.ConfirmedBlockNumber = snap.getLastConfirmedBlockNumber(currentHeaderExtra.CurrentBlockConfirmations).Uint64()
		// write signerQueue in first header, from self vote signers in genesis block
		if number == 1 {
			currentHeaderExtra.LoopStartTime = a.config.GenesisTimestamp
			if len(a.config.SelfVoteSigners) > 0 {
				for i := 0; i < int(a.config.MaxSignerCount); i++ {
					currentHeaderExtra.SignerQueue = append(currentHeaderExtra.SignerQueue, common.Address(a.config.SelfVoteSigners[i%len(a.config.SelfVoteSigners)]))
				}
			}
		} else if number%a.config.MaxSignerCount == 0 {
			//currentHeaderExtra.LoopStartTime = header.Time.Uint64()
			currentHeaderExtra.LoopStartTime = currentHeaderExtra.LoopStartTime + a.config.Period*a.config.MaxSignerCount
			// create random signersQueue in currentHeaderExtra by snapshot.Tally
			currentHeaderExtra.SignerQueue = []common.Address{}
			newSignerQueue, err := snap.createSignerQueue()
			if err != nil {
				return nil, err
			}
			currentHeaderExtra.SignerQueue = newSignerQueue
		}

		// Accumulate any block rewards and commit the final state root
		if err := accumulateRewards(chain.Config(), state, header, snap, refundGas); err != nil {
			return nil, errUnauthorized
		}
	} else {
		// use currentHeaderExtra.SignerQueue as signer queue
		currentHeaderExtra.SignerQueue = append([]common.Address{header.Coinbase}, parentHeaderExtra.SignerQueue...)
		if len(currentHeaderExtra.SignerQueue) > int(a.config.MaxSignerCount) {
			currentHeaderExtra.SignerQueue = currentHeaderExtra.SignerQueue[:int(a.config.MaxSignerCount)]
		}
		sideChainRewards(chain.Config(), state, header, snap)
	}
	// encode header.extra
	currentHeaderExtraEnc, err := encodeHeaderExtra(a.config, header.Number, currentHeaderExtra)
	if err != nil {
		return nil, err
	}

	header.Extra = append(header.Extra, currentHeaderExtraEnc...)
	header.Extra = append(header.Extra, make([]byte, extraSeal)...)

	// Set the correct difficulty
	header.Difficulty = new(big.Int).Set(defaultDifficulty)

	header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number))
	// No uncle block
	header.UncleHash = types.CalcUncleHash(nil)

	// Assemble and return the final block for sealing
	return types.NewBlock(header, txs, nil, receipts), nil
}

// Authorize injects a private key into the consensus engine to mint new blocks with.
func (a *Alien) Authorize(signer common.Address, signFn SignerFn, signTxFn SignTxFn) {
	a.lock.Lock()
	defer a.lock.Unlock()

	a.signer = signer
	a.signFn = signFn
	a.signTxFn = signTxFn
}

// ApplyGenesis
func (a *Alien) ApplyGenesis(chain consensus.ChainReader, genesisHash common.Hash) error {
	if a.config.LightConfig != nil {
		var genesisVotes []*Vote
		alreadyVote := make(map[common.Address]struct{})
		for _, unPrefixVoter := range a.config.SelfVoteSigners {
			voter := common.Address(unPrefixVoter)
			if genesisAccount, ok := a.config.LightConfig.Alloc[unPrefixVoter]; ok {
				if _, ok := alreadyVote[voter]; !ok {
					stake := new(big.Int)
					stake.UnmarshalText([]byte(genesisAccount.Balance))
					genesisVotes = append(genesisVotes, &Vote{
						Voter:     voter,
						Candidate: voter,
						Stake:     stake,
					})
					alreadyVote[voter] = struct{}{}
				}
			}
		}
		// Assemble the voting snapshot to check which votes make sense
		if _, err := a.snapshot(chain, 0, genesisHash, nil, genesisVotes, defaultLoopCntRecalculateSigners); err != nil {
			return err
		}
		return nil
	}
	return errMissingGenesisLightConfig
}

// Seal implements consensus.Engine, attempting to create a sealed block using
// the local signing credentials.
func (a *Alien) Seal(chain consensus.ChainReader, block *types.Block, stop <-chan struct{}) (*types.Block, error) {
	header := block.Header()

	// Sealing the genesis block is not supported
	number := header.Number.Uint64()
	if number == 0 {
		return nil, errUnknownBlock
	}

	// For 0-period chains, refuse to seal empty blocks (no reward but would spin sealing)
	if a.config.Period == 0 && len(block.Transactions()) == 0 {
		return nil, errWaitTransactions
	}
	// Don't hold the signer fields for the entire sealing procedure
	a.lock.RLock()
	signer, signFn := a.signer, a.signFn
	a.lock.RUnlock()

	// Bail out if we're unauthorized to sign a block
	snap, err := a.snapshot(chain, number-1, header.ParentHash, nil, nil, defaultLoopCntRecalculateSigners)
	if err != nil {
		return nil, err
	}

	if !chain.Config().Alien.SideChain {
		if !snap.inturn(signer, header.Time.Uint64()) {
			<-stop
			return nil, errUnauthorized
		}
	} else {
		if notice, loopStartTime, period, signerLength, _, err := a.mcSnapshot(chain, signer, header.Time.Uint64()); err != nil {
			<-stop
			return nil, err
		} else {
			mcLoopStartTime = loopStartTime
			mcPeriod = period
			mcSignerLength = signerLength
			if notice != nil {
				// rebuild the header.Extra for gas charging
				currentHeaderExtra := HeaderExtra{}
				if err = decodeHeaderExtra(a.config, header.Number, header.Extra[extraVanity:len(header.Extra)-extraSeal], &currentHeaderExtra); err != nil {
					return nil, err
				}
				for _, charge := range notice.CurrentCharging {
					currentHeaderExtra.SideChainCharging = append(currentHeaderExtra.SideChainCharging, charge)
				}
				currentHeaderExtraEnc, err := encodeHeaderExtra(a.config, header.Number, currentHeaderExtra)
				if err != nil {
					return nil, err
				}
				header.Extra = header.Extra[:extraVanity]
				header.Extra = append(header.Extra, currentHeaderExtraEnc...)
				header.Extra = append(header.Extra, make([]byte, extraSeal)...)
			}
			// send tx to main chain to confirm this block
			a.mcConfirmBlock(chain, header, notice)
		}
	}

	// correct the time
	delay := time.Unix(header.Time.Int64(), 0).Sub(time.Now())

	select {
	case <-stop:
		return nil, nil
	case <-time.After(delay):
	}

	// Sign all the things!
	headerSigHash, err := sigHash(header)
	if err != nil {
		return nil, err
	}

	sighash, err := signFn(accounts.Account{Address: signer}, headerSigHash.Bytes())
	if err != nil {
		return nil, err
	}

	copy(header.Extra[len(header.Extra)-extraSeal:], sighash)

	return block.WithSeal(header), nil
}

// CalcDifficulty is the difficulty adjustment algorithm. It returns the difficulty
// that a new block should have based on the previous blocks in the chain and the
// current signer.
func (a *Alien) CalcDifficulty(chain consensus.ChainReader, time uint64, parent *types.Header) *big.Int {

	return new(big.Int).Set(defaultDifficulty)
}

// APIs implements consensus.Engine, returning the user facing RPC API to allow
// controlling the signer voting.
func (a *Alien) APIs(chain consensus.ChainReader) []rpc.API {
	return []rpc.API{{
		Namespace: "alien",
		Version:   ufoVersion,
		Service:   &API{chain: chain, alien: a},
		Public:    false,
	}}
}

func sideChainRewards(config *params.ChainConfig, state *state.StateDB, header *types.Header, snap *Snapshot) {
	// vanish gas fee
	gasUsed := new(big.Int).SetUint64(header.GasUsed)
	if state.GetBalance(header.Coinbase).Cmp(gasUsed) >= 0 {
		state.SubBalance(header.Coinbase, gasUsed)
	}
	// gas charging
	for target, volume := range snap.calculateGasCharging() {
		state.AddBalance(target, volume)
	}
}

// AccumulateRewards credits the coinbase of the given block with the mining reward.
func accumulateRewards(config *params.ChainConfig, state *state.StateDB, header *types.Header, snap *Snapshot, refundGas RefundGas) error {
	// Calculate the block reword by year
	blockNumPerYear := secondsPerYear / config.Alien.Period
	initSignerBlockReward := new(big.Int).Div(totalBlockReward, big.NewInt(int64(2*blockNumPerYear)))
	yearCount := header.Number.Uint64() / blockNumPerYear
	blockReward := new(big.Int).Rsh(initSignerBlockReward, uint(yearCount))

	minerReward := new(big.Int).Set(blockReward)
	minerReward.Mul(minerReward, new(big.Int).SetUint64(snap.MinerReward))
	minerReward.Div(minerReward, big.NewInt(1000)) // cause the reward is calculate by cnt per thousand

	votersReward := blockReward.Sub(blockReward, minerReward)

	// rewards for the voters
	voteRewardMap, err := snap.calculateVoteReward(header.Coinbase, votersReward)
	if err != nil {
		return err
	}
	for voter, reward := range voteRewardMap {
		state.AddBalance(voter, reward)
	}

	// calculate for proposal refund
	for proposer, refund := range snap.calculateProposalRefund() {
		state.AddBalance(proposer, refund)
	}

	scReward, minerLeft := snap.calculateSCReward(minerReward)
	minerReward.Set(minerLeft)
	// rewards for the side chain coinbase
	for scCoinbase, reward := range scReward {
		state.AddBalance(scCoinbase, reward)
	}
	// refund gas for custom txs
	for sender, gas := range refundGas {
		state.AddBalance(sender, gas)
		minerReward.Sub(minerReward, gas)
	}

	// rewards for the miner, check minerReward value for refund gas
	if minerReward.Cmp(big.NewInt(0)) > 0 {
		state.AddBalance(header.Coinbase, minerReward)
	}

	return nil
}

// Get the signer missing from last signer till header.Coinbase
func getSignerMissing(lastSigner common.Address, currentSigner common.Address, extra HeaderExtra, newLoop bool) []common.Address {

	var signerMissing []common.Address

	if newLoop {
		for i, qlen := 0, len(extra.SignerQueue); i < len(extra.SignerQueue); i++ {
			if lastSigner == extra.SignerQueue[qlen-1-i] {
				break
			} else {
				signerMissing = append(signerMissing, extra.SignerQueue[qlen-1-i])
			}
		}
	} else {
		recordMissing := false
		for _, signer := range extra.SignerQueue {
			if signer == lastSigner {
				recordMissing = true
				continue
			}
			if signer == currentSigner {
				break
			}
			if recordMissing {
				signerMissing = append(signerMissing, signer)
			}
		}

	}

	return signerMissing
}

// Get the signer missing from last signer till header.Coinbase
func getSignerMissingTrantor(lastSigner common.Address, currentSigner common.Address, extra *HeaderExtra, gpExtra *HeaderExtra) []common.Address {

	var signerMissing []common.Address
	signerQueue := append(extra.SignerQueue, extra.SignerQueue...)
	if gpExtra != nil {
		for i, v := range gpExtra.SignerQueue {
			if v == lastSigner {
				signerQueue[i] = lastSigner
				signerQueue = signerQueue[i:]
				break
			}
		}
	}

	recordMissing := false
	for _, signer := range signerQueue {
		if !recordMissing && signer == lastSigner {
			recordMissing = true
			continue
		}
		if recordMissing && signer == currentSigner {
			break
		}
		if recordMissing {
			signerMissing = append(signerMissing, signer)
		}
	}

	return signerMissing

}
