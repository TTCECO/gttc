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
	"cloud.google.com/go/firestore"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/TTCECO/gttc/extra/browserdb"
	_ "github.com/go-sql-driver/mysql"
	"gopkg.in/mgo.v2"
	"os"
	"strings"
)

type TTCBrowserDB struct {
	driver       string
	mysqlDB      *sql.DB
	mongoDB      *mgo.Database
	mongoSession *mgo.Session
	fireContext  context.Context
	fireClient   *firestore.Client
}

var (
	errProjectIDMissing = errors.New("Set Firebase project ID via GCLOUD_PROJECT env variable.")
)

func (b *TTCBrowserDB) Open(driver string, ip string, port int, user string, password string, DBName string) error {
	b.driver = strings.ToLower(driver)

	if b.driver == browserdb.MySQLDriver {
		db, err := sql.Open(driver, fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", user, password, ip, port, DBName))
		if err != nil {
			return err
		}
		_, err = db.Exec(fmt.Sprintf("use %s;", DBName))
		if err != nil {
			return err
		}
		b.mysqlDB = db
	} else if b.driver == browserdb.MongoDriver {
		session, err := mgo.Dial(fmt.Sprintf("%s:%s@%s:%d", user, password, ip, port))
		if err != nil {
			return err
		}
		session.SetMode(mgo.Monotonic, true)
		b.mongoSession = session
		b.mongoDB = b.mongoSession.DB(DBName)

	} else if b.driver == browserdb.FirestoreDriver {
		b.fireContext = context.Background()
		projectID := os.Getenv("GCLOUD_PROJECT")
		if projectID == "" {
			return errProjectIDMissing
		}
		client, err := firestore.NewClient(b.fireContext, projectID)
		if err != nil {
			return err
		}
		b.fireClient = client
	} else {
		return errors.New(fmt.Sprintf("%s database is not support", driver))
	}

	return nil
}

func (b *TTCBrowserDB) Close() error {
	if b.driver == browserdb.MySQLDriver && b.mysqlDB != nil {
		return b.mysqlDB.Close()
	} else if b.driver == browserdb.MongoDriver && b.mongoSession != nil {
		b.mongoSession.Close()
		return nil
	} else if b.driver == browserdb.FirestoreDriver && b.fireClient != nil {
		return b.fireClient.Close()
	} else {
		return nil
	}
}

func (b *TTCBrowserDB) GetDriver() string {
	return b.driver
}

func (b *TTCBrowserDB) CreateDefaultTable() error {
	return nil
}



