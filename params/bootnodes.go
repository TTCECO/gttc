// Copyright 2015 The go-ethereum Authors
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

// MainnetBootnodes are the enode URLs of the P2P bootstrap nodes running on
// the main Ethereum network.
var MainnetBootnodes = []string{

	// tokyo
	"enode://94499e31da30473576643d1c27ec163f158fbef47aa1f80a05f400ab2f2ac22e6f8fac224b8a903a095ba7f3ac7c528f80d06157ef9f2f2e0aa6d0f504d9f9ca@35.189.152.23:30310",
	"enode://da661c34ded2ba0547d5d1955fc7f4e53500db46fecf4d880284e5b3a540563c17be59f90350d0639472bb310e8117fe1baea1e59c75460c3af673b2623609fa@35.189.152.23:30311",
	"enode://e6a43da20ac2b52a4396ed9f368c5f36a0d9211bc1651a7c9d548654f26699d48882405164c8f8141d3b22134c930feee225586de8a4777009417a7901c9451c@35.189.152.23:30312",
	// us-east
	"enode://a3c9998080edb5faebb488397485686bfff13b45c77b05ae9e93f414570a0f8a7931ceab4ecec3bebbf9a4167a56aa34a62cfb2c70a792f70b871c6376e7d499@35.243.190.93:30310",
	"enode://0e552edb895bfdf3a66849e85e761f83697b7e51f8dce4092a858fd2e5276aa4ff669ccc415d97941e33826113404afe41a2fd4e6aff0c08f1357e37072723ba@35.243.190.93:30311",
	"enode://3c2cbfd07064983327d7153c75918bcedb32e70a3856e93445ee47334cb9ea9dd0d9c2debddb47737cca5484be2fac352ba86a601ca7ca17eb427db79d7d7c25@35.243.190.93:30312",
}

// TestnetBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// TTC test network.
var TestnetBootnodes = []string{

	"enode://b9487401a29c9c2b6f58ad96d080c22ed5c93a037d5cbd3412dc310837081ce6aa2b43910c3f1e8c94e072d56ddd2741941a04ae889f9119ab3bcfde8172fff8@47.111.177.215:30310",
	"enode://b52b9dec747cada8e4697544d857ab4b7e4298dce707f0591166cb3fedc07de86daffe73c49a23fd8e2b911b2246ea81a6ef709f42eab01c6d61b49c6f2b89c8@47.111.177.215:30311",
	"enode://f70bc22fc68fa25de65d365926deecb961b33165cba4078236d83e05259d6082979c2a0c5eb436351f49ddec2150e20b1425ca2c288512db5eeb2702b43f0d94@47.111.178.14:30312",
	"enode://bfbdf7ecb9be785ea0105a28db115dbdb61f6e486d6310859985974f05555d10e531290bd821a962efafdec087c075f6e5d15efd94937fb00d10c053b2bd56ed@47.111.177.154:30313",
}

// SidechainBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// TTC sidechain network.
var SidechainBootnodes = []string{
	// todo : the enode below is the test address local
	"enode://1a4cc9c8512256ff475990c4ab03636ac7771315a42432a240c6161befa162324be5af0f65828f0b7ec468b3d2183c430fb4a33d88ba5057eaa35b36a3af4b56@127.0.0.1:30510",
}


// RinkebyBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Rinkeby test network.
var RinkebyBootnodes = []string{
	"enode://a24ac7c5484ef4ed0c5eb2d36620ba4e4aa13b8c84684e1b4aab0cebea2ae45cb4d375b77eab56516d34bfbd3c1a833fc51296ff084b770b94fb9028c4d25ccf@52.169.42.101:30303", // IE
	"enode://343149e4feefa15d882d9fe4ac7d88f885bd05ebb735e547f12e12080a9fa07c8014ca6fd7f373123488102fe5e34111f8509cf0b7de3f5b44339c9f25e87cb8@52.3.158.184:30303",  // INFURA
	"enode://b6b28890b006743680c52e64e0d16db57f28124885595fa03a562be1d2bf0f3a1da297d56b13da25fb992888fd556d4c1a27b1f39d531bde7de1921c90061cc6@159.89.28.211:30303", // AKASHA
}

// DiscoveryV5Bootnodes are the enode URLs of the P2P bootstrap nodes for the
// experimental RLPx v5 topic-discovery network.
var DiscoveryV5Bootnodes = []string{
	"enode://06051a5573c81934c9554ef2898eb13b33a34b94cf36b202b69fde139ca17a85051979867720d4bdae4323d4943ddf9aeeb6643633aa656e0be843659795007a@35.177.226.168:30303",
	"enode://0cc5f5ffb5d9098c8b8c62325f3797f56509bff942704687b6530992ac706e2cb946b90a34f1f19548cd3c7baccbcaea354531e5983c7d1bc0dee16ce4b6440b@40.118.3.223:30304",
	"enode://1c7a64d76c0334b0418c004af2f67c50e36a3be60b5e4790bdac0439d21603469a85fad36f2473c9a80eb043ae60936df905fa28f1ff614c3e5dc34f15dcd2dc@40.118.3.223:30306",
	"enode://85c85d7143ae8bb96924f2b54f1b3e70d8c4d367af305325d30a61385a432f247d2c75c45c6b4a60335060d072d7f5b35dd1d4c45f76941f62a4f83b6e75daaf@40.118.3.223:30307",
}
