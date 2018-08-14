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
Alien is the delegated-proof-of-stake consensus engine

```
type HeaderExtra struct {
	CurrentBlockConfirmations []Confirmation
	CurrentBlockVotes         []Vote
	ModifyPredecessorVotes    []Vote
	LoopStartTime             uint64
	SignerQueue               []common.Address
	SignerMissing             []common.Address
	ConfirmedBlockNumber      uint64
}
```
As describe in abstract, HeaderExtra is the struct of current block information, after rpl encode to []byte, the data save in header.Extra[32:len(header.extra)-65]. The header.Extra[:32] keep the geth and go version, and header.Extra[len(header.extra)-65:] keep the signature of miner.

```
type Vote struct {
	Voter     common.Address
	Candidate common.Address
	Stake     *big.Int
}

```
Vote come from custom tx which data like "ufo:1:event:vote",which the sender of tx is Voter,and  the tx.to is Candidate. Stake is the balance of Voter when that address create this vote


```
type Confirmation struct {
	Signer      common.Address
	BlockNumber *big.Int
}

```
Confirmation come  from custom tx which data like "ufo:1:event:confirm:123", 123 is the block number be confirmed, the sender of tx is valid Signer only if the signer in the SignerQueue for block number 123


```
type Snapshot struct {
	config   *params.AlienConfig // Consensus engine parameters to fine tune behavior
	sigcache *lru.ARCCache       // Cache of recent block signatures to speed up ecrecover

	Number          uint64                       `json:"number"`          // Block number where the snapshot was created
	ConfirmedNumber uint64                       `json:"confirmedNumber"` // Block number confirmed when the snapshot was created
	Hash            common.Hash                  `json:"hash"`            // Block hash where the snapshot was created
	HistoryHash     []common.Hash                `json:"historyHash"`     // Block hash list for two recent loop
	Signers         []*common.Address            `json:"signers"`         // Signers queue in current header
	Votes           map[common.Address]*Vote     `json:"votes"`           // All validate votes from genesis block
	Tally           map[common.Address]*big.Int  `json:"tally"`           // Stake for each candidate address
	Voters          map[common.Address]*big.Int  `json:"voters"`          // block number for each voter address
	Punished        map[common.Address]uint64    `json:"punished"`        // The signer be punished count cause of missing seal
	Confirmations   map[uint64][]*common.Address `json:"confirms"`        // The signer confirm given block number
	HeaderTime      uint64                       `json:"headerTime"`      // Time of the current header
	LoopStartTime   uint64                       `json:"loopStartTime"`   // Start Time of the current loop
}
```
Snapshot is the state of the authorization voting at a given point in time.


#### Vote by Transaction

Before the signer seal the block (in Finalize func), the signer filter all transaction in this block, compare the data field of transaction.

If the data match the vote pattern (ufo:1:evnet:vote), then the signer will deal with this vote, check validation of the transaction and keep the valid vote in HeaderExtra.CurrentBlockVotes.

In the meaning time, this process will filter the transaction related to all voter, not just vote in current block.

You can find details to process votes in processCustomTx func.

```
// Calculate Votes from transaction in this block, write into header.Extra
func (a *Alien) processCustomTx(chain consensus.ChainReader, header *types.Header, state *state.StateDB, txs []*types.Transaction) ([]Vote, []Vote, []Confirmation, error) {
	...

```

All node will update their Snapshot by snapshot func, which by the information in HeaderExtra, and use Snapshot to verifySeal and Seal.

#### Howto Calculate the Top Signer and Order

* Alien use the balance of voter as the stake of candidate, keep in Snapshot.Tally for each candidate.
* When balance of voter change ,the Snapshot.Tally change.
* After epoch number of block, if the voter not renew the vote Transaction and there is enough signer left, then the vote is expired
```
func (s *Snapshot) apply(headers []*types.Header) (*Snapshot, error) {
    ...
}
```
* For each loop (number of maxsignercount in Alien.config ), the snapshot will calculate the top signer by Snapshot.Tally
* The order of signer is decide by the hash value of last block.
```
func (s *Snapshot) createSignerQueue() ([]common.Address, error) {
    ...
}
```
#### Confirm block by Transaction

Confirm operation is a transaction which from signer to itself. The transaction write custom information (ufo:1:event:confirm:123) in data.

This transaction be created in [worker.go](../miner/worker.go) by the miner, because the processCustomTx func will check the validation of sender, which must be signer in headerExtra.SignerQueue.









