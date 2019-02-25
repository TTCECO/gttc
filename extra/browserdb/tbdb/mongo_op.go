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
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

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
