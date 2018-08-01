## Vote to change the Signer node

#### Vote operation

Vote operation is a normal transaction which from voter to candidate. The transaction write custom information (ufo:1:event:vote) in data.

The sample vote in console is like this

```
> eth.sendTransaction({from:"0x7544bf9c90d175da395b8d08fcaf34da0a3e0688",to:"0xaafeaaf6111762fea733ff7b4c8b59ac69316385",value:0,data:web3.toHex("ufo:1:event:vote")})

```
*   Voter address is 0x7544bf9c90d175da395b8d08fcaf34da0a3e0688
*   Candidate address is 0xaafeaaf6111762fea733ff7b4c8b59ac69316385
*   The balance of voter is used to calculate the number of tickets.


#### Check the vote status
The name of our DPOS Algorithm is alien, which provide API with three function.

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
> alien.getSnapshot()
{
  hash: "0x2dde5f3fd1f9ef346abc4bcb6a29ad47c130da88c8dc86f29967be93d7225a0c",
  headerTime: 1533098803,
  loopStartTime: 1532980288,
  number: 16756,
  punished: {
    0x1962e91cf047accd9d157fbb2661a8242a2293ce: 3368,
    0x755200bcb42ea69643a1bed4c6e79a9b55e8c29d: 3689,
    0x8733cf7726055b6d83b28b54b0de39276fddb3da: 4030,
    0x8d07f644daf08062164073205ae4f10dd2ca8b32: 3712,
    0xbbdacd1dbd49e3df60ae5c683be7739d237b7328: 63296,
    0xc5e094488e4d1284d5a1a849c1110ad14a8700bb: 3623,
    0xc9edbb408d307072a50a0282b09ac10ee1dc1610: 2613
  },
  signers: ["0x755200bcb42ea69643a1bed4c6e79a9b55e8c29d", "0xbbdacd1dbd49e3df60ae5c683be7739d237b7328", "0xc9edbb408d307072a50a0282b09ac10ee1dc1610", "0xc5e094488e4d1284d5a1a849c1110ad14a8700bb", "0x8d07f644daf08062164073205ae4f10dd2ca8b32", "0x8733cf7726055b6d83b28b54b0de39276fddb3da", "0x1962e91cf047accd9d157fbb2661a8242a2293ce"],
  tally: {
    0x1962e91cf047accd9d157fbb2661a8242a2293ce: 7.1362384635298e+44,
    0x755200bcb42ea69643a1bed4c6e79a9b55e8c29d: 7.1362384635298e+44,
    0x8733cf7726055b6d83b28b54b0de39276fddb3da: 7.1362384635298e+44,
    0x8d07f644daf08062164073205ae4f10dd2ca8b32: 7.1362384635298e+44,
    0xbbdacd1dbd49e3df60ae5c683be7739d237b7328: 7.1362384635298e+44,
    0xc5e094488e4d1284d5a1a849c1110ad14a8700bb: 7.1362384635298e+44,
    0xc9edbb408d307072a50a0282b09ac10ee1dc1610: 2.1408715390589398e+45
  },
  voters: {
    0x1962e91cf047accd9d157fbb2661a8242a2293ce: 0,
    0x755200bcb42ea69643a1bed4c6e79a9b55e8c29d: 0,
    0x8733cf7726055b6d83b28b54b0de39276fddb3da: 0,
    0x8d07f644daf08062164073205ae4f10dd2ca8b32: 0,
    0xbbdacd1dbd49e3df60ae5c683be7739d237b7328: 0,
    0xc5e094488e4d1284d5a1a849c1110ad14a8700bb: 0,
    0xc9edbb408d307072a50a0282b09ac10ee1dc1610: 0
  },
  votes: {
    0x1962e91cf047accd9d157fbb2661a8242a2293ce: {
      Candidate: "0x1962e91cf047accd9d157fbb2661a8242a2293ce",
      Stake: 7.1362384635298e+44,
      Voter: "0x1962e91cf047accd9d157fbb2661a8242a2293ce"
    },
    0x755200bcb42ea69643a1bed4c6e79a9b55e8c29d: {
      Candidate: "0x755200bcb42ea69643a1bed4c6e79a9b55e8c29d",
      Stake: 7.1362384635298e+44,
      Voter: "0x755200bcb42ea69643a1bed4c6e79a9b55e8c29d"
    },
    0x8733cf7726055b6d83b28b54b0de39276fddb3da: {
      Candidate: "0x8733cf7726055b6d83b28b54b0de39276fddb3da",
      Stake: 7.1362384635298e+44,
      Voter: "0x8733cf7726055b6d83b28b54b0de39276fddb3da"
    },
    0x8d07f644daf08062164073205ae4f10dd2ca8b32: {
      Candidate: "0x8d07f644daf08062164073205ae4f10dd2ca8b32",
      Stake: 7.1362384635298e+44,
      Voter: "0x8d07f644daf08062164073205ae4f10dd2ca8b32"
    },
    0xbbdacd1dbd49e3df60ae5c683be7739d237b7328: {
      Candidate: "0xbbdacd1dbd49e3df60ae5c683be7739d237b7328",
      Stake: 7.1362384635298e+44,
      Voter: "0xbbdacd1dbd49e3df60ae5c683be7739d237b7328"
    },
    0xc5e094488e4d1284d5a1a849c1110ad14a8700bb: {
      Candidate: "0xc5e094488e4d1284d5a1a849c1110ad14a8700bb",
      Stake: 7.1362384635298e+44,
      Voter: "0xc5e094488e4d1284d5a1a849c1110ad14a8700bb"
    },
    0xc9edbb408d307072a50a0282b09ac10ee1dc1610: {
      Candidate: "0xc9edbb408d307072a50a0282b09ac10ee1dc1610",
      Stake: 2.1408715390589398e+45,
      Voter: "0xc9edbb408d307072a50a0282b09ac10ee1dc1610"
    }
  }
}


```
*	hash        :   Block hash where the snapshot was created
*	headerTime  :   Time of the current header
*	loopStartTime:  Start Time of the current loop, used to calculate the right miner(signer) by time
*	number      :   Block number where the snapshot was created
*	punished    :   Each time of missing seal will be record,
*	signers     :   Signers queue in current loop
*	tally       :   Number of tickets (sum of voters balance) for each candidate address
*	voters      :   block number for each voter address, the vote will be expired after one epoch
*	votes       :   All validate votes from genesis block