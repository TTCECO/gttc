# Go TTC

Golang implementation of MARO Blockchain (Previously known as TTC protocol).

[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg)](https://godoc.org/github.com/TTCECO/gttc)
[![GoReport](https://goreportcard.com/badge/github.com/TTCECO/gttc)](https://goreportcard.com/report/github.com/TTCECO/gttc)
[![Travis](https://travis-ci.org/TTCECO/gttc.svg?branch=master)](https://travis-ci.org/TTCECO/gttc)
[![License](https://img.shields.io/badge/license-GPL%20v3-blue.svg)](LICENSE)
## About gttc

gttc is base on [go-ethereum](https://github.com/ethereum/go-ethereum), the main part be modified is in [consensus](consensus/) directory. We add a new consensus algorithm named [alien](consensus/alien/) in it.

Alien is a simple version of DPOS-PBFT consensus algorithm, which in [consensus/alien](consensus/alien/):

* **alien.go**          : Implement the consensus interface
* **custom_tx.go**      : Process the custom transaction such as vote,proposal,declare and so on...
* **snapshot.go**       : Keep the snapshot of vote and confirm status for each block
* **signer_queue.go**   : calculate the order of signer queue
* **api.go**            : API
* **cross_chain.go**    : Cross chain communication by custom transaction

Alien use header.extra to record the all infomation of current block and keep signature of miner. The snapshot keep vote & confirm information of whole chain, which will be update by each Seal or VerifySeal. By the end of each loop, the miner will calculate the next loop miners base on the snapshot. Code annotation will show the details about how it works.

## Mainnet Information
* **Current Mainnet and Testnet is deploy the code of branch release/v0.2.3**

More information about this upgrade will be found [UPGRADE TO GTTC V0.2.3](https://github.com/TTCECO/gttc/wiki/UPGRADE-TO-GTTC-V0.2.3)

## Minimum Requirements

Requirement|Notes
---|---
Go version | Go1.9 ~ Go1.15, Go1.13 is recommended.

## Install

See the [HOWTO_INSTALL](https://github.com/TTCECO/gttc/wiki/Building-GTTC)

[Enode list for Mainnet & Slavenet](https://github.com/TTCECO/gttc/wiki/Public-Enode-address)

## Other Documents List

You can find all documents in our [Wiki](https://github.com/TTCECO/gttc/wiki/)

* [DPOS_CONSENSUS_ALGORITHM](https://github.com/TTCECO/gttc/wiki/DPOS_CONSENSUS_ALGORITHM): `description of DPOS algorithm`
* [PBFT_CONSENSUS_ALGORITHM](https://github.com/TTCECO/gttc/wiki/PBFT_CONSENSUS_ALGORITHM): `description of PBFT algorithm`
* [HOWTO_IMPLEMENT_DPOS_PBFT_IN_ALIEN](https://github.com/TTCECO/gttc/wiki/HOWTO_IMPLEMENT_DPOS_PBFT_IN_ALIEN): `details about how we implement dpos and pbft`
* [genesis.json](https://github.com/TTCECO/gttc/wiki/genesis.json)  : `genesis.json file for the testnet we deploy`
* [HOWTO_RUNNING_TEST_ON_PRIVATE_NETWORK](https://github.com/TTCECO/gttc/wiki/HOWTO_RUNNING_TEST_ON_PRIVATE_NETWORK) : `The instruction of deploy your own testnet`
* [HOWTO_VOTE_ON_GTTC](https://github.com/TTCECO/gttc/wiki//HOWTO_VOTE_ON_GTTC)  : `how to vote on alien testnet and view snapshot through API`
* [GENESIS_JSON_SAMPLE](https://github.com/TTCECO/gttc/wiki/GENESIS_JSON_SAMPLE) : `genesis.json sample`

* [build slave network instruction](https://github.com/TTCECO/gttc/wiki/build-slave-network-instruction)
* [how to check running status of slave network](https://github.com/TTCECO/gttc/wiki/how-to-check-running-status-of-slave-network)

## Contact

email: peng@ttc.eco
