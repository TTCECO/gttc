.PHONY: build cluster test

default: build

build:
	go build -o bin/swarm    github.com/TTCECO/gttc/cmd/swarm
	go build -o bin/geth     github.com/TTCECO/gttc/cmd/geth
	go build -o bin/bootnode github.com/TTCECO/gttc/cmd/bootnode

cluster: build
	scripts/boot-cluster.sh

test:
	go test -v github.com/TTCECO/gttc/swarm/...
