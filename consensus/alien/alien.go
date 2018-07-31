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
	"errors"
	"math/big"
	"strconv"
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
	checkpointInterval = 3600            // About N hours if config.period is N

	/*
	 *  ufo:version:category:action/data
	 */
	ufoPrefix             = "ufo"
	ufoVersion            = "1"
	ufoCategoryEvent      = "event"
	ufoCategoryLog        = "oplog"
	ufoEventVote          = "vote"
	ufoEventConfirm       = "confirm"
	ufoMinSplitLen        = 3
	posPrefix             = 0
	posVersion            = 1
	posCategory           = 2
	posEventVote          = 3
	posEventConfirm       = 3
	posEventConfirmNumber = 4
)

// Alien delegated-proof-of-stake protocol constants.
var (
	SignerBlockReward      = big.NewInt(5e+18) // Block reward in wei for successfully mining a block first year
	defaultEpochLength     = uint64(3000000)   // Default number of blocks after which vote's period of validity
	defaultBlockPeriod     = uint64(3)         // Default minimum difference between two consecutive block's timestamps
	defaultMaxSignerCount  = uint64(21)        //
	defaultMinVoterBalance = new(big.Int).Mul(big.NewInt(10000), big.NewInt(1e+18))
	extraVanity            = 32                       // Fixed number of extra-data prefix bytes reserved for signer vanity
	extraSeal              = 65                       // Fixed number of extra-data suffix bytes reserved for signer seal
	uncleHash              = types.CalcUncleHash(nil) // Always Keccak256(RLP([])) as uncles are meaningless outside of PoW.
	defaultDifficulty      = big.NewInt(1)            // Default difficulty
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
)

// Vote :
// vote come from custom tx which data like "ufo:1:event:vote"
// Sender of tx is Voter, the tx.to is Candidate
// Stake is the balance of Voter when create this vote
type Vote struct {
	Voter     common.Address
	Candidate common.Address
	Stake     *big.Int
}

// Confirmation :
// confirmation come  from custom tx which data like "ufo:1:event:confirm:123"
// 123 is the block number be confirmed
// Sender of tx is Signer only if the signer in the SignerQueue for block number 123
type Confirmation struct {
	Signer      common.Address
	BlockNumber *big.Int
}

// HeaderExtra is the struct of info in header.Extra[extraVanity:len(header.extra)-extraSeal]
type HeaderExtra struct {
	CurrentBlockConfirmations []Confirmation
	CurrentBlockVotes         []Vote
	ModifyPredecessorVotes    []Vote
	LoopStartTime             uint64
	SignerQueue               []common.Address
	SignerMissing             []common.Address
	ConfirmedBlockNumber      uint64
}

// Alien is the delegated-proof-of-stake consensus engine proposed to support the
// Ethereum testnet following the Ropsten attacks.
type Alien struct {
	config     *params.AlienConfig // Consensus engine configuration parameters
	db         ethdb.Database      // Database to store and retrieve snapshot checkpoints
	recents    *lru.ARCCache       // Snapshots for recent block to speed up reorgs
	signatures *lru.ARCCache       // Signatures of recent blocks to speed up mining
	signer     common.Address      // Ethereum address of the signing key
	signFn     SignerFn            // Signer function to authorize hashes with
	lock       sync.RWMutex        // Protects the signer fields
}

// SignerFn is a signer callback function to request a hash to be signed by a
// backing account.
type SignerFn func(accounts.Account, []byte) ([]byte, error)

// sigHash returns the hash which is used as input for the delegated-proof-of-stake
// signing. It is the hash of the entire header apart from the 65 byte signature
// contained at the end of the extra data.
//
// Note, the method requires the extra data to be at least 65 bytes, otherwise it
// panics. This is done to avoid accidentally using both forms (signature present
// or not), which could be abused to produce different hashes for the same header.
func sigHash(header *types.Header) (hash common.Hash) {
	hasher := sha3.NewKeccak256()
	rlp.Encode(hasher, []interface{}{
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
	})
	hasher.Sum(hash[:0])
	return hash
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
	pubkey, err := crypto.Ecrecover(sigHash(header).Bytes(), signature)
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
	if conf.MinVoterBalance.Uint64() == 0 {
		conf.MinVoterBalance = defaultMinVoterBalance
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
	_, err := a.snapshot(chain, number-1, header.ParentHash, parents, nil)
	if err != nil {
		return err
	}

	// All basic checks passed, verify the seal and return
	return a.verifySeal(chain, header, parents)
}

// snapshot retrieves the authorization snapshot at a given point in time.
func (a *Alien) snapshot(chain consensus.ChainReader, number uint64, hash common.Hash, parents []*types.Header, genesisVotes []*Vote) (*Snapshot, error) {
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

			snap = newSnapshot(a.config, a.signatures, genesis.Hash(), genesisVotes)
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
	snap, err := a.snapshot(chain, number-1, header.ParentHash, parents, nil)
	if err != nil {
		return err
	}

	// Resolve the authorization key and check against signers
	signer, err := ecrecover(header, a.signatures)
	if err != nil {
		return err
	}

	if number > a.config.MaxSignerCount {
		var parent *types.Header
		if len(parents) > 0 {
			parent = parents[len(parents)-1]
		} else {
			parent = chain.GetHeader(header.ParentHash, number-1)
		}
		parentHeaderExtra := HeaderExtra{}

		err = rlp.DecodeBytes(parent.Extra[extraVanity:len(parent.Extra)-extraSeal], &parentHeaderExtra)
		if err != nil {
			log.Info("Fail to decode parent header", "err", err)
		}
		parentSignerMissing := getSignerMissing(parent.Coinbase, header.Coinbase, parentHeaderExtra)

		currentHeaderExtra := HeaderExtra{}
		err = rlp.DecodeBytes(header.Extra[extraVanity:len(header.Extra)-extraSeal], &currentHeaderExtra)
		if err != nil {
			log.Info("Fail to decode header", "err", err)
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

	return nil
}

// Prepare implements consensus.Engine, preparing all the consensus fields of the
// header for running the transactions on top.
func (a *Alien) Prepare(chain consensus.ChainReader, header *types.Header) error {

	// Set the correct difficulty
	header.Difficulty = new(big.Int).Set(defaultDifficulty)
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

// Finalize implements consensus.Engine, ensuring no uncles are set, nor block
// rewards given, and returns the final block.
func (a *Alien) Finalize(chain consensus.ChainReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header, receipts []*types.Receipt) (*types.Block, error) {

	number := header.Number.Uint64()

	// Mix digest is reserved for now, set to empty
	header.MixDigest = common.Hash{}

	// Ensure the timestamp has the correct delay
	parent := chain.GetHeader(header.ParentHash, number-1)
	if parent == nil {
		return nil, consensus.ErrUnknownAncestor
	}
	header.Time = new(big.Int).Add(parent.Time, new(big.Int).SetUint64(a.config.Period))
	if header.Time.Int64() < time.Now().Unix() {
		header.Time = big.NewInt(time.Now().Unix())
	}

	// Ensure the extra data has all it's components
	if len(header.Extra) < extraVanity {
		header.Extra = append(header.Extra, bytes.Repeat([]byte{0x00}, extraVanity-len(header.Extra))...)
	}
	header.Extra = header.Extra[:extraVanity]

	// calculate votes write into header.extra
	currentBlockVotes, modifyPredecessorVotes, currentBlockConfirmations, err := a.processCustomTx(chain, header, state, txs)
	if err != nil {
		return nil, err
	}

	// genesisVotes write direct into snapshot, which number is 1
	var genesisVotes []*Vote
	if number == 1 {
		alreadyVote := make(map[common.Address]struct{})
		for _, voter := range a.config.SelfVoteSigners {

			if _, ok := alreadyVote[voter]; !ok {
				genesisVotes = append(genesisVotes, &Vote{
					Voter:     voter,
					Candidate: voter,
					Stake:     state.GetBalance(voter),
				})
				alreadyVote[voter] = struct{}{}
			}
		}
	}

	// decode extra from last header.extra
	currentHeaderExtra := HeaderExtra{}
	err = rlp.DecodeBytes(parent.Extra[extraVanity:len(parent.Extra)-extraSeal], &currentHeaderExtra)
	if err != nil {
		log.Info("Fail to decode parent header", "err", err)
	}
	// notice : the currentHeaderExtra contain the info of parent HeaderExtra
	currentHeaderExtra.SignerMissing = getSignerMissing(parent.Coinbase, header.Coinbase, currentHeaderExtra)
	currentHeaderExtra.CurrentBlockVotes = currentBlockVotes
	currentHeaderExtra.CurrentBlockConfirmations = currentBlockConfirmations
	currentHeaderExtra.ModifyPredecessorVotes = modifyPredecessorVotes

	// Assemble the voting snapshot to check which votes make sense
	snap, err := a.snapshot(chain, number-1, header.ParentHash, nil, genesisVotes)
	if err != nil {
		return nil, err
	}

	currentHeaderExtra.ConfirmedBlockNumber = snap.getLastConfirmedBlockNumber(currentBlockConfirmations).Uint64()

	// write signerQueue in first header, from self vote signers in genesis block
	if number == 1 {
		currentHeaderExtra.LoopStartTime = a.config.GenesisTimestamp
		for i := 0; i < int(a.config.MaxSignerCount); i++ {
			currentHeaderExtra.SignerQueue = append(currentHeaderExtra.SignerQueue, a.config.SelfVoteSigners[i%len(a.config.SelfVoteSigners)])
		}
	}

	if number%a.config.MaxSignerCount == 0 {
		//currentHeaderExtra.LoopStartTime = header.Time.Uint64()
		currentHeaderExtra.LoopStartTime = currentHeaderExtra.LoopStartTime + a.config.Period*a.config.MaxSignerCount
		// create random signersQueue in currentHeaderExtra by snapshot.Tally
		currentHeaderExtra.SignerQueue = []common.Address{}
		newSignerQueue := snap.getSignerQueue()
		for i := 0; i < int(a.config.MaxSignerCount); i++ {
			currentHeaderExtra.SignerQueue = append(currentHeaderExtra.SignerQueue, newSignerQueue[i%len(newSignerQueue)])
		}
	}

	// encode header.extra
	currentHeaderExtraEnc, err := rlp.EncodeToBytes(currentHeaderExtra)
	if err != nil {
		return nil, err
	}

	header.Extra = append(header.Extra, currentHeaderExtraEnc...)
	header.Extra = append(header.Extra, make([]byte, extraSeal)...)

	// Set the correct difficulty
	header.Difficulty = new(big.Int).Set(defaultDifficulty)

	// Accumulate any block rewards and commit the final state root
	accumulateRewards(chain.Config(), state, header)

	header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number))
	// No uncle block
	header.UncleHash = types.CalcUncleHash(nil)

	// Assemble and return the final block for sealing
	return types.NewBlock(header, txs, nil, receipts), nil
}

// Authorize injects a private key into the consensus engine to mint new blocks with.
func (a *Alien) Authorize(signer common.Address, signFn SignerFn) {
	a.lock.Lock()
	defer a.lock.Unlock()

	a.signer = signer
	a.signFn = signFn
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
	snap, err := a.snapshot(chain, number-1, header.ParentHash, nil, nil)
	if err != nil {
		return nil, err
	}

	if !snap.inturn(signer, header.Time.Uint64()) {
		<-stop
		return nil, errUnauthorized
	}

	// correct the time
	delay := time.Unix(header.Time.Int64(), 0).Sub(time.Now())

	select {
	case <-stop:
		return nil, nil
	case <-time.After(delay):
	}

	// Sign all the things!
	sighash, err := signFn(accounts.Account{Address: signer}, sigHash(header).Bytes())
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
		Version:   "0.1",
		Service:   &API{chain: chain, alien: a},
		Public:    false,
	}}
}

// AccumulateRewards credits the coinbase of the given block with the mining reward.
func accumulateRewards(config *params.ChainConfig, state *state.StateDB, header *types.Header) {
	// Calculate the block reword by year
	blockNumPerYear := secondsPerYear / config.Alien.Period
	yearCount := header.Number.Uint64() / blockNumPerYear
	blockReward := new(big.Int).Rsh(SignerBlockReward, uint(yearCount))
	// rewards for the miner
	state.AddBalance(header.Coinbase, blockReward)
}

// Get the signer missing from last signer till header.Coinbase
func getSignerMissing(lastSigner common.Address, currentSigner common.Address, extra HeaderExtra) []common.Address {

	var signerMissing []common.Address
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
	return signerMissing
}

// Calculate Votes from transaction in this block, write into header.Extra
func (a *Alien) processCustomTx(chain consensus.ChainReader, header *types.Header, state *state.StateDB, txs []*types.Transaction) ([]Vote, []Vote, []Confirmation, error) {
	// if predecessor voter make transaction and vote in this block,
	// just process as vote, do it in snapshot.apply
	var (
		currentBlockConfirmations []Confirmation
		currentBlockVotes         []Vote
		modifyPredecessorVotes    []Vote
		snap                      *Snapshot
		err                       error
		number                    uint64
	)
	number = header.Number.Uint64()
	if number > 1 {
		snap, err = a.snapshot(chain, number-1, header.ParentHash, nil, nil)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	for _, tx := range txs {
		if len(string(tx.Data())) >= len(ufoPrefix) {
			txData := string(tx.Data())
			txDataInfo := strings.Split(txData, ":")
			if len(txDataInfo) >= ufoMinSplitLen {
				if txDataInfo[posPrefix] == ufoPrefix {
					if txDataInfo[posVersion] == ufoVersion {
						// process vote event
						if txDataInfo[posCategory] == ufoCategoryEvent {
							if len(txDataInfo) > ufoMinSplitLen {
								// check is vote or not
								if posEventVote >= ufoMinSplitLen && txDataInfo[posEventVote] == ufoEventVote {
									//a.lock.RLock()
									signer := types.NewEIP155Signer(tx.ChainId())
									voter, _ := types.Sender(signer, tx)
									if state.GetBalance(voter).Cmp(a.config.MinVoterBalance) > 0 {
										currentBlockVotes = append(currentBlockVotes, Vote{
											Voter:     voter,
											Candidate: *tx.To(),
											Stake:     state.GetBalance(voter),
										})
									}
									//a.lock.RUnlock()
									if tx.Value().Cmp(big.NewInt(0)) == 0 {
										// if value is not zero, this vote may influence the balance of tx.To()
										continue
									}
								} else if posEventConfirm >= ufoMinSplitLen && txDataInfo[posEventConfirm] == ufoEventConfirm {
									//a.lock.RLock()
									if len(txDataInfo) >= posEventConfirmNumber {
										confirmedBlockNumber, err := strconv.Atoi(txDataInfo[posEventConfirmNumber])
										if err != nil || number-uint64(confirmedBlockNumber) > a.config.MaxSignerCount || number-uint64(confirmedBlockNumber) < 0 {
											continue
										}
										signer := types.NewEIP155Signer(tx.ChainId())
										confirmer, _ := types.Sender(signer, tx)
										// check if the voter is in block
										confirmedHeader := chain.GetHeaderByNumber(uint64(confirmedBlockNumber))
										if confirmedHeader == nil {
											log.Info("Fail to get confirmedHeader")
											continue
										}
										confirmedHeaderExtra := HeaderExtra{}
										if extraVanity+extraSeal > len(confirmedHeader.Extra) {
											continue
										}
										err = rlp.DecodeBytes(confirmedHeader.Extra[extraVanity:len(confirmedHeader.Extra)-extraSeal], &confirmedHeaderExtra)
										if err != nil {
											log.Info("Fail to decode parent header", "err", err)
											continue
										}
										for _, s := range confirmedHeaderExtra.SignerQueue {
											if s == confirmer {
												currentBlockConfirmations = append(currentBlockConfirmations, Confirmation{
													Signer:      confirmer,
													BlockNumber: big.NewInt(int64(confirmedBlockNumber)),
												})
												break
											}
										}
									}
									//a.lock.RUnlock()
									if tx.Value().Cmp(big.NewInt(0)) == 0 {
										// if value is not zero, this vote may influence the balance of tx.To()
										continue
									}

								} else {
									//todo : other event not vote

								}
							} else {
								// todo : something wrong, leave this transaction to process as normal transaction
							}
						} else if txDataInfo[posCategory] == ufoCategoryLog {
							// todo :
						}
					}
				}
			}
		}

		if number > 1 {
			// process normal transaction which relate to voter
			if tx.Value().Cmp(big.NewInt(0)) > 0 {
				//a.lock.RLock()
				signer := types.NewEIP155Signer(tx.ChainId())
				voter, _ := types.Sender(signer, tx)
				if snap.isVoter(voter) {
					modifyPredecessorVotes = append(modifyPredecessorVotes, Vote{
						Voter:     voter,
						Candidate: common.Address{},
						Stake:     state.GetBalance(voter),
					})
				}
				if snap.isVoter(*tx.To()) {
					modifyPredecessorVotes = append(modifyPredecessorVotes, Vote{
						Voter:     *tx.To(),
						Candidate: common.Address{},
						Stake:     state.GetBalance(*tx.To()),
					})

				}
				//a.lock.RUnlock()
			}
		}

	}

	return currentBlockVotes, modifyPredecessorVotes, currentBlockConfirmations, nil
}
