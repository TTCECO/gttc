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
	"google.golang.org/api/iterator"
)

// Firestore operate
func (b *TTCBrowserDB) FirestoreUpsert(collection string, id string, data map[string]interface{}) error {
	_, err := b.fireClient.Collection(collection).Doc(id).Set(b.fireContext, data)
	return err
}

//
func (b *TTCBrowserDB) FirestoreQueryById(collection string, id string) (map[string]interface{}, error) {
	query, err := b.fireClient.Collection(collection).Doc(id).Get(b.fireContext)
	if err != nil {
		return nil, err
	}
	return query.Data(), nil
}

//
func (b *TTCBrowserDB) FirestoreQuery(collection string, condition map[string]interface{}) ([]map[string]interface{}, error) {
	var result []map[string]interface{}
	query := b.fireClient.Collection(collection).Query
	//var query Query
	for k, v := range condition {
		query = query.Where(k, "==", v)
	}
	query = query.Limit(10)
	iter := query.Documents(b.fireContext)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		result = append(result, doc.Data())
	}
	return result, nil
}
