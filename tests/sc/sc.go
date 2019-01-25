package main

import (
	"fmt"
	"math/big"
	"os"
	"strconv"

	"github.com/TTCECO/gttc/common"
	"github.com/TTCECO/gttc/core/types"
	"github.com/TTCECO/gttc/crypto"
	"github.com/TTCECO/gttc/rlp"
	"github.com/TTCECO/gttc/rpc"
)

var scAddressList = []string{}
var nodeAddressList = []string{}
var pkList = []string{}

func main() {
	if len(scAddressList) < 36 || len(nodeAddressList) < 36 || len(pkList) < 36 {
		fmt.Println(" set nodeList ")
		return
	}

	if len(os.Args) < 5 {
		fmt.Println("arg missing ")
		return
	}

	ip := string(os.Args[1])
	fmt.Println("ip : ", ip)

	port, err := strconv.ParseInt(os.Args[2], 10, 64)
	if err != nil {
		fmt.Println("port err", err)
		return
	}
	fmt.Println("port : ", port)

	operType := 0
	if string(os.Args[3]) == "del" {
		operType = 5
	} else if string(os.Args[3]) == "add" {
		operType = 4
	} else {
		fmt.Println("operType err ")
		return
	}
	fmt.Println("operType : ", operType)

	scHash := string(os.Args[4])
	fmt.Println("scHash : ", scHash)

	client, err := rpc.Dial(fmt.Sprintf("http://%s:%d", ip, port))
	if err != nil {
		fmt.Println("rpc.Dial err", err)
		return
	}
	var result string
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
	if operType == 5 {
		// remove side chain
		amount := big.NewInt(0)
		fromAddress := nodeAddressList[1]
		toAddress := scAddressList[0]
		pKey := pkList[1]

		dataStr := "ufo:1:event:proposal:proposal_type:5:vlcnt:2:schash:"
		dataStr += scHash
		txhash := sendTx(client, chainID, fromAddress, pKey, toAddress, amount, []byte(dataStr))

		amount = big.NewInt(0)
		fromAddress = nodeAddressList[0]
		toAddress = scAddressList[0]
		pKey = pkList[0]
		// here only need one declare, because the tally of this address is larger than 2/3 +1
		dataStr = "ufo:1:event:declare:decision:yes:hash:"
		dataStr += txhash
		txhash = sendTx(client, chainID, fromAddress, pKey, toAddress, amount, []byte(dataStr))
		fmt.Println("declare res hash : ", txhash)
	} else if operType == 4 {
		// add side chain
		amount := big.NewInt(0)
		fromAddress := nodeAddressList[1]
		toAddress := scAddressList[0]
		pKey := pkList[1]

		dataStr := "ufo:1:event:proposal:proposal_type:4:vlcnt:2:sccount:1:screward:50:schash:"
		dataStr += scHash
		txhash := sendTx(client, chainID, fromAddress, pKey, toAddress, amount, []byte(dataStr))

		amount = big.NewInt(0)
		fromAddress = nodeAddressList[0]
		toAddress = scAddressList[0]
		pKey = pkList[0]

		dataStr = "ufo:1:event:declare:decision:yes:hash:"
		dataStr += txhash
		txhash = sendTx(client, chainID, fromAddress, pKey, toAddress, amount, []byte(dataStr))

		fmt.Println("declare res hash : ", txhash)
		// set side chain coinbase
		for i := 0; i < 36; i++ {

			amount = big.NewInt(10)
			fromAddress = nodeAddressList[i]
			toAddress = scAddressList[i]
			pKey = pkList[i]
			dataStr = "ufo:1:sc:setcb:"
			dataStr += scHash

			txhash = sendTx(client, chainID, fromAddress, pKey, toAddress, amount, []byte(dataStr))

		}

	}
}

func sendTx(client *rpc.Client, chainID int64, fromAddress string, pKey string, toAddress string, amount *big.Int, data []byte) string {
	var result string

	// start to send transaction
	// get nonce
	err := client.Call(&result, "eth_getTransactionCount", fromAddress, "latest")
	if err != nil {
		fmt.Println("client.nonce err", err)
		return ""
	}
	fmt.Printf("nonce : %s\n", result)
	nonce, err := strconv.ParseInt(result[2:], 16, 64)
	if err != nil {
		fmt.Println("nonce parse fail", err)
		return ""
	}
	fmt.Printf("nonce : %d\n", nonce)

	to := common.HexToAddress(toAddress)
	privateKey, err := crypto.HexToECDSA(pKey)
	if err != nil {
		fmt.Println("create private key err :", err)
		return ""
	}

	value := new(big.Int).Mul(big.NewInt(1e+18), amount)
	fmt.Println("value :", value)
	tx := types.NewTransaction(uint64(nonce), to, value, uint64(2000000), big.NewInt(2010000), data)

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(big.NewInt(chainID)), privateKey)
	txData, err := rlp.EncodeToBytes(signedTx)
	if err != nil {
		fmt.Println("rlp Encode fail", err)
		return ""
	}
	err = client.Call(&result, "eth_sendRawTransaction", common.ToHex(txData))
	if err != nil {
		fmt.Println("send Transaction fail", err)
		return ""
	}
	fmt.Println("result txHash:", result)
	return result
}
