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
	"encoding/json"
	"fmt"
	"github.com/TTCECO/gttc/extra/browserdb/tbdb"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"net/http"
	"strings"
)

type TTCBrowserWeb struct {
	port uint64
	e    *echo.Echo
	db   *tbdb.TTCBrowserDB
}

func (t *TTCBrowserWeb) getIndex(c echo.Context) error {
	return c.HTML(http.StatusOK, "<b>Hellow World!</b>")
}

func (t *TTCBrowserWeb) getCollection(c echo.Context) error {
	collection := c.Param("collection")
	var queryRes []map[string]interface{}

	if id := c.QueryParam("id"); id != "" {
		if res, err := t.db.FirestoreQueryById(collection, id); err != nil {
			return c.HTML(http.StatusOK, err.Error())
		} else {
			queryRes = append(queryRes, res)
		}
	} else {
		condition := make(map[string]interface{})
		for _, v := range strings.Split(c.QueryString(), "&"){
			k := strings.Split(v,"=")[0]
			condition[k] = c.QueryParam(k)
		}
		if res, err := t.db.FirestoreQuery(collection, condition); err != nil {
			return c.HTML(http.StatusOK, err.Error())
		} else {
			queryRes = res
		}
	}
	bolB, err := json.Marshal(queryRes)
	if err != nil {
		return c.HTML(http.StatusOK, err.Error())
	}
	return c.HTML(http.StatusOK, string(bolB))
}

func (t *TTCBrowserWeb) New(port uint64, db *tbdb.TTCBrowserDB) {
	if t.e == nil {
		t.e = echo.New()
		t.port = port
		t.e.GET("/", t.getIndex)
		t.e.GET("/:collection", t.getCollection)
		t.e.Use(middleware.Gzip())
		t.e.Use(middleware.Recover())
		t.db = db
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