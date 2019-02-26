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
	query := b.fireClient.Collection(collection).Limit(10)
	for k, v := range condition {
		query = query.Where(k, "==", v)
	}
	res, err := query.Documents(b.fireContext).GetAll()
	if err != nil {
		return nil, err
	}
	for _, item := range res {
		result = append(result, item.Data())
	}

	return result, nil
}
