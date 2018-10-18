# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

.PHONY: gttc android ios gttc-cross swarm evm all test clean
.PHONY: gttc-linux gttc-linux-386 gttc-linux-amd64 gttc-linux-mips64 gttc-linux-mips64le
.PHONY: gttc-linux-arm gttc-linux-arm-5 gttc-linux-arm-6 gttc-linux-arm-7 gttc-linux-arm64
.PHONY: gttc-darwin gttc-darwin-386 gttc-darwin-amd64
.PHONY: gttc-windows gttc-windows-386 gttc-windows-amd64

GOBIN = $(shell pwd)/build/bin
GO ?= latest

gttc:
	build/env.sh go run build/ci.go install ./cmd/gttc
	@echo "Done building."
	@echo "Run \"$(GOBIN)/gttc\" to launch gttc."

swarm:
	build/env.sh go run build/ci.go install ./cmd/swarm
	@echo "Done building."
	@echo "Run \"$(GOBIN)/swarm\" to launch swarm."

all:
	build/env.sh go run build/ci.go install

android:
	build/env.sh go run build/ci.go aar --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/gttc.aar\" to use the library."

ios:
	build/env.sh go run build/ci.go xcode --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/Geth.framework\" to use the library."

test: all
	build/env.sh go run build/ci.go test

lint: ## Run linters.
	build/env.sh go run build/ci.go lint

clean:
	rm -fr build/_workspace/pkg/ $(GOBIN)/*

# The devtools target installs tools required for 'go generate'.
# You need to put $GOBIN (or $GOPATH/bin) in your PATH to use 'go generate'.

devtools:
	env GOBIN= go get -u golang.org/x/tools/cmd/stringer
	env GOBIN= go get -u github.com/kevinburke/go-bindata/go-bindata
	env GOBIN= go get -u github.com/fjl/gencodec
	env GOBIN= go get -u github.com/golang/protobuf/protoc-gen-go
	env GOBIN= go install ./cmd/abigen
	@type "npm" 2> /dev/null || echo 'Please install node.js and npm'
	@type "solc" 2> /dev/null || echo 'Please install solc'
	@type "protoc" 2> /dev/null || echo 'Please install protoc'

# Cross Compilation Targets (xgo)

gttc-cross: gttc-linux gttc-darwin gttc-windows gttc-android gttc-ios
	@echo "Full cross compilation done:"
	@ls -ld $(GOBIN)/gttc-*

gttc-linux: gttc-linux-386 gttc-linux-amd64 gttc-linux-arm gttc-linux-mips64 gttc-linux-mips64le
	@echo "Linux cross compilation done:"
	@ls -ld $(GOBIN)/gttc-linux-*

gttc-linux-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/386 -v ./cmd/gttc
	@echo "Linux 386 cross compilation done:"
	@ls -ld $(GOBIN)/gttc-linux-* | grep 386

gttc-linux-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/amd64 -v ./cmd/gttc
	@echo "Linux amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gttc-linux-* | grep amd64

gttc-linux-arm: gttc-linux-arm-5 gttc-linux-arm-6 gttc-linux-arm-7 gttc-linux-arm64
	@echo "Linux ARM cross compilation done:"
	@ls -ld $(GOBIN)/gttc-linux-* | grep arm

gttc-linux-arm-5:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-5 -v ./cmd/gttc
	@echo "Linux ARMv5 cross compilation done:"
	@ls -ld $(GOBIN)/gttc-linux-* | grep arm-5

gttc-linux-arm-6:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-6 -v ./cmd/gttc
	@echo "Linux ARMv6 cross compilation done:"
	@ls -ld $(GOBIN)/gttc-linux-* | grep arm-6

gttc-linux-arm-7:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-7 -v ./cmd/gttc
	@echo "Linux ARMv7 cross compilation done:"
	@ls -ld $(GOBIN)/gttc-linux-* | grep arm-7

gttc-linux-arm64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm64 -v ./cmd/gttc
	@echo "Linux ARM64 cross compilation done:"
	@ls -ld $(GOBIN)/gttc-linux-* | grep arm64

gttc-linux-mips:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips --ldflags '-extldflags "-static"' -v ./cmd/gttc
	@echo "Linux MIPS cross compilation done:"
	@ls -ld $(GOBIN)/gttc-linux-* | grep mips

gttc-linux-mipsle:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mipsle --ldflags '-extldflags "-static"' -v ./cmd/gttc
	@echo "Linux MIPSle cross compilation done:"
	@ls -ld $(GOBIN)/gttc-linux-* | grep mipsle

gttc-linux-mips64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64 --ldflags '-extldflags "-static"' -v ./cmd/gttc
	@echo "Linux MIPS64 cross compilation done:"
	@ls -ld $(GOBIN)/gttc-linux-* | grep mips64

gttc-linux-mips64le:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64le --ldflags '-extldflags "-static"' -v ./cmd/gttc
	@echo "Linux MIPS64le cross compilation done:"
	@ls -ld $(GOBIN)/gttc-linux-* | grep mips64le

gttc-darwin: gttc-darwin-386 gttc-darwin-amd64
	@echo "Darwin cross compilation done:"
	@ls -ld $(GOBIN)/gttc-darwin-*

gttc-darwin-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/386 -v ./cmd/gttc
	@echo "Darwin 386 cross compilation done:"
	@ls -ld $(GOBIN)/gttc-darwin-* | grep 386

gttc-darwin-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/amd64 -v ./cmd/gttc
	@echo "Darwin amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gttc-darwin-* | grep amd64

gttc-windows: gttc-windows-386 gttc-windows-amd64
	@echo "Windows cross compilation done:"
	@ls -ld $(GOBIN)/gttc-windows-*

gttc-windows-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/386 -v ./cmd/gttc
	@echo "Windows 386 cross compilation done:"
	@ls -ld $(GOBIN)/gttc-windows-* | grep 386

gttc-windows-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/amd64 -v ./cmd/gttc
	@echo "Windows amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gttc-windows-* | grep amd64
