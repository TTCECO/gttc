// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/TTCECO/gttc/common"
	"github.com/TTCECO/gttc/common/hexutil"
	"github.com/TTCECO/gttc/common/math"
	"github.com/TTCECO/gttc/core/rawdb"
	"github.com/TTCECO/gttc/core/state"
	"github.com/TTCECO/gttc/core/types"
	"github.com/TTCECO/gttc/ethdb"
	"github.com/TTCECO/gttc/log"
	"github.com/TTCECO/gttc/params"
	"github.com/TTCECO/gttc/rlp"
)

//go:generate gencodec -type Genesis -field-override genesisSpecMarshaling -out gen_genesis.go
//go:generate gencodec -type GenesisAccount -field-override genesisAccountMarshaling -out gen_genesis_account.go

var errGenesisNoConfig = errors.New("genesis has no chain configuration")

// Genesis specifies the header fields, state of a genesis block. It also defines hard
// fork switch-over blocks through the chain configuration.
type Genesis struct {
	Config     *params.ChainConfig `json:"config"`
	Nonce      uint64              `json:"nonce"`
	Timestamp  uint64              `json:"timestamp"`
	ExtraData  []byte              `json:"extraData"`
	GasLimit   uint64              `json:"gasLimit"   gencodec:"required"`
	Difficulty *big.Int            `json:"difficulty" gencodec:"required"`
	Mixhash    common.Hash         `json:"mixHash"`
	Coinbase   common.Address      `json:"coinbase"`
	Alloc      GenesisAlloc        `json:"alloc"      gencodec:"required"`

	// These fields are used for consensus tests. Please don't use them
	// in actual genesis blocks.
	Number     uint64      `json:"number"`
	GasUsed    uint64      `json:"gasUsed"`
	ParentHash common.Hash `json:"parentHash"`
}

// GenesisAlloc specifies the initial state that is part of the genesis block.
type GenesisAlloc map[common.Address]GenesisAccount

func (ga *GenesisAlloc) UnmarshalJSON(data []byte) error {
	m := make(map[common.UnprefixedAddress]GenesisAccount)
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	*ga = make(GenesisAlloc)
	for addr, a := range m {
		(*ga)[common.Address(addr)] = a
	}
	return nil
}

// GenesisAccount is an account in the state of the genesis block.
type GenesisAccount struct {
	Code       []byte                      `json:"code,omitempty"`
	Storage    map[common.Hash]common.Hash `json:"storage,omitempty"`
	Balance    *big.Int                    `json:"balance" gencodec:"required"`
	Nonce      uint64                      `json:"nonce,omitempty"`
	PrivateKey []byte                      `json:"secretKey,omitempty"` // for tests
}

// field type overrides for gencodec
type genesisSpecMarshaling struct {
	Nonce      math.HexOrDecimal64
	Timestamp  math.HexOrDecimal64
	ExtraData  hexutil.Bytes
	GasLimit   math.HexOrDecimal64
	GasUsed    math.HexOrDecimal64
	Number     math.HexOrDecimal64
	Difficulty *math.HexOrDecimal256
	Alloc      map[common.UnprefixedAddress]GenesisAccount
}

type genesisAccountMarshaling struct {
	Code       hexutil.Bytes
	Balance    *math.HexOrDecimal256
	Nonce      math.HexOrDecimal64
	Storage    map[storageJSON]storageJSON
	PrivateKey hexutil.Bytes
}

// storageJSON represents a 256 bit byte array, but allows less than 256 bits when
// unmarshaling from hex.
type storageJSON common.Hash

func (h *storageJSON) UnmarshalText(text []byte) error {
	text = bytes.TrimPrefix(text, []byte("0x"))
	if len(text) > 64 {
		return fmt.Errorf("too many hex characters in storage key/value %q", text)
	}
	offset := len(h) - len(text)/2 // pad on the left
	if _, err := hex.Decode(h[offset:], text); err != nil {
		fmt.Println(err)
		return fmt.Errorf("invalid hex storage key/value %q", text)
	}
	return nil
}

func (h storageJSON) MarshalText() ([]byte, error) {
	return hexutil.Bytes(h[:]).MarshalText()
}

// GenesisMismatchError is raised when trying to overwrite an existing
// genesis block with an incompatible one.
type GenesisMismatchError struct {
	Stored, New common.Hash
}

func (e *GenesisMismatchError) Error() string {
	return fmt.Sprintf("database already contains an incompatible genesis block (have %x, new %x)", e.Stored[:8], e.New[:8])
}

// SetupGenesisBlock writes or updates the genesis block in db.
// The block that will be used is:
//
//                          genesis == nil       genesis != nil
//                       +------------------------------------------
//     db has no genesis |  main-net default  |  genesis
//     db has genesis    |  from DB           |  genesis (if compatible)
//
// The stored chain configuration will be updated if it is compatible (i.e. does not
// specify a fork block below the local head block). In case of a conflict, the
// error is a *params.ConfigCompatError and the new, unwritten config is returned.
//
// The returned chain configuration is never nil.
func SetupGenesisBlock(db ethdb.Database, genesis *Genesis) (*params.ChainConfig, common.Hash, error) {
	if genesis != nil && genesis.Config == nil {
		return params.AllEthashProtocolChanges, common.Hash{}, errGenesisNoConfig
	}

	// Just commit the new block if there is no stored genesis block.
	stored := rawdb.ReadCanonicalHash(db, 0)
	if (stored == common.Hash{}) {
		if genesis == nil {
			log.Info("Writing default main-net genesis block")
			genesis = DefaultGenesisBlock()
		} else {
			log.Info("Writing custom genesis block")
		}
		block, err := genesis.Commit(db)
		return genesis.Config, block.Hash(), err
	}

	// Check whether the genesis block is already written.
	if genesis != nil {
		hash := genesis.ToBlock(nil).Hash()
		if hash != stored {
			return genesis.Config, hash, &GenesisMismatchError{stored, hash}
		}
	}

	// Get the existing chain configuration.
	newcfg := genesis.configOrDefault(stored)
	storedcfg := rawdb.ReadChainConfig(db, stored)
	if storedcfg == nil {
		log.Warn("Found genesis block without chain config")
		rawdb.WriteChainConfig(db, stored, newcfg)
		return newcfg, stored, nil
	}
	// Special case: don't change the existing config of a non-mainnet chain if no new
	// config is supplied. These chains would get AllProtocolChanges (and a compat error)
	// if we just continued here.
	if genesis == nil && stored != params.MainnetGenesisHash {
		return storedcfg, stored, nil
	}

	// Check config compatibility and write the config. Compatibility errors
	// are returned to the caller unless we're already at block zero.
	height := rawdb.ReadHeaderNumber(db, rawdb.ReadHeadHeaderHash(db))
	if height == nil {
		return newcfg, stored, fmt.Errorf("missing block number for head header hash")
	}
	compatErr := storedcfg.CheckCompatible(newcfg, *height)
	if compatErr != nil && *height != 0 && compatErr.RewindTo != 0 {
		return newcfg, stored, compatErr
	}
	rawdb.WriteChainConfig(db, stored, newcfg)
	return newcfg, stored, nil
}

func (g *Genesis) configOrDefault(ghash common.Hash) *params.ChainConfig {
	switch {
	case g != nil:
		return g.Config
	case ghash == params.MainnetGenesisHash:
		return params.MainnetChainConfig
	case ghash == params.TestnetGenesisHash:
		return params.TestnetChainConfig
	default:
		return params.AllEthashProtocolChanges
	}
}

// ToBlock creates the genesis block and writes state of a genesis specification
// to the given database (or discards it if nil).
func (g *Genesis) ToBlock(db ethdb.Database) *types.Block {
	if db == nil {
		db = ethdb.NewMemDatabase()
	}
	statedb, _ := state.New(common.Hash{}, state.NewDatabase(db))
	for addr, account := range g.Alloc {
		statedb.AddBalance(addr, account.Balance)
		statedb.SetCode(addr, account.Code)
		statedb.SetNonce(addr, account.Nonce)
		for key, value := range account.Storage {
			statedb.SetState(addr, key, value)
		}
	}
	root := statedb.IntermediateRoot(false)
	head := &types.Header{
		Number:     new(big.Int).SetUint64(g.Number),
		Nonce:      types.EncodeNonce(g.Nonce),
		Time:       new(big.Int).SetUint64(g.Timestamp),
		ParentHash: g.ParentHash,
		Extra:      g.ExtraData,
		GasLimit:   g.GasLimit,
		GasUsed:    g.GasUsed,
		Difficulty: g.Difficulty,
		MixDigest:  g.Mixhash,
		Coinbase:   g.Coinbase,
		Root:       root,
	}
	if g.GasLimit == 0 {
		head.GasLimit = params.GenesisGasLimit
	}
	if g.Difficulty == nil {
		head.Difficulty = params.GenesisDifficulty
	}
	statedb.Commit(false)
	statedb.Database().TrieDB().Commit(root, true)

	return types.NewBlock(head, nil, nil, nil)
}

// Commit writes the block and state of a genesis specification to the database.
// The block is committed as the canonical head block.
func (g *Genesis) Commit(db ethdb.Database) (*types.Block, error) {
	block := g.ToBlock(db)
	if block.Number().Sign() != 0 {
		return nil, fmt.Errorf("can't commit genesis block with number > 0")
	}
	rawdb.WriteTd(db, block.Hash(), block.NumberU64(), g.Difficulty)
	rawdb.WriteBlock(db, block)
	rawdb.WriteReceipts(db, block.Hash(), block.NumberU64(), nil)
	rawdb.WriteCanonicalHash(db, block.Hash(), block.NumberU64())
	rawdb.WriteHeadBlockHash(db, block.Hash())
	rawdb.WriteHeadHeaderHash(db, block.Hash())

	config := g.Config
	if config == nil {
		config = params.AllEthashProtocolChanges
	}
	rawdb.WriteChainConfig(db, block.Hash(), config)
	return block, nil
}

// MustCommit writes the genesis block and state to db, panicking on error.
// The block is committed as the canonical head block.
func (g *Genesis) MustCommit(db ethdb.Database) *types.Block {
	block, err := g.Commit(db)
	if err != nil {
		panic(err)
	}
	return block
}

// GenesisBlockForTesting creates and writes a block in which addr has the given wei balance.
func GenesisBlockForTesting(db ethdb.Database, addr common.Address, balance *big.Int) *types.Block {
	g := Genesis{Alloc: GenesisAlloc{addr: {Balance: balance}}}
	return g.MustCommit(db)
}

// DefaultGenesisBlock returns the Ethereum main net genesis block.
func DefaultGenesisBlock() *Genesis {
	mainnetAlloc := make(GenesisAlloc, 50)
	for _, addr := range params.MainnetChainConfig.Alien.SelfVoteSigners {
		balance, _ := new(big.Int).SetString("400000000000000000", 16)
		mainnetAlloc[common.Address(addr)] = GenesisAccount{Balance: balance}
	}

	balance, _ := new(big.Int).SetString("26c566f0a2b77a000000000", 16)
	mainnetAlloc[common.HexToAddress("t0bce13d77339971d1f5f00c38f523ba7ee44c95ed")] = GenesisAccount{Balance: balance}

	return &Genesis{
		Config:     params.MainnetChainConfig,
		Timestamp:  1554004800,
		Nonce:      0,
		ExtraData:  hexutil.MustDecode("0x436f6769746f206572676f2073756d2e20457863657074206f7572206f776e2074686f75676874732c207468657265206973206e6f7468696e67206162736f6c7574656c7920696e206f757220706f7765722e20596f75206e656564206368616f7320696e20796f757220736f756c20746f206769766520626972746820746f20612064616e63696e6720737461722e20416c6561206961637461206573742e2020416c6c2063726564697420676f657320746f20746865207465616d3a2053686968616f2047756f2c2050656e67204c69752c205969204d6f2c204368617365204368616e672c2059697869616f2057616e672c20436520476f6e672c204865205a68616e672c2059756e6a69204d612c204a69652057752c205869616e6779616e672057616e672c204368656e687569204c752c204368656e6c69616e672057616e672c205765692050616e2c205175616e205975616e2c20577571696f6e67204c69752c204368616f204368656e2c204a756e6875612046616e2c20536875616e67205a68616f2c2059756e204c762c204a696e676a69652048652c204a696e68752044696e672c2059616e672048616e2c2053756d656920486f6e672c204c69616e67205a68616e672c204a75616e205a68656e672c204a69616e6a69616f204875616e672c204c756e205169616e2c205869616f7969204368656e2c205975666569205a68616e672c20516920416e2c205a6869636f6e67205975616e2c2059696e677975652053752c2048616e205a68616e672c204a69616e677765692057616e672c2046656970656e67204875616e672c205975746f6e6720446f6e672c2054656e67204d612c205169616e6c6569205368692c2059756e7869616f204c692c2052756971696e67205975652c205068696c6c6970204368756e2c205469616e7869616e672059752c204e61205a68616e672c2053687561692057616e672c2048616966656e672059616e2c204368656e6768616f2059696e2c2048656e67205a686f752c20536875616e67205a68616e672c204c696e7a68656e205869652c204b657368756e2058752c204a756e79692046616e672c204c696e66656e67204c692c20596f6e676c696e204c6920616e6420427269616e204368656f6e672e205370656369616c207468616e6b7320746f2053696d6f6e204b696d2c205279616e204b696d2c2053686f756a69205a686f752c205975616e205a68616e672c205468616e68204e677579656e2c204a69616e204361692c20486f6e677765692043616f2c205374656e204c61757265797373656e732e"),
		GasLimit:   4700000,
		Difficulty: big.NewInt(1),
		Alloc:      mainnetAlloc,
	}
}

// DefaultTestnetGenesisBlock returns the Ropsten network genesis block.
func DefaultTestnetGenesisBlock() *Genesis {

	testnetAlloc := make(GenesisAlloc, 3)
	balance1, _ := new(big.Int).SetString("40000000000000000000000", 16)
	testnetAlloc[common.HexToAddress("t0be6865ffcbbe5f9746bef5c84b912f2ad9e52075")] = GenesisAccount{Balance: balance1}

	balance2, _ := new(big.Int).SetString("40000000000000000000000", 16)
	testnetAlloc[common.HexToAddress("t04909b4e54395de9e313ad8a2254fe2dcda99e91c")] = GenesisAccount{Balance: balance2}

	balance3, _ := new(big.Int).SetString("26c566f0a2b77a000000000", 16)
	testnetAlloc[common.HexToAddress("t0a034350c8e80eb4d15ac62310657b29c711bb3d5")] = GenesisAccount{Balance: balance3}

	return &Genesis{
		Config:     params.TestnetChainConfig,
		Timestamp:  1554004800,
		Nonce:      0,
		ExtraData:  hexutil.MustDecode("0x436f6769746f206572676f2073756d2e20457863657074206f7572206f776e2074686f75676874732c207468657265206973206e6f7468696e67206162736f6c7574656c7920696e206f757220706f7765722e20596f75206e656564206368616f7320696e20796f757220736f756c20746f206769766520626972746820746f20612064616e63696e6720737461722e20416c6561206961637461206573742e2020416c6c2063726564697420676f657320746f20746865207465616d3a2053686968616f2047756f2c2050656e67204c69752c205969204d6f2c204368617365204368616e672c2059697869616f2057616e672c20436520476f6e672c204865205a68616e672c2059756e6a69204d612c204a69652057752c205869616e6779616e672057616e672c204368656e687569204c752c204368656e6c69616e672057616e672c205765692050616e2c205175616e205975616e2c20577571696f6e67204c69752c204368616f204368656e2c204a756e6875612046616e2c20536875616e67205a68616f2c2059756e204c762c204a696e676a69652048652c204a696e68752044696e672c2059616e672048616e2c2053756d656920486f6e672c204c69616e67205a68616e672c204a75616e205a68656e672c204a69616e6a69616f204875616e672c204c756e205169616e2c205869616f7969204368656e2c205975666569205a68616e672c20516920416e2c205a6869636f6e67205975616e2c2059696e677975652053752c2048616e205a68616e672c204a69616e677765692057616e672c2046656970656e67204875616e672c205975746f6e6720446f6e672c2054656e67204d612c205169616e6c6569205368692c2059756e7869616f204c692c2052756971696e67205975652c205068696c6c6970204368756e2c205469616e7869616e672059752c204e61205a68616e672c2053687561692057616e672c2048616966656e672059616e2c204368656e6768616f2059696e2c2048656e67205a686f752c20536875616e67205a68616e672c204c696e7a68656e205869652c204b657368756e2058752c204a756e79692046616e672c204c696e66656e67204c692c20596f6e676c696e204c6920616e6420427269616e204368656f6e672e205370656369616c207468616e6b7320746f2053696d6f6e204b696d2c205279616e204b696d2c2053686f756a69205a686f752c205975616e205a68616e672c205468616e68204e677579656e2c204a69616e204361692c20486f6e677765692043616f2c205374656e204c61757265797373656e732e"),
		GasLimit:   4700000,
		Difficulty: big.NewInt(1),
		Alloc:      testnetAlloc,
	}
}

// DefaultSCGenesisBlock returns the Ropsten network genesis block.
func DefaultSCGenesisBlock() *Genesis {

	return &Genesis{
		Config:     params.SideChainConfig,
		Timestamp:  1554004800,
		Nonce:      0,
		ExtraData:  hexutil.MustDecode("0x436f6769746f206572676f2073756d2e20457863657074206f7572206f776e2074686f75676874732c207468657265206973206e6f7468696e67206162736f6c7574656c7920696e206f757220706f7765722e20596f75206e656564206368616f7320696e20796f757220736f756c20746f206769766520626972746820746f20612064616e63696e6720737461722e20416c6561206961637461206573742e2020416c6c2063726564697420676f657320746f20746865207465616d3a2053686968616f2047756f2c2050656e67204c69752c205969204d6f2c204368617365204368616e672c2059697869616f2057616e672c20436520476f6e672c204865205a68616e672c2059756e6a69204d612c204a69652057752c205869616e6779616e672057616e672c204368656e687569204c752c204368656e6c69616e672057616e672c205765692050616e2c205175616e205975616e2c20577571696f6e67204c69752c204368616f204368656e2c204a756e6875612046616e2c20536875616e67205a68616f2c2059756e204c762c204a696e676a69652048652c204a696e68752044696e672c2059616e672048616e2c2053756d656920486f6e672c204c69616e67205a68616e672c204a75616e205a68656e672c204a69616e6a69616f204875616e672c204c756e205169616e2c205869616f7969204368656e2c205975666569205a68616e672c20516920416e2c205a6869636f6e67205975616e2c2059696e677975652053752c2048616e205a68616e672c204a69616e677765692057616e672c2046656970656e67204875616e672c205975746f6e6720446f6e672c2054656e67204d612c205169616e6c6569205368692c2059756e7869616f204c692c2052756971696e67205975652c205068696c6c6970204368756e2c205469616e7869616e672059752c204e61205a68616e672c2053687561692057616e672c2048616966656e672059616e2c204368656e6768616f2059696e2c2048656e67205a686f752c20536875616e67205a68616e672c204c696e7a68656e205869652c204b657368756e2058752c204a756e79692046616e672c204c696e66656e67204c692c20596f6e676c696e204c6920616e6420427269616e204368656f6e672e205370656369616c207468616e6b7320746f2053696d6f6e204b696d2c205279616e204b696d2c2053686f756a69205a686f752c205975616e205a68616e672c205468616e68204e677579656e2c204a69616e204361692c20486f6e677765692043616f2c205374656e204c61757265797373656e732e"),
		GasLimit:   4700000,
		Difficulty: big.NewInt(1),
		ParentHash: common.HexToHash("0x3210000000000000000000000000000000000000000000000000000000000000"),
		Alloc:      make(GenesisAlloc),
	}
}

// DefaultRinkebyGenesisBlock returns the Rinkeby network genesis block.
func DefaultRinkebyGenesisBlock() *Genesis {
	return &Genesis{
		Config:     params.RinkebyChainConfig,
		Timestamp:  1492009146,
		ExtraData:  hexutil.MustDecode("0x52657370656374206d7920617574686f7269746168207e452e436172746d616e42eb768f2244c8811c63729a21a3569731535f067ffc57839b00206d1ad20c69a1981b489f772031b279182d99e65703f0076e4812653aab85fca0f00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"),
		GasLimit:   4700000,
		Difficulty: big.NewInt(1),
		Alloc:      decodePrealloc(rinkebyAllocData),
	}
}

// DeveloperGenesisBlock returns the 'geth --dev' genesis block. Note, this must
// be seeded with the
func DeveloperGenesisBlock(period uint64, faucet common.Address) *Genesis {
	// Override the default period to the user requested one
	config := *params.AllCliqueProtocolChanges
	config.Clique.Period = period

	// Assemble and return the genesis with the precompiles and faucet pre-funded
	return &Genesis{
		Config:     &config,
		ExtraData:  append(append(make([]byte, 32), faucet[:]...), make([]byte, 65)...),
		GasLimit:   6283185,
		Difficulty: big.NewInt(1),
		Alloc: map[common.Address]GenesisAccount{
			common.BytesToAddress([]byte{1}): {Balance: big.NewInt(1)}, // ECRecover
			common.BytesToAddress([]byte{2}): {Balance: big.NewInt(1)}, // SHA256
			common.BytesToAddress([]byte{3}): {Balance: big.NewInt(1)}, // RIPEMD
			common.BytesToAddress([]byte{4}): {Balance: big.NewInt(1)}, // Identity
			common.BytesToAddress([]byte{5}): {Balance: big.NewInt(1)}, // ModExp
			common.BytesToAddress([]byte{6}): {Balance: big.NewInt(1)}, // ECAdd
			common.BytesToAddress([]byte{7}): {Balance: big.NewInt(1)}, // ECScalarMul
			common.BytesToAddress([]byte{8}): {Balance: big.NewInt(1)}, // ECPairing
			faucet:                           {Balance: new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(9))},
		},
	}
}

func decodePrealloc(data string) GenesisAlloc {
	var p []struct{ Addr, Balance *big.Int }
	if err := rlp.NewStream(strings.NewReader(data), 0).Decode(&p); err != nil {
		panic(err)
	}
	ga := make(GenesisAlloc, len(p))
	for _, account := range p {
		ga[common.BigToAddress(account.Addr)] = GenesisAccount{Balance: account.Balance}
	}
	return ga
}
