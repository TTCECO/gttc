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

package tbdb

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
)

type TTCBrowserDB struct {
	db *sql.DB
}

func (b *TTCBrowserDB) Open(driver string, ip string, port int, user string, password string, DBName string) error {
	db, err := sql.Open(driver, fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", user, password, ip, port, DBName))
	if err != nil {
		return err
	}
	_, err = db.Exec(fmt.Sprintf("use %s;", DBName))
	if err != nil {
		return err
	}
	b.db = db
	return nil
}

func (b *TTCBrowserDB) Close() error {
	return b.db.Close()
}

func (b *TTCBrowserDB) CreateDefaultTable() error {

	return nil
}

func (b *TTCBrowserDB) SaveTx() error {

	return nil
}

func (b *TTCBrowserDB) SaveBlock() error {
	return nil
}

func (b *TTCBrowserDB) SaveSnapshot() error {
	return nil
}
