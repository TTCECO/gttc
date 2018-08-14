## Implement DPOS & PBFT Algorithm

#### Abstract

There are two consensus already implement by go-ethereum,  ethash (POW consensus) is used by ethereum, and clique (POA consensus) is used by testnet of ethereum. If you familiar with clique, you will find alien like that very much.

Alien also use Extra field in header of block to record the all infomation of current block and keep signature of miner. The snapshot keep votes & confirm information of whole blockchain, which will be update by each Seal or VerifySeal func.

We will describe how its work in the follow sections.

#### Directory Structure

Alien contain 4 files in [consensus/alien](../consensus/alien/):

* **alien.go**    : Implement the consensus interface such as Seal, VerifySeal, Finalize ...
* **snapshot.go** : Keep the snapshot of vote and confirm status for each block
* **snapshot_test.go** : test case for snapshot
* **api.go**      : APIs


#### Data Structure
```
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

```

#### Vote by Transaction


#### Calculate the Signers Order


#### Confirm block by Transaction











