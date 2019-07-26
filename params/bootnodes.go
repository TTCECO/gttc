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

	"enode://61f22feba7bd39147630431a0001e4348d86c6cf78814901695d00f37601e66ab4e0c01d3eee21dc5b3619639d84a117d59bf6231bd8632cda7d3ee0e3452beb@35.190.239.155:30311",
	"enode://3072c9ac8a0890119958ba6a1cc1825f65470e1f1dea17252bdb412e93e48cdb8f849d01d80af0df2fb8909c000980758211f9e49fddb2979dff9f74313f3336@34.77.28.121:30311",
	"enode://ca535274576024a01a27d95c5adf708c93c741cefbde5acbceb382186e4dcb840129a66d358df95806d412ffb24aae55b223ff5ee5e42b034530d4f20132bc67@35.243.161.121:30311",
}

// TestnetBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// TTC test network.
var TestnetBootnodes = []string{

	"enode://b9487401a29c9c2b6f58ad96d080c22ed5c93a037d5cbd3412dc310837081ce6aa2b43910c3f1e8c94e072d56ddd2741941a04ae889f9119ab3bcfde8172fff8@47.111.177.215:30310",
	"enode://b52b9dec747cada8e4697544d857ab4b7e4298dce707f0591166cb3fedc07de86daffe73c49a23fd8e2b911b2246ea81a6ef709f42eab01c6d61b49c6f2b89c8@47.111.177.215:30311",
	"enode://f70bc22fc68fa25de65d365926deecb961b33165cba4078236d83e05259d6082979c2a0c5eb436351f49ddec2150e20b1425ca2c288512db5eeb2702b43f0d94@47.111.178.14:30312",
	"enode://bfbdf7ecb9be785ea0105a28db115dbdb61f6e486d6310859985974f05555d10e531290bd821a962efafdec087c075f6e5d15efd94937fb00d10c053b2bd56ed@47.111.177.154:30313",
	"enode://8327d9c22125b548fe9bf6ea463d011a87774974f7178847931d9ff4ca1b01151a4ef2a6c3a2626e7c2132d5e39f4f59437ab47feee8f3b9c5fd32cef118eb6b@47.111.178.14:30314",
	"enode://d72b636e208ad90cf46da4b6c04b2bf2e9a5a7a240655ea0dd8162e9f59bf734b12a8e4770c9e1f9eef11f2308241c49ffce62fef86fb78fc48d92fcbd72cace@47.111.178.14:30315",
	"enode://c8712027f2351bc59d0f5e8d8d278c645d4e6f91875a706684a20677e2695f20e29903d7ee679831e1bad7550b65cccc34b8c23abf3d09749ee556dce7415047@47.111.178.14:30316",
	"enode://082b1d3068ca66dc35c2439b64ce5b43c67585f0bb140fe15c4acf9db6a2cf821c0ab399bfa0f850cc2f30b71179ede9687a81e9d406cac52134386d8838224d@47.111.177.154:30318",
	"enode://2ee5103c7ce3b3cfda3ac081d0541903ebd834cba75907e4612dc825e37e2d83deb3ce579bc4bc64322f68ebdc7fa0b8181d00852aa92d5df6735030aff9fd53@47.111.177.154:30319",
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
