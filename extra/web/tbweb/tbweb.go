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
	"fmt"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"net/http"
)

type TTCBrowserWeb struct {
	port uint64
	e    *echo.Echo
}

func getIndex(c echo.Context) error {
	return c.HTML(http.StatusOK, "<b>Hellow World!</b>")
}

func (t *TTCBrowserWeb) New(port uint64) {
	if t.e == nil {
		t.e = echo.New()
		t.port = port

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
