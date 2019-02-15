package main

import (
	"fmt"
	"github.com/TTCECO/gttc/common"
	"github.com/TTCECO/gttc/core/types"
	"github.com/TTCECO/gttc/crypto"
	"github.com/TTCECO/gttc/rlp"
	"github.com/TTCECO/gttc/rpc"
	"os"
	"strconv"

	"math/big"
)

const (
	fromAddress = "0x74883A82Eb05c1EFfF9C40F6101b7D8E7DE52504"
	pKey        = "4dba440cab5a1768a5573ff0eba2700fffcd189c0a98383add184b271be7f3da"
	toAddress   = "0x2a84f498d27805D49a92277eDBe670b83036F14A"

	defaultCount      = 10000
	defaultPortStart  = 8501
	defaultPortLength = 1
)

func main() {

	count := int64(defaultCount)
	if len(os.Args) > 1 {
		argCount, err := strconv.ParseInt(os.Args[1], 10, 64)
		if err == nil {
			count = argCount
		}
	}
	fmt.Println("count : ", count)

	portStart := defaultPortStart
	if len(os.Args) > 2 {
		argPortStart, err := strconv.ParseInt(os.Args[2], 10, 64)
		if err == nil {
			portStart = int(argPortStart)
		}
	}
	fmt.Println("start port : ", portStart)

	portLen := defaultPortLength
	if len(os.Args) > 3 {
		argPortLength, err := strconv.ParseInt(os.Args[3], 10, 64)
		if err == nil {
			portLen = int(argPortLength)
		}
	}
	fmt.Println("start Len : ", portLen)

	cl := []*rpc.Client{}
	for i := 0; i < portLen; i++ {
		client, err := rpc.Dial(fmt.Sprintf("http://localhost:%d", i+portStart))
		if err != nil {
			fmt.Println("rpc.Dial err", err)
			return
		}
		cl = append(cl, client)
	}

	var result string
	err := cl[0].Call(&result, "eth_getTransactionCount", fromAddress, "latest")
	if err != nil {
		fmt.Println("client.nonce err", err)
		return
	}
	fmt.Printf("nonce : %s\n", result)
	nonce, err := strconv.ParseInt(result[2:], 16, 64)
	if err != nil {
		fmt.Println("nonce parse fail", err)
		return
	}
	fmt.Printf("nonce : %d\n", nonce)

	err = cl[0].Call(&result, "net_version")
	if err != nil {
		fmt.Println("get chain id fail", err)
		return
	}
	fmt.Printf("chainId: %s\n", result)
	chainID, err := strconv.ParseInt(result, 10, 64)
	if err != nil {
		fmt.Println("parse chain id fail", err)
		return
	}

	toAddress := common.HexToAddress(toAddress)
	privateKey, err := crypto.HexToECDSA(pKey)
	if err != nil {

		fmt.Println("create private key err :", err)
		return
	}

	txDataList := make([][]byte, count, count)

	for i := nonce; i < nonce+count; i++ {
		tx := types.NewTransaction(uint64(i), toAddress, big.NewInt(1), uint64(100000), big.NewInt(21000000), []byte{})
		signedTx, err := types.SignTx(tx, types.NewEIP155Signer(big.NewInt(chainID)), privateKey)
		data, err := rlp.EncodeToBytes(signedTx)
		if err != nil {
			fmt.Println("rlp Encode fail", err)
			return
		}
		txDataList[i-nonce] = data
	}

	fmt.Println("tx data is ready")

	wait := make(chan bool, portLen)
	for i := 0; i < portLen; i++ {

		go startSendTx(cl, txDataList, i, portLen, wait)
	}

	for i := 0; i < portLen; i++ {
		<-wait
	}

	return
}

func startSendTx(cl []*rpc.Client, txDataList [][]byte, mod int, portLen int, finish chan<- bool) {
	var result string
	for i := range txDataList {
		if i%portLen == mod {
			err := cl[mod].Call(&result, "eth_sendRawTransaction", common.ToHex(txDataList[i]))
			if err != nil {
				fmt.Println("send Transaction fail", err)
				return
			}

			if i%500 == 0 {
				fmt.Printf("nonce is : %d \n", i)
				fmt.Printf("send Transaction result : %s \n", result)
			}
		}
	}
	finish <- true
}
