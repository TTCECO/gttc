// Copyright 2018 The gttc Authors
// This file is part of the gttc library.
//
// The gttc library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gttc library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gttc library. If not, see <http://www.gnu.org/licenses/>.

package tbweb

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/TTCECO/gttc/extra/browserdb/tbdb"
	"github.com/TTCECO/gttc/node"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

var (
	errRPCResultMissing      = errors.New("rpc result missing")
	errRPCResponseIdNotMatch = errors.New("response id not match request id")
)

type TTCBrowserWeb struct {
	port uint64
	e    *echo.Echo
	db   *tbdb.TTCBrowserDB
}

func getIndex(c echo.Context) error {
	return c.HTML(http.StatusOK, "<b>Hellow World!</b>")
}

func getLocalRPC(method string, params []interface{}, result *map[string]interface{}) error {

	requestId := rand.Intn(100)
	localURL := fmt.Sprintf("http://127.0.0.1:%d", node.DefaultHTTPPort)
	contentType := "application/json"
	data := map[string]interface{}{"jsonrpc": "2.0", "method": method, "params": params, "id": requestId}
	jsonValue, _ := json.Marshal(data)
	resp, err := http.Post(localURL, contentType, bytes.NewBuffer(jsonValue))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(body, result); err != nil {
		return err
	}

	if responseId, ok := (*result)["id"]; !ok || responseId.(int) != requestId {
		return errRPCResponseIdNotMatch
	}
	return nil
}

func getBalance(address string) (string, error) {
	var res map[string]interface{}
	err := getLocalRPC("eth_getBalance", []interface{}{address, "latest"}, &res)
	if err != nil {
		return "", err
	}
	if result, ok := res["result"]; ok {
		return result.(string), nil
	}
	return "", errRPCResultMissing
}

func getVote(address string) (string, error) {
	var res map[string]interface{}
	err := getLocalRPC("alien_getSnapshot", []interface{}{}, &res)
	if err != nil {
		return "", err
	}

	if result, ok := res["result"]; ok {
		if votes, ok := result.(map[string]interface{})["votes"]; ok {
			if vote, ok := votes.(map[string]interface{})[address]; ok {
				stake := vote.(map[string]interface{})["Stake"].(float64)
				return strconv.FormatFloat(stake, 'E', -1, 64), nil
			}
		}
	}

	return "", errRPCResultMissing
}

func (t *TTCBrowserWeb) queryAddress(c echo.Context) error {
	address := c.QueryParam("address")
	var balance string
	var errBalance error
	var vote string
	var errVote error
	var txs []map[string]interface{}
	var errTxs error
	wg := sync.WaitGroup{}
	wg.Add(3)
	go func() {
		defer wg.Done()
		balance, errBalance = getBalance(address)
	}()
	go func() {
		defer wg.Done()
		vote, errVote = getVote(address)
	}()
	go func() {
		defer wg.Done()
		txs, errTxs = t.db.MongoQuery("txs", map[string]interface{}{"from": strings.ToLower(address)}, 0, 10)
	}()
	wg.Wait()
	if errBalance != nil {
		return c.HTML(http.StatusOK, "Balance err : "+errBalance.Error())
	}
	if errVote != nil {
		vote = "no vote"
	}
	if errTxs != nil {
		return c.HTML(http.StatusOK, "Transaction err : "+errTxs.Error())
	}

	result := "<html><body>"
	result += "<b> Address </b> " + c.QueryParam("address") + "</br>"
	result += "<b> Balance </b> " + balance + "</br>"
	result += "<b> Vote </b> " + vote + "</br>"
	result += "<b> ==================</br>"

	for _, tx := range txs {
		result += "<b> From </b> " + tx["from"].(string) + "</br>"
		result += "<b> To </b> " + tx["to"].(string) + "</br>"
		result += "<b> Value </b> " + tx["value"].(string) + "</br>"
	}
	result += "</body></html>"
	return c.HTML(http.StatusOK, result)

}

func (t *TTCBrowserWeb) New(port uint64, db *tbdb.TTCBrowserDB) {
	if t.e == nil {
		t.e = echo.New()
		t.port = port
		t.db = db

		t.e.GET("/address", t.queryAddress)
		t.e.GET("/", getIndex)
		t.e.Use(middleware.Gzip())
		t.e.Use(middleware.Recover())

	}
}
func (t *TTCBrowserWeb) Use(params ...interface{}) {
	if len(params) > 0 {
		f := params[0].(echo.MiddlewareFunc)
		t.e.Use(f)
	}
}

func (t *TTCBrowserWeb) GET(params ...interface{}) {
	if len(params) > 2 {
		path := params[0].(string)
		h := params[1].(echo.HandlerFunc)
		m := make([]echo.MiddlewareFunc, len(params)-2)
		for i, item := range params[2:] {
			m[i] = item.(echo.MiddlewareFunc)
		}

		t.e.GET(path, h, m...)
	}

}
func (t *TTCBrowserWeb) POST(params ...interface{}) {
	if len(params) > 2 {
		path := params[0].(string)
		h := params[1].(echo.HandlerFunc)
		m := make([]echo.MiddlewareFunc, len(params)-2)
		for i, item := range params[2:] {
			m[i] = item.(echo.MiddlewareFunc)
		}

		t.e.POST(path, h, m...)
	}
}

func (t *TTCBrowserWeb) Start() error {
	t.e.Logger.Fatal(t.e.Start(fmt.Sprintf(":%d", t.port)))
	return nil
}
