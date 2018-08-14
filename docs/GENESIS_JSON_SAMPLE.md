## Sample of genesis.json

The follow is the content of genesis.json create by the instruction of HOWTO_RUNNING_TEST_ON_PRIVATE_NETWORK.md.

we already deploy a private network by this genesis.json file online. Anyone can do test on this network.

#### connection instruction


```
> mkdir node1
> gttc --datadir node1/ init genesis.json
> gttc --datadir node1/ --syncmode 'full' --rpcapi 'personal,db,eth,net,web3,txpool,miner,net' --bootnodes 'enode://68142f6c02507e064236f3cab156b2b04392e868da87d910af61b066b6fcea1a95a21bc71799c7b0827e26dea4b23a494b1c9976635d4cf52a4699fc9502d940@39.106.104.30:30312' --networkid 1084

```


#### genesis.json

```
{
  "config": {
    "chainId": 1084,
    "homesteadBlock": 1,
    "eip150Block": 2,
    "eip150Hash": "0x0000000000000000000000000000000000000000000000000000000000000000",
    "eip155Block": 3,
    "eip158Block": 3,
    "byzantiumBlock": 4,
    "alien": {
      "period": 3,
      "epoch": 3000,
      "maxSignersCount": 7,
      "minVoterBalance": 10000000000000000000000,
      "genesisTimestamp": 1533611102,
      "signers": [
        "0x755200bcb42ea69643a1bed4c6e79a9b55e8c29d",
        "0x8d07f644daf08062164073205ae4f10dd2ca8b32",
        "0x8733cf7726055b6d83b28b54b0de39276fddb3da",
        "0xc5e094488e4d1284d5a1a849c1110ad14a8700bb",
        "0x1962e91cf047accd9d157fbb2661a8242a2293ce",
        "0xc9edbb408d307072a50a0282b09ac10ee1dc1610",
        "0xbbdacd1dbd49e3df60ae5c683be7739d237b7328"
      ]
    }
  },
  "nonce": "0x0",
  "timestamp": "0x5b5ea585",
  "extraData": "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
  "gasLimit": "0x47b760",
  "difficulty": "0x1",
  "mixHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
  "coinbase": "0x0000000000000000000000000000000000000000",
  "alloc": {
    "1962e91cf047accd9d157fbb2661a8242a2293ce": {
      "balance": "0x20000000000000000000000000000000000000"
    },
    "7544bf9c90d175da395b8d08fcaf34da0a3e0688": {
      "balance": "0x20000000000000000000000000000000000000"
    },
    "755200bcb42ea69643a1bed4c6e79a9b55e8c29d": {
      "balance": "0x20000000000000000000000000000000000000"
    },
    "8733cf7726055b6d83b28b54b0de39276fddb3da": {
      "balance": "0x20000000000000000000000000000000000000"
    },
    "8d07f644daf08062164073205ae4f10dd2ca8b32": {
      "balance": "0x20000000000000000000000000000000000000"
    },
    "aafeaaf6111762fea733ff7b4c8b59ac69316385": {
      "balance": "0x20000000000000000000000000000000000000"
    },
    "bbdacd1dbd49e3df60ae5c683be7739d237b7328": {
      "balance": "0x20000000000000000000000000000000000000"
    },
    "c5e094488e4d1284d5a1a849c1110ad14a8700bb": {
      "balance": "0x20000000000000000000000000000000000000"
    },
    "c9edbb408d307072a50a0282b09ac10ee1dc1610": {
      "balance": "0x60000000000000000000000000000000000000"
    }
  },
  "number": "0x0",
  "gasUsed": "0x0",
  "parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000"
}
```