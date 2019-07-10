// Copyright 2016 The go-ethereum Authors
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

package params

import (
	"fmt"
	"github.com/TTCECO/gttc/common"
	"github.com/TTCECO/gttc/rpc"
	"math/big"
)

var (
	MainnetGenesisHash = common.HexToHash("t0c6f20167950874a396ca06b717df8cd96597bc849b4935003cda98598c5e98be") // Mainnet genesis hash to enforce below configs on
	TestnetGenesisHash = common.HexToHash("t01fa3701d9164c47b587e335e9d4c2a34360dd77c84859c818ba7ea684bdf6ee1") // Testnet genesis hash to enforce below configs on
)

var (
	// MainnetChainConfig is the chain parameters to run a node on the main network.
	MainnetChainConfig = &ChainConfig{
		ChainId:             big.NewInt(8848),
		HomesteadBlock:      big.NewInt(1),
		EIP150Block:         big.NewInt(2),
		EIP150Hash:          common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
		EIP155Block:         big.NewInt(3),
		EIP158Block:         big.NewInt(3),
		ByzantiumBlock:      big.NewInt(4),
		ConstantinopleBlock: nil,
		Alien: &AlienConfig{
			Period:           3,
			Epoch:            201600,
			MaxSignerCount:   21,
			TrantorBlock:     new(big.Int).SetUint64(2968888),
			MinVoterBalance:  new(big.Int).Mul(big.NewInt(100), big.NewInt(1e+18)),
			GenesisTimestamp: 1554004800,
			SelfVoteSigners: []common.UnprefixedAddress{
				common.UnprefixedAddress(common.HexToAddress("t06e83430ca56ee33a26e5ce87239cb251981ccc2b")),
				common.UnprefixedAddress(common.HexToAddress("t01807efcb4dc252ff6958eaab770c8b3936a5378f")),
				common.UnprefixedAddress(common.HexToAddress("t0350fccf36124cecd26318e9931414ce872bdb68c")),
				common.UnprefixedAddress(common.HexToAddress("t009cbad80e089754f610cb8771d9eca05e4e22bdb")),
				common.UnprefixedAddress(common.HexToAddress("t09d507c10960531c9adc0ffdc9d9c735167275caf")),
				common.UnprefixedAddress(common.HexToAddress("t0c252c0f4d460554c679532072c8dbecd8d9ee89b")),
				common.UnprefixedAddress(common.HexToAddress("t0d8f68e2af8a061f0ea5e57ab7aca1b7fa96dab8a")),
				common.UnprefixedAddress(common.HexToAddress("t00c58019b9c8e293e3be8d3fd50f77af5f2e84bb7")),
				common.UnprefixedAddress(common.HexToAddress("t065dd958f7433cbe8353401d131c925e0424330b6")),
				common.UnprefixedAddress(common.HexToAddress("t0076c15f06f36b15544f1e97b4aacbd358d60cdf0")),
				common.UnprefixedAddress(common.HexToAddress("t0ecfd032885b4b9e69ab732e800c72296733165d7")),
				common.UnprefixedAddress(common.HexToAddress("t01a7910fe43b49b8bc33c04cb138cb2a8e1842f32")),
				common.UnprefixedAddress(common.HexToAddress("t0f52fe2e8decbbb3b00ebec7a1a50a41055d784ea")),
				common.UnprefixedAddress(common.HexToAddress("t035ef874a0f12581fd01fd2b178da7472475e253c")),
				common.UnprefixedAddress(common.HexToAddress("t090d4a9e77bf64b58f7c07d3bc19f8bb5e9d49031")),
				common.UnprefixedAddress(common.HexToAddress("t07bd38c427c685fbecbbe0daf49cda466b6475cc6")),
				common.UnprefixedAddress(common.HexToAddress("t0db1f586092917033e15298663594abb01eb98e39")),
				common.UnprefixedAddress(common.HexToAddress("t049574ad7832ff9a9214eb462cce2accf35f9118c")),
				common.UnprefixedAddress(common.HexToAddress("t0c8a7ca612be71d84c82c2c1fefbd035517df6745")),
				common.UnprefixedAddress(common.HexToAddress("t07e13706bab4bfae1f856d75e96676ab27eeea083")),
				common.UnprefixedAddress(common.HexToAddress("t0c5981e7fb6726be96345a732de6206bb1d66b963")),
				common.UnprefixedAddress(common.HexToAddress("t0ba99e0bb3fb9537db76a8ac1e76ebca5177954c9")),
				common.UnprefixedAddress(common.HexToAddress("t0d039d1feb6b13c3abe5089da9157fd41104c1aee")),
				common.UnprefixedAddress(common.HexToAddress("t0532c8772925e4b55a6bc99e954aa4cacc7d152b3")),
				common.UnprefixedAddress(common.HexToAddress("t0b464963fcb52b4666577987538a45e68876dc4e7")),
				common.UnprefixedAddress(common.HexToAddress("t08967f6d04ce36683ebe08c55caa15a177447f983")),
				common.UnprefixedAddress(common.HexToAddress("t005f39bfe9588f9297b8f3b019a3ee336efe47c47")),
				common.UnprefixedAddress(common.HexToAddress("t00c59dd1a15c3d5db4b4297cd79bfe72b60affc3e")),
				common.UnprefixedAddress(common.HexToAddress("t08f05387c4d637288dd197e26d5bdd3cb7087793c")),
				common.UnprefixedAddress(common.HexToAddress("t002289f35b60c97e27141c6aeb2691d25b531c755")),
				common.UnprefixedAddress(common.HexToAddress("t039e18521278e5121fdb0b691e84869bd4c645241")),
				common.UnprefixedAddress(common.HexToAddress("t04b4a0c8cb17b50d8d22610b307c349b63560ca4b")),
				common.UnprefixedAddress(common.HexToAddress("t09b25f97fa4e3892d9a86ac035a338b36dace5c4b")),
				common.UnprefixedAddress(common.HexToAddress("t0c520c15d943603dc333ebf6b5e39eb4d509fc1f8")),
				common.UnprefixedAddress(common.HexToAddress("t0da863ba260a36a11e3ea953b61de4a0eeffaa6f5")),
				common.UnprefixedAddress(common.HexToAddress("t03692048ef49479294bcfe9ee7e97508633756f3f")),
				common.UnprefixedAddress(common.HexToAddress("t027f7fcf7938618dfb0fc3668cd6fe7c1f7315870")),
				common.UnprefixedAddress(common.HexToAddress("t0f34961e5654a76335e0480bd7c7d370ad41ac74f")),
				common.UnprefixedAddress(common.HexToAddress("t092918ee96f529fdabab1a1ffda627c3d6b442ad9")),
				common.UnprefixedAddress(common.HexToAddress("t02024cc8d89f7cbd09a4085fbc729e3b9ee92c1be")),
				common.UnprefixedAddress(common.HexToAddress("t01b5887157beff2e2eff9ea9b8409f3ca1b6a052f")),
				common.UnprefixedAddress(common.HexToAddress("t0ad11612be2d9811ffe80f9e9ec1bbdc0ff34067c")),
				common.UnprefixedAddress(common.HexToAddress("t04e3011ab5b261cff133f4e8fb597dd0980814a94")),
				common.UnprefixedAddress(common.HexToAddress("t02834dc6b4b054fcf9cb206df4cce17fa0044826b")),
				common.UnprefixedAddress(common.HexToAddress("t07a2da45fd12d9bd44227ec58a5f0c3085ef18bf1")),
				common.UnprefixedAddress(common.HexToAddress("t0e8ae4d470fb87381f34a77c992a1de53fc2d2a3c")),
				common.UnprefixedAddress(common.HexToAddress("t0777689118d95751e1d709d7134adddd387226ac3")),
				common.UnprefixedAddress(common.HexToAddress("t00d6556b96b2b7cd095bf42aa2c287df99f22fc87")),
				common.UnprefixedAddress(common.HexToAddress("t0b97b279af3aa97655e6592b320e94505b41631ec")),
				common.UnprefixedAddress(common.HexToAddress("t0bce13d77339971d1f5f00c38f523ba7ee44c95ed")),
			},
		},
	}

	// TestnetChainConfig contains the chain parameters to run a node on the Ropsten test network.
	TestnetChainConfig = &ChainConfig{
		ChainId:             big.NewInt(8341),
		HomesteadBlock:      big.NewInt(1),
		EIP150Block:         big.NewInt(2),
		EIP150Hash:          common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
		EIP155Block:         big.NewInt(3),
		EIP158Block:         big.NewInt(3),
		ByzantiumBlock:      big.NewInt(4),
		ConstantinopleBlock: nil,
		Alien: &AlienConfig{
			Period:           3,
			Epoch:            201600,
			MaxSignerCount:   21,
			MinVoterBalance:  new(big.Int).Mul(big.NewInt(100), big.NewInt(1e+18)),
			TrantorBlock:     big.NewInt(695000),
			GenesisTimestamp: 1554004800,
			SelfVoteSigners: []common.UnprefixedAddress{
				common.UnprefixedAddress(common.HexToAddress("t0be6865ffcbbe5f9746bef5c84b912f2ad9e52075")),
				common.UnprefixedAddress(common.HexToAddress("t04909b4e54395de9e313ad8a2254fe2dcda99e91c")),
				common.UnprefixedAddress(common.HexToAddress("t0a034350c8e80eb4d15ac62310657b29c711bb3d5")),
			},
		},
	}

	// SideChainConfig contains the chain parameters to run a node on the Ropsten test network.
	SideChainConfig = &ChainConfig{
		ChainId:             big.NewInt(8123),
		HomesteadBlock:      big.NewInt(1),
		EIP150Block:         big.NewInt(2),
		EIP150Hash:          common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
		EIP155Block:         big.NewInt(3),
		EIP158Block:         big.NewInt(3),
		ByzantiumBlock:      big.NewInt(4),
		ConstantinopleBlock: nil,
		Alien: &AlienConfig{
			Period:           1,
			Epoch:            201600,
			MaxSignerCount:   21,
			TrantorBlock:     big.NewInt(5),
			MinVoterBalance:  new(big.Int).Mul(big.NewInt(100), big.NewInt(1e+18)),
			GenesisTimestamp: 1554004800,
			SelfVoteSigners:  []common.UnprefixedAddress{},
		},
	}

	// RinkebyChainConfig contains the chain parameters to run a node on the Rinkeby test network.
	RinkebyChainConfig = &ChainConfig{
		ChainId:             big.NewInt(4),
		HomesteadBlock:      big.NewInt(1),
		EIP150Block:         big.NewInt(2),
		EIP150Hash:          common.HexToHash("0x9b095b36c15eaf13044373aef8ee0bd3a382a5abb92e402afa44b8249c3a90e9"),
		EIP155Block:         big.NewInt(3),
		EIP158Block:         big.NewInt(3),
		ByzantiumBlock:      big.NewInt(1035301),
		ConstantinopleBlock: nil,
		Clique: &CliqueConfig{
			Period: 15,
			Epoch:  30000,
		},
	}

	// AllEthashProtocolChanges contains every protocol change (EIPs) introduced
	// and accepted by the Ethereum core developers into the Ethash consensus.
	//
	// This configuration is intentionally not using keyed fields to force anyone
	// adding flags to the config to also have to set these fields.
	AllEthashProtocolChanges = &ChainConfig{big.NewInt(1337), big.NewInt(0), big.NewInt(0), common.Hash{}, big.NewInt(0), big.NewInt(0), big.NewInt(0), nil, new(EthashConfig), nil, nil}

	// AllCliqueProtocolChanges contains every protocol change (EIPs) introduced
	// and accepted by the Ethereum core developers into the Clique consensus.
	//
	// This configuration is intentionally not using keyed fields to force anyone
	// adding flags to the config to also have to set these fields.
	AllCliqueProtocolChanges = &ChainConfig{big.NewInt(1337), big.NewInt(0), big.NewInt(0), common.Hash{}, big.NewInt(0), big.NewInt(0), big.NewInt(0), nil, nil, &CliqueConfig{Period: 0, Epoch: 30000}, nil}

	// AllAlienProtocolChanges contains every protocol change (EIPs) introduced
	// and accepted by the Ethereum core developers into the Alien consensus.
	//
	// This configuration is intentionally not using keyed fields to force anyone
	// adding flags to the config to also have to set these fields.
	AllAlienProtocolChanges = &ChainConfig{big.NewInt(1337), big.NewInt(0), big.NewInt(0), common.Hash{}, big.NewInt(0), big.NewInt(0), big.NewInt(0), nil, nil, nil, &AlienConfig{Period: 3, Epoch: 30000, MaxSignerCount: 21, MinVoterBalance: new(big.Int).Mul(big.NewInt(10000), big.NewInt(1000000000000000000)), GenesisTimestamp: 0, SelfVoteSigners: []common.UnprefixedAddress{}}}

	TestChainConfig = &ChainConfig{big.NewInt(1), big.NewInt(0), big.NewInt(0), common.Hash{}, big.NewInt(0), big.NewInt(0), big.NewInt(0), nil, new(EthashConfig), nil, nil}
	TestRules       = TestChainConfig.Rules(new(big.Int))
)

// ChainConfig is the core config which determines the blockchain settings.
//
// ChainConfig is stored in the database on a per block basis. This means
// that any network, identified by its genesis block, can have its own
// set of configuration options.
type ChainConfig struct {
	ChainId *big.Int `json:"chainId"` // Chain id identifies the current chain and is used for replay protection

	HomesteadBlock *big.Int `json:"homesteadBlock,omitempty"` // Homestead switch block (nil = no fork, 0 = already homestead)

	// EIP150 implements the Gas price changes (https://github.com/ethereum/EIPs/issues/150)
	EIP150Block *big.Int    `json:"eip150Block,omitempty"` // EIP150 HF block (nil = no fork)
	EIP150Hash  common.Hash `json:"eip150Hash,omitempty"`  // EIP150 HF hash (needed for header only clients as only gas pricing changed)

	EIP155Block *big.Int `json:"eip155Block,omitempty"` // EIP155 HF block
	EIP158Block *big.Int `json:"eip158Block,omitempty"` // EIP158 HF block

	ByzantiumBlock      *big.Int `json:"byzantiumBlock,omitempty"`      // Byzantium switch block (nil = no fork, 0 = already on byzantium)
	ConstantinopleBlock *big.Int `json:"constantinopleBlock,omitempty"` // Constantinople switch block (nil = no fork, 0 = already activated)

	// Various consensus engines
	Ethash *EthashConfig `json:"ethash,omitempty"`
	Clique *CliqueConfig `json:"clique,omitempty"`
	Alien  *AlienConfig  `json:"alien,omitempty"`
}

// EthashConfig is the consensus engine configs for proof-of-work based sealing.
type EthashConfig struct{}

// String implements the stringer interface, returning the consensus engine details.
func (c *EthashConfig) String() string {
	return "ethash"
}

// CliqueConfig is the consensus engine configs for proof-of-authority based sealing.
type CliqueConfig struct {
	Period uint64 `json:"period"` // Number of seconds between blocks to enforce
	Epoch  uint64 `json:"epoch"`  // Epoch length to reset votes and checkpoint
}

// String implements the stringer interface, returning the consensus engine details.
func (c *CliqueConfig) String() string {
	return "clique"
}

type GenesisAccount struct {
	Balance string `json:"balance"`
}

// AlienLightConfig is the config for light node of alien
type AlienLightConfig struct {
	Alloc map[common.UnprefixedAddress]GenesisAccount `json:"alloc"`
}

// AlienConfig is the consensus engine configs for delegated-proof-of-stake based sealing.
type AlienConfig struct {
	Period           uint64                     `json:"period"`           // Number of seconds between blocks to enforce
	Epoch            uint64                     `json:"epoch"`            // Epoch length to reset votes and checkpoint
	MaxSignerCount   uint64                     `json:"maxSignersCount"`  // Max count of signers
	MinVoterBalance  *big.Int                   `json:"minVoterBalance"`  // Min voter balance to valid this vote
	GenesisTimestamp uint64                     `json:"genesisTimestamp"` // The LoopStartTime of first Block
	SelfVoteSigners  []common.UnprefixedAddress `json:"signers"`          // Signers vote by themselves to seal the block, make sure the signer accounts are pre-funded
	SideChain        bool                       `json:"sideChain"`        // If side chain or not
	MCRPCClient      *rpc.Client                // Main chain rpc client for side chain
	PBFTEnable       bool                       `json:"pbft"` //

	TrantorBlock  *big.Int          `json:"trantorBlock,omitempty"`  // Trantor switch block (nil = no fork)
	TerminusBlock *big.Int          `json:"terminusBlock,omitempty"` // Terminus switch block (nil = no fork)
	LightConfig   *AlienLightConfig `json:"lightConfig,omitempty"`
}

// String implements the stringer interface, returning the consensus engine details.
func (a *AlienConfig) String() string {
	return "alien"
}

// IsTrantor returns whether num is either equal to the Trantor block or greater.
func (a *AlienConfig) IsTrantor(num *big.Int) bool {
	return isForked(a.TrantorBlock, num)
}

// IsTerminus returns whether num is either equal to the Terminus block or greater.
func (a *AlienConfig) IsTerminus(num *big.Int) bool {
	return isForked(a.TerminusBlock, num)
}

// String implements the fmt.Stringer interface.
func (c *ChainConfig) String() string {
	var engine interface{}
	switch {
	case c.Ethash != nil:
		engine = c.Ethash
	case c.Clique != nil:
		engine = c.Clique
	case c.Alien != nil:
		engine = c.Alien
	default:
		engine = "unknown"
	}
	return fmt.Sprintf("{ChainID: %v Homestead: %v EIP150: %v EIP155: %v EIP158: %v Byzantium: %v Constantinople: %v Engine: %v}",
		c.ChainId,
		c.HomesteadBlock,
		c.EIP150Block,
		c.EIP155Block,
		c.EIP158Block,
		c.ByzantiumBlock,
		c.ConstantinopleBlock,
		engine,
	)
}

// IsHomestead returns whether num is either equal to the homestead block or greater.
func (c *ChainConfig) IsHomestead(num *big.Int) bool {
	return isForked(c.HomesteadBlock, num)
}

func (c *ChainConfig) IsEIP150(num *big.Int) bool {
	return isForked(c.EIP150Block, num)
}

func (c *ChainConfig) IsEIP155(num *big.Int) bool {
	return isForked(c.EIP155Block, num)
}

func (c *ChainConfig) IsEIP158(num *big.Int) bool {
	return isForked(c.EIP158Block, num)
}

func (c *ChainConfig) IsByzantium(num *big.Int) bool {
	return isForked(c.ByzantiumBlock, num)
}

func (c *ChainConfig) IsConstantinople(num *big.Int) bool {
	return isForked(c.ConstantinopleBlock, num)
}

// GasTable returns the gas table corresponding to the current phase (homestead or homestead reprice).
//
// The returned GasTable's fields shouldn't, under any circumstances, be changed.
func (c *ChainConfig) GasTable(num *big.Int) GasTable {
	if num == nil {
		return GasTableHomestead
	}
	switch {
	case c.IsEIP158(num):
		return GasTableEIP158
	case c.IsEIP150(num):
		return GasTableEIP150
	default:
		return GasTableHomestead
	}
}

// CheckCompatible checks whether scheduled fork transitions have been imported
// with a mismatching chain configuration.
func (c *ChainConfig) CheckCompatible(newcfg *ChainConfig, height uint64) *ConfigCompatError {
	bhead := new(big.Int).SetUint64(height)

	// Iterate checkCompatible to find the lowest conflict.
	var lasterr *ConfigCompatError
	for {
		err := c.checkCompatible(newcfg, bhead)
		if err == nil || (lasterr != nil && err.RewindTo == lasterr.RewindTo) {
			break
		}
		lasterr = err
		bhead.SetUint64(err.RewindTo)
	}
	return lasterr
}

func (c *ChainConfig) checkCompatible(newcfg *ChainConfig, head *big.Int) *ConfigCompatError {
	if isForkIncompatible(c.HomesteadBlock, newcfg.HomesteadBlock, head) {
		return newCompatError("Homestead fork block", c.HomesteadBlock, newcfg.HomesteadBlock)
	}
	if isForkIncompatible(c.EIP150Block, newcfg.EIP150Block, head) {
		return newCompatError("EIP150 fork block", c.EIP150Block, newcfg.EIP150Block)
	}
	if isForkIncompatible(c.EIP155Block, newcfg.EIP155Block, head) {
		return newCompatError("EIP155 fork block", c.EIP155Block, newcfg.EIP155Block)
	}
	if isForkIncompatible(c.EIP158Block, newcfg.EIP158Block, head) {
		return newCompatError("EIP158 fork block", c.EIP158Block, newcfg.EIP158Block)
	}
	if c.IsEIP158(head) && !configNumEqual(c.ChainId, newcfg.ChainId) {
		return newCompatError("EIP158 chain ID", c.EIP158Block, newcfg.EIP158Block)
	}
	if isForkIncompatible(c.ByzantiumBlock, newcfg.ByzantiumBlock, head) {
		return newCompatError("Byzantium fork block", c.ByzantiumBlock, newcfg.ByzantiumBlock)
	}
	if isForkIncompatible(c.ConstantinopleBlock, newcfg.ConstantinopleBlock, head) {
		return newCompatError("Constantinople fork block", c.ConstantinopleBlock, newcfg.ConstantinopleBlock)
	}
	return nil
}

// isForkIncompatible returns true if a fork scheduled at s1 cannot be rescheduled to
// block s2 because head is already past the fork.
func isForkIncompatible(s1, s2, head *big.Int) bool {
	return (isForked(s1, head) || isForked(s2, head)) && !configNumEqual(s1, s2)
}

// isForked returns whether a fork scheduled at block s is active at the given head block.
func isForked(s, head *big.Int) bool {
	if s == nil || head == nil {
		return false
	}
	return s.Cmp(head) <= 0
}

func configNumEqual(x, y *big.Int) bool {
	if x == nil {
		return y == nil
	}
	if y == nil {
		return x == nil
	}
	return x.Cmp(y) == 0
}

// ConfigCompatError is raised if the locally-stored blockchain is initialised with a
// ChainConfig that would alter the past.
type ConfigCompatError struct {
	What string
	// block numbers of the stored and new configurations
	StoredConfig, NewConfig *big.Int
	// the block number to which the local chain must be rewound to correct the error
	RewindTo uint64
}

func newCompatError(what string, storedblock, newblock *big.Int) *ConfigCompatError {
	var rew *big.Int
	switch {
	case storedblock == nil:
		rew = newblock
	case newblock == nil || storedblock.Cmp(newblock) < 0:
		rew = storedblock
	default:
		rew = newblock
	}
	err := &ConfigCompatError{what, storedblock, newblock, 0}
	if rew != nil && rew.Sign() > 0 {
		err.RewindTo = rew.Uint64() - 1
	}
	return err
}

func (err *ConfigCompatError) Error() string {
	return fmt.Sprintf("mismatching %s in database (have %d, want %d, rewindto %d)", err.What, err.StoredConfig, err.NewConfig, err.RewindTo)
}

// Rules wraps ChainConfig and is merely syntatic sugar or can be used for functions
// that do not have or require information about the block.
//
// Rules is a one time interface meaning that it shouldn't be used in between transition
// phases.
type Rules struct {
	ChainId                                   *big.Int
	IsHomestead, IsEIP150, IsEIP155, IsEIP158 bool
	IsByzantium                               bool
}

func (c *ChainConfig) Rules(num *big.Int) Rules {
	chainId := c.ChainId
	if chainId == nil {
		chainId = new(big.Int)
	}
	return Rules{ChainId: new(big.Int).Set(chainId), IsHomestead: c.IsHomestead(num), IsEIP150: c.IsEIP150(num), IsEIP155: c.IsEIP155(num), IsEIP158: c.IsEIP158(num), IsByzantium: c.IsByzantium(num)}
}
