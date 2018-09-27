# Install gttc

## From Source

You'll need `go` [installed](https://golang.org/doc/install)

### Get Source Code

```
mkdir -p $GOPATH/src/github.com/TTCECO
cd $GOPATH/src/github.com/TTCECO
git clone https://github.com/TTCECO/gttc.git
cd gttc
```

### Compile

```
go run build/ci.go install
```

## Run

To start a node connect to testnet

```
gttc --testnet
```

Or

[HOWTO_RUNNING_TEST_ON_PRIVATE_NETWORK.md](HOWTO_RUNNING_TEST_ON_PRIVATE_NETWORK.md)
