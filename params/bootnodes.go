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
	// Ethereum Foundation Go Bootnodes
	"enode://a979fb575495b8d6db44f750317d0f4622bf4c2aa3365d6af7c284339968eef29b69ad0dce72a4d8db5ebb4968de0e3bec910127f134779fbcb0cb6d3331163c@52.16.188.185:30303", // IE
	"enode://3f1d12044546b76342d59d4a05532c14b85aa669704bfe1f864fe079415aa2c02d743e03218e57a33fb94523adb54032871a6c51b2cc5514cb7c7e35b3ed0a99@13.93.211.84:30303",  // US-WEST
	"enode://78de8a0916848093c73790ead81d1928bec737d565119932b98c6b100d944b7a95e94f847f689fc723399d2e31129d182f7ef3863f2b4c820abbf3ab2722344d@191.235.84.50:30303", // BR
	"enode://158f8aab45f6d19c6cbf4a089c2670541a8da11978a2f90dbf6a502a4a3bab80d288afdbeb7ec0ef6d92de563767f3b1ea9e8e334ca711e9f8e2df5a0385e8e6@13.75.154.138:30303", // AU
	"enode://1118980bf48b0a3640bdba04e0fe78b1add18e1cd99bf22d53daac1fd9972ad650df52176e7c7d89d1114cfef2bc23a2959aa54998a46afcf7d91809f0855082@52.74.57.123:30303",  // SG

	// Ethereum Foundation C++ Bootnodes
	"enode://979b7fa28feeb35a4741660a16076f1943202cb72b6af70d327f053e248bab9ba81760f39d0701ef1d8f89cc1fbd2cacba0710a12cd5314d5e0c9021aa3637f9@5.1.83.226:30303", // DE
}

// TestnetBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// TTC test network.
var TestnetBootnodes = []string{

	"enode://b50b114ddebcccf0f452b433eed1ceaca49d1bc8075b89ccc627b110ad5e30d1a8d4f7eb5f936c17f0b8ee9a751879336b806f8c5d66e64b0191a24d879deaf8@47.105.86.215:30342",
	"enode://a990d36e7f9e647d0249b329dca164a20482310d6959b549d664eaf865ad249bcc2f45b07c6123aedbbe2316a02e9dbe7ff408c5677dbccf1329f276e7079fc9@47.105.86.215:30344",
	"enode://7798a1ff0e7352cbca47146f3b6b0d767ca7623d4902fda53b24d28fdd1422fa8a0c2f45d6c60d530b249c0189a055425d8258052de54ef5a9a04e281a9dd876@47.105.140.129:30330",
	"enode://cd426aab26e25b80d119861dbfdd800a44c066c87e7bdf3bae89b966d16b1cfce96ec559f870ee0113b8085636636d4b6c719ef975a81ebc98ee9b6d5bf541ae@47.105.140.129:30332",
	"enode://7b23c59907c0a3913d43f752d233019ee8113c7b50e1629cddfa001e6490d3a9fa695c76d503765c8104ccde398be7d9d87237e604eacaa83a214d2ba991341c@47.105.97.19:30312",
	"enode://5b6186b6b489af740db39035aa8bf29aec7b7aa8d4eee826f58e5f647a6188c12451b14dd4fc3055692b56d8daaec4f8bf2a46b879788caf90e1ac85b01cc6fb@47.105.97.19:30314",
	"enode://7ea312f4093e54a488ac53ea32c1a956885ffa4450fbc7e7d96768268ec5d35d0a17e2de8ce8a80d1ef41d79d423be6f95e7d5bb7351adcd33cd954a5cbf4c6d@47.105.142.208:30336",
	"enode://7323f49f6e4ef9c1f26430daebb82bce0c2c0bb0cb29d70b032ff4f6c473320cb4627cda4916cc950730d95580f571ca5c395d96a7d55b53e339a959e49237bc@47.105.142.208:30338",
	"enode://b61f14fe23d58692aad392511c8fabe89beeb564c925eea4480c540d23b45333437d90e3fb8ab554b237e2338adcd609da62130835b9028129d385c1ce426be4@47.105.142.208:30340",
	"enode://18f5e0481f49102ec63313b8bffb1d459b4dbbd863fed51b3f4dac9545e739ebab738e9304eaed03c06b12d3d3a88be6bcf0933978640cf7425630b1d9b2034e@47.105.131.192:30324",
	"enode://a9e82a3f8a45b537376c9444548e52bf60cd120bb5cb5df3f4497462dce61760a5b3a6d24f97d4bfaa57d161750b4b8bcf04e4a1b7fc2618df69e82a6b903a01@47.105.131.192:30326",
	"enode://382761b8c39e6565c445a0d696abbf5e9f1372f7db32d08b28c83c102112938c9980170c0e04f208fc25320aa6742c7530ee7ab99b1a535d68dd07c8415ac6d7@47.105.131.192:30328",
	"enode://cb60856cea1bd459f9fe6669eac7075e86e9f0ff5cf31c54dc26df84fa47ecdaa2c1aae0b0f274c45b8a585a8d532e2552e9ed3a82b1e8a067c929c26dfeaf66@47.105.78.210:30320",
	"enode://1462d1ee968baac8e8278624263e40e9864ca8b3193ca32b6c6df28f718c57b0d3f9b277ec628a341e848f486481aac5b1200d0718a11ce9c2e40688ef2017e8@47.105.78.210:30322",
	"enode://03116ab2a06a74449a4aa621b9a59aaf05ffef4a6541abe3617ba70622a2c063d424ac41a9af1c21ba396c81a191c1da341d3957b5a9802e2567b050bf016ad1@47.105.78.210:30318",
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
