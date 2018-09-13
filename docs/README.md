
## Readme for Code Review

#### Prerequisites for code review
You should have good knowledge of golang, understand the concept of DPOS algorithm & PBFT algorithm, familiar with the code of go-ethereum, especially the consensus part.

If you not familiar with DPOS or PBFT algorithm, you can find sample description for these algorithm in [DPOS](DPOS_CONSENSUS_ALGORITHM.md), [PBFT](PBFT_CONSENSUS_ALGORITHM.md)

#### About gttc

gttc is base on [go-ethereum (v1.8.9)](https://github.com/ethereum/go-ethereum), the main part be modified is in [consensus](../consensus/) directory. We add a new consensus algorithm named [alien](../consensus/alien/) in it.

Alien is a simple version of DPOS-PBFT consensus algorithm, which contain 4 files in [consensus/alien](../consensus/alien/):

* **alien.go**    : Implement the consensus interface
* **custom_tx.go** : Process the custom transaction such as vote,proposal,declare and so on...
* **snapshot.go** : Keep the snapshot of vote and confirm status for each block
* **snapshot_test.go** : test for snapshot
* **signer_queue.go**  : calculate the order of signer queue
* **signer_queue_test.go** : test for signer_queue
* **api.go**      : API

If you familiar with clique, you will find alien like that very much. We also use header.extra to record the all infomation of current block and keep signature of miner. The snapshot keep vote & confirm information of whole chain, which will be update by each Seal or VerifySeal. By the end of each loop, the miner will calculate the next loop miners from the snapshot. Code annotation will show the details about how it works.

All the code & documents in this directory is in http://github.com/TTCECO/gttc and will be update there, but it's a private repository, so if you need the access right, please contact us.

Current test chain is deploy the code of branch v0.0.4

#### Documents List

* [DPOS_CONSENSUS_ALGORITHM.md](DPOS_CONSENSUS_ALGORITHM.md): `description of DPOS algorithm`
* [PBFT_CONSENSUS_ALGORITHM.md](PBFT_CONSENSUS_ALGORITHM.md): `description of PBFT algorithm`
* [HOWTO_IMPLEMENT_DPOS_PBFT_IN_ALIEN.md](HOWTO_IMPLEMENT_DPOS_PBFT_IN_ALIEN.md): `details about how we implement dpos and pbft`
* [genesis.json](genesis.json)  : `genesis.json file for the testnet we deploy`
* [HOWTO_RUNNING_TEST_ON_PRIVATE_NETWORK.md](HOWTO_RUNNING_TEST_ON_PRIVATE_NETWORK.md) : `The instruction of deploy your own testnet`
* [HOWTO_VOTE_ON_GTTC.md](HOWTO_VOTE_ON_GTTC.md)  : `how to vote on alien testnet and view snapshot through API`
* [GENESIS_JSON_SAMPLE.md](GENESIS_JSON_SAMPLE.md) : `genesis.json sample`
* [HOWTO_BUILDING_GTTC.md](HOWTO_BUILDING_GTTC.md) : `a link to how to build geth, it's same as process of build our code`

#### Connection to Testnet

* [gttc](../cmd/gttc)

```

gttc --datedir node1/ init genesis.json

gttc --datadir node1/ --syncmode 'full' --rpcapi 'enode://f98bd1311c937f4314adacf2a258f88a470c0b0b199c1b2098d5e4c4ec91797a95525d1f62bdb09c251aa3b0aa0f92b212111cbc62b6dce732f10eca10d22f0e@47.105.142.208:30339,enode://bef0466f865d1abbe8e9090805fa30250c013b9d41ad15353bc6e5d58591fb15af1ac6709ac08f8c7f422939617454b207ab4696ac28c8cd4c33eb5d52136912@47.105.140.129:30331,enode://25a64b450d0d23d36f8326e2ee79b9a4f072bc6518981f19c761164840dcec8497b7bfa2ed5dfbe189ad05a6bfc1e69cdb05528d15b9a06f1d923ce8d16ad560@47.105.131.192:30325,enode://30d66152c08d7fdb50269ae2af0f2d39b98a081564662b9ca9267e8cba6c43a61262bdd6435657791ff897a23b4e9f1a43ca3f31c2d517f09f735955b0061666@47.105.78.210:30319' --networkid 8434

```

You can test, attack or do anything to this test chain, it is just for test.

#### Contact

email: liupeng@tataufo.com