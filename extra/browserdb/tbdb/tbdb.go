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
	"gopkg.in/mgo.v2/bson"
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
		b.fireClient.Close()
		return nil
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

// todo :
func (b *TTCBrowserDB) MysqlExec(input string) error {
	_, err := b.mysqlDB.Exec(input)
	if err != nil {
		return err
	}
	return nil
}

// Mongo operate
func (b *TTCBrowserDB) MongoSave(collection string, data ...interface{}) error {
	return b.mongoDB.C(collection).Insert(data...)
}

func (b *TTCBrowserDB) MongoUpdate(collection string, condition bson.M, data bson.M) error {
	return b.mongoDB.C(collection).Update(condition, data)
}

func (b *TTCBrowserDB) MongoUpsert(collection string, condition bson.M, data bson.M) (*mgo.ChangeInfo, error) {
	return b.mongoDB.C(collection).Upsert(condition, data)
}

func (b *TTCBrowserDB) MongoExist(collection string, condition bson.M) bool {
	var res bson.M
	err := b.mongoDB.C(collection).Find(condition).One(&res)
	if err != nil || res == nil {
		return false
	}
	return true
}

// Firestore operate
func (b *TTCBrowserDB) FirestoreUpsert(collection string, id string, data map[string]interface{}) error {
	_, err := b.fireClient.Collection(collection).Doc(id).Set(b.fireContext, data, firestore.MergeAll)
	b.PrepareQuery()
	// [END fs_update_create_if_missing]
	return err
}


type City struct {
	Name       string   `firestore:"name,omitempty"`
	State      string   `firestore:"state,omitempty"`
	Country    string   `firestore:"country,omitempty"`
	Capital    bool     `firestore:"capital,omitempty"`
	Population int64    `firestore:"population,omitempty"`
	Regions    []string `firestore:"regions,omitempty"`
}

func (b *TTCBrowserDB) PrepareQuery() error {

	// [START fs_query_create_examples]
	cities := []struct {
		id string
		c  City
	}{
		{
			id: "SF",
			c: City{Name: "San Francisco", State: "CA", Country: "USA",
				Capital: false, Population: 860000,
				Regions: []string{"west_coast", "norcal"}},
		},
		{
			id: "LA",
			c: City{Name: "Los Angeles", State: "CA", Country: "USA",
				Capital: false, Population: 3900000,
				Regions: []string{"west_coast", "socal"}},
		},
	}
	for _, c := range cities {
		if _, err :=  b.fireClient.Collection("cities").Doc(c.id).Set(b.fireContext, c.c); err != nil {
			return err
		}
	}
	// [END fs_query_create_examples]
	return nil
}