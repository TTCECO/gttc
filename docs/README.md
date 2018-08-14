
## Readme for Code Review

#### About gttc

gttc is base on [go-ethereum (v1.8.9)](https://github.com/ethereum/go-ethereum), the main part be modified is in [consensus](../consensus/) directory. We add a new consensus algorithm named [alien](../consensus/alien/) in it.

Alien is a simple version of DPOS-PBFT consensus algorithm, which contain 4 files in [consensus/alien](../consensus/alien/):

* **alien.go**    : Implement the consensus interface
* **snapshot.go** : Keep the snapshot of vote and confirm status for each block
* **snapshot_test.go** : test for snapshot
* **api.go**      : API

If you familiar with clique, you will find alien like that very much. We also use header.extra to record the all infomation of current block and keep signature of miner. The snapshot keep vote & confirm information of whole blockchain, which will be update by each Seal or VerifySeal. By the end of each loop, the miner will calculate the next loop miners from the snapshot. Code annotation will show the details about how it works.

All the code & documents in this directory is in http://github.com/TTCECO/gttc and will be update there, but it's a private repository, so if you need the access right, please contact me(liupeng@tataufo.com).

gttc-release-v0.0.3.zip in this folder is the latest version of release.

#### Documents List

* gttc-release-v0.0.3.zip
* [DPOS_CONSENSUS_ALGORITHM.md](DPOS_CONSENSUS_ALGORITHM.md): description of DPOS algorithm
* [genesis.json](genesis.json)  : genesis.json file for the testnet we deploy
* [HOWTO_RUNNING_TEST_ON_PRIVATE_NETWORK.md](HOWTO_RUNNING_TEST_ON_PRIVATE_NETWORK.md) : The instruction of deploy your own testnet.
* [HOWTO_VOTE_ON_GTTC.md](HOWTO_VOTE_ON_GTTC.md)  : how to vote & confirm in alien(dpos-pbft).
* [GENESIS_JSON_SAMPLE.md](GENESIS_JSON_SAMPLE.md) : genesis.json sample.
* [HOWTO_BUILDING_GTTC.md](HOWTO_BUILDING_GTTC.md) : a link to how to build geth, it's same as process of build our code.

#### Connection to Testnet

* [gttc](../cmd/gttc)

```

gttc --datedir node1/ init genesis.json

gttc --datadir node1/ --syncmode 'full' --rpcapi 'personal,db,eth,net,web3,txpool,miner,net' --bootnodes 'enode://65fb31ebbc15eb9370dad7696416786492b504492f8e0b854015a9f543ca8f630b9f2d74dfefce15b4027a6977765a9a4941c105cf5bb8f87c706726287ecb39@39.106.104.30:30312' --networkid 1084

```

You can test, attack or do anything to this testnet, it is just for test.

The test account we provide on testnet is "0xaafeaaf6111762fea733ff7b4c8b59ac69316385", the current balance of this account is: 7.13623846352979940529142984724747568191373312e+44 . The detail of this address is:

```
Address:        0xAafEAAf6111762fEA733fF7B4C8B59ac69316385
Public key:     046074b72df5d4eccb492e0cc12b8144a446529a040b9aec2f57cb73c631c1ffa70df74f6055f6efdc4c6b9e65a2361360491d55913d9e3ad364ba1839d0c100d9
Private key:    5e0ddf6900fee7e399bd4e258821ee4cacf1ed8ddc3cb11976311b895553c1a1
```

#### Contact

email: liupeng@tataufo.com