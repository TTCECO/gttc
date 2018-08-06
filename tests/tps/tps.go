package main

import (
	"fmt"
	"github.com/TTCECO/gttc/common"
	"github.com/TTCECO/gttc/core/types"
	"github.com/TTCECO/gttc/crypto"
	"github.com/TTCECO/gttc/rlp"
	"github.com/TTCECO/gttc/rpc"
	"strconv"

	"math/big"
)

const (
	fromAddress = "0xb7055b228EFE49219231ef3F3A384f3062570Ab1"
	toAddress   = "0x2a84f498d27805D49a92277eDBe670b83036F14A"
	pKey        = "65f9e4ee1dbfc4a751dc4e2f6037a8760ce203e1342f84a55f6266d52ae3c96f"
	count       = 10000
)

func main() {

	client, err := rpc.Dial("http://localhost:8503")
	if err != nil {
		fmt.Println("rpc.Dial err", err)
		return
	}
	var result string
	err = client.Call(&result, "eth_getTransactionCount", fromAddress, "latest")
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

	err = client.Call(&result, "net_version")
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

	for i := nonce; i < nonce+count; i++ {

		tx := types.NewTransaction(uint64(i), toAddress, big.NewInt(1), uint64(100000000), big.NewInt(21000000), []byte{})
		signedTx, err := types.SignTx(tx, types.NewEIP155Signer(big.NewInt(chainID)), privateKey)
		data, err := rlp.EncodeToBytes(signedTx)
		if err != nil {
			fmt.Println("rlp Encode fail", err)
			return
		}
		err = client.Call(&result, "eth_sendRawTransaction", common.ToHex(data))
		if err != nil {
			fmt.Println("send Transaction fail", err)
			return
		}

		if i%300 == 0 {
			fmt.Printf(" nonce is : %d \n", i)
			fmt.Printf("send Transaction result : %s \n", result)
		}
	}

	return

}
