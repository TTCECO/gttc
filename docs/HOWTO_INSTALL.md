## Install gttc

You'll need `go` [installed](https://golang.org/doc/install)

#### Get Source Code

```
mkdir -p $GOPATH/src/github.com/TTCECO
cd $GOPATH/src/github.com/TTCECO
git clone https://github.com/TTCECO/gttc.git
cd gttc
```

#### Compile

```
go run build/ci.go install
```

#### Run

To start a node connect to testnet

```
build/bin/gttc --testnet
```

Or

[deploy your TestNet](HOWTO_RUNNING_TEST_ON_PRIVATE_NETWORK.md)
