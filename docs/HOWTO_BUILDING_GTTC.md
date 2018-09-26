## Building gttc

The code of gttc is based on geth, so if your environment can build geth, it can build gttc as well.

You can find instruction docs in [https://github.com/ethereum/go-ethereum/wiki/Building-Ethereum](https://github.com/ethereum/go-ethereum/wiki/Building-Ethereum)


#### Quick Instruction to build gttc

```
unzip gttc-release-v0.0.4.zip

mv gttc $HOME/go/src/github.com/TTCECO/gttc

cd $HOME/go/src/github.com/TTCECO/gttc/cmd/gttc

go build

```

**$HOME/go/** is the default directory when you install golang, if you use a different workspace directory, you will need to replace **$HOME/go/** by **$GOPATH**.