## Vote to change the Signer node

#### Vote operation

Vote operation is a transaction which from voter to candidate. The transaction write custom information (ufo:1:event:vote) in data.

* ufo   :   prefix of custom data
* 1     :   custom info version
* event :   category of info (event, oplog ...)
* vote  :   vote event

The sample vote in console is like this

```
> eth.sendTransaction({from:"0x7544bf9c90d175da395b8d08fcaf34da0a3e0688",to:"0xaafeaaf6111762fea733ff7b4c8b59ac69316385",value:0,data:web3.toHex("ufo:1:event:vote")})

```
*   Voter address is 0x7544bf9c90d175da395b8d08fcaf34da0a3e0688
*   Candidate address is 0xaafeaaf6111762fea733ff7b4c8b59ac69316385
*   The balance of voter is used to calculate the number of tickets.


#### Confirm operation

Confirm operation is a transaction which from signer to itself. The transaction write custom information (ufo:1:event:confirm:123) in data.

* ufo   :   prefix of custom data
* 1     :   custom info version
* event :   category of info (event, oplog ...)
* confirm  :   confirm event
* 123   :   block numb be confirmed by this transaction sender

Only the transaction from signer of current loop is valid and will be recorded, the largest block number which received more than (2 * MaxSignerCount)/3 +1 is the last block be confirmed and the result will be record in the ConfirmedBlockNumber in header.extra.

#### Check the snapshot status
The name of our DPOS-PBFT Algorithm is alien, which provide API with three function.

```
> alien
{
  getSnapshot: function(),
  getSnapshotAtHash: function(),
  getSnapshotAtNumber: function()
}
```

Anyone can check the current DPOS status by alien.getSnapshot() or status by block number/hash.

```
> alien.getSnapshotAtNumber(48)
  {
    confirmedNumber: 44,
    confirms: {
      41: ["0x8feba63259ef7e79da13246dc20ff79fe446a478", "0xcc9c08721c7d8a792e238e80f6c20cb066e919a1", "0x7c352585cd7549bfbc0a7e88bf820fa574174598"],
      42: ["0x7c352585cd7549bfbc0a7e88bf820fa574174598", "0xcc9c08721c7d8a792e238e80f6c20cb066e919a1", "0x8feba63259ef7e79da13246dc20ff79fe446a478"],
      43: ["0xcc9c08721c7d8a792e238e80f6c20cb066e919a1", "0x7c352585cd7549bfbc0a7e88bf820fa574174598", "0x8feba63259ef7e79da13246dc20ff79fe446a478"],
      44: ["0x7c352585cd7549bfbc0a7e88bf820fa574174598", "0xcc9c08721c7d8a792e238e80f6c20cb066e919a1", "0x8feba63259ef7e79da13246dc20ff79fe446a478"],
      45: ["0x8feba63259ef7e79da13246dc20ff79fe446a478", "0x7c352585cd7549bfbc0a7e88bf820fa574174598", "0xcc9c08721c7d8a792e238e80f6c20cb066e919a1"],
      46: ["0x8feba63259ef7e79da13246dc20ff79fe446a478", "0xcc9c08721c7d8a792e238e80f6c20cb066e919a1", "0x7c352585cd7549bfbc0a7e88bf820fa574174598"]
    },
    hash: "0x27009a5d7480cb2802e45b9d0ef2547ce7ad5e720d9281b0d628f314998a6416",
    headerTime: 1533341653,
    historyHash: ["0xfb741ff569db0a9277111285d157a0412c4dc6e30509834e7e87ce84755db336", "0x855589fcf6fb8157b29df69cba0393737a1c15a7427666f5e84b34c8a5680523", "0x7b37e235e0686c8defaa0068b0508fe55921bd5e27bf0f1ee3e6909184dc74f4", "0x664bb4e6889acfd8bd4c277ba109cd9fbaf1366ce8472761b050451c01cf1a04", "0xd3d64b715e8568be7d86a82dbfd9a801013132d4229d96963f3869b584426f7a", "0x68a1241b27c209ab584155a670a0473f29ae302eb2fe47d1a75b1eea45898ab2", "0x4c8e6e969c950810c2acf128866f2aa416dc56b1ef2987281de4812622ebb44f", "0x3c7047ba4574bf1e49c9e068d82e1b2c3f59e958658d897f4836db23f6ffcc3c", "0xc05b07cb1c00be507742bd9de8702a42d8cf77f43c566f52196f5b0bc47d084c", "0xd1f96774a2feb4c21add4aeb14ac0d92d5357b1cdc53cead0ed9df459ce308ac", "0xc95bd9b4fd9f9d4ca5a38d2433f6f9d6f48027098316d8b9d95fa1436639338d", "0x300addbe720652487d69c62b37d7281298200b2c7c68170a24dad8503c73940e", "0x6ede482139d9a8f5a7bc385b16c5d3c59ad11715091ad1609aa451f605fcc633", "0x27009a5d7480cb2802e45b9d0ef2547ce7ad5e720d9281b0d628f314998a6416"],
    loopStartTime: 1533341639,
    number: 48,
    punished: {
      0x7c352585cd7549bfbc0a7e88bf820fa574174598: 880,
      0x8feba63259ef7e79da13246dc20ff79fe446a478: 950,
      0xcc9c08721c7d8a792e238e80f6c20cb066e919a1: 1370
    },
    signers: ["0x7c352585cd7549bfbc0a7e88bf820fa574174598", "0x8feba63259ef7e79da13246dc20ff79fe446a478", "0xcc9c08721c7d8a792e238e80f6c20cb066e919a1", "0x7c352585cd7549bfbc0a7e88bf820fa574174598", "0x8feba63259ef7e79da13246dc20ff79fe446a478", "0xcc9c08721c7d8a792e238e80f6c20cb066e919a1", "0x7c352585cd7549bfbc0a7e88bf820fa574174598"],
    tally: {
      0x7c352585cd7549bfbc0a7e88bf820fa574174598: 9.046256971665328e+74,
      0x8feba63259ef7e79da13246dc20ff79fe446a478: 9.046256971665328e+74,
      0xcc9c08721c7d8a792e238e80f6c20cb066e919a1: 9.046256971665328e+74
    },
    voters: {
      0x7c352585cd7549bfbc0a7e88bf820fa574174598: 0,
      0x8feba63259ef7e79da13246dc20ff79fe446a478: 0,
      0xcc9c08721c7d8a792e238e80f6c20cb066e919a1: 0
    },
    votes: {
      0x7c352585cd7549bfbc0a7e88bf820fa574174598: {
        Candidate: "0x7c352585cd7549bfbc0a7e88bf820fa574174598",
        Stake: 9.046256971665328e+74,
        Voter: "0x7c352585cd7549bfbc0a7e88bf820fa574174598"
      },
      0x8feba63259ef7e79da13246dc20ff79fe446a478: {
        Candidate: "0x8feba63259ef7e79da13246dc20ff79fe446a478",
        Stake: 9.046256971665328e+74,
        Voter: "0x8feba63259ef7e79da13246dc20ff79fe446a478"
      },
      0xcc9c08721c7d8a792e238e80f6c20cb066e919a1: {
        Candidate: "0xcc9c08721c7d8a792e238e80f6c20cb066e919a1",
        Stake: 9.046256971665328e+74,
        Voter: "0xcc9c08721c7d8a792e238e80f6c20cb066e919a1"
      }
    }
  }



```
*   confirmNumber:  Block number confirmed when the snapshot was created
*   confirms    :   The signer confirm given block number
*	hash        :   Block hash where the snapshot was created
*   historyHash :   Block hash list for two recent loop, used to calculate the order of next loop.
*	headerTime  :   Time of the current header
*	loopStartTime:  Start Time of the current loop, used to calculate the right miner(signer) by time
*	number      :   Block number where the snapshot was created
*	punished    :   Each time of missing seal will be record,
*	signers     :   Signers queue in current loop
*	tally       :   Number of tickets (sum of voters balance) for each candidate address
*	voters      :   block number for each voter address, the vote will be expired after one epoch
*	votes       :   All validate votes from genesis block