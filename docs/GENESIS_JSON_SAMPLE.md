## Sample of genesis.json

The follow is the content of genesis.json create by the instruction of HOWTO_RUNNING_TEST_ON_PRIVATE_NETWORK.md.

we already deploy a private network by this genesis.json file online. Anyone can do test on this network.

#### connection instruction


```
> mkdir node1
> gttc --datadir node1/ init genesis.json
> gttc --datadir node1/ --syncmode 'full' --rpcapi 'personal,db,eth,net,web3,txpool,miner,net' --bootnodes 'enode://f98bd1311c937f4314adacf2a258f88a470c0b0b199c1b2098d5e4c4ec91797a95525d1f62bdb09c251aa3b0aa0f92b212111cbc62b6dce732f10eca10d22f0e@47.105.142.208:30339,enode://bef0466f865d1abbe8e9090805fa30250c013b9d41ad15353bc6e5d58591fb15af1ac6709ac08f8c7f422939617454b207ab4696ac28c8cd4c33eb5d52136912@47.105.140.129:30331,enode://25a64b450d0d23d36f8326e2ee79b9a4f072bc6518981f19c761164840dcec8497b7bfa2ed5dfbe189ad05a6bfc1e69cdb05528d15b9a06f1d923ce8d16ad560@47.105.131.192:30325,enode://30d66152c08d7fdb50269ae2af0f2d39b98a081564662b9ca9267e8cba6c43a61262bdd6435657791ff897a23b4e9f1a43ca3f31c2d517f09f735955b0061666@47.105.78.210:30319' --networkid 8434

```


#### genesis.json

* [genesis.json](genesis.json)  : `genesis.json file for the testnet we deploy`