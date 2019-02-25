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
	"context"
	"log"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
)

// Tests query function and speed in firestore.google
func TestQuery(t *testing.T) {

	ctx := context.Background()

	projectID := os.Getenv("GCLOUD_PROJECT")
	if projectID == "" {
		log.Fatalf("Set Firebase project ID via GCLOUD_PROJECT env variable.")
	}

	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Cannot create client: %v", err)
	}

	defer client.Close()

	// check data from snapshot, tally and txs
	headerHash := "0x00032be859906732178a6bfd67b626ecd5120bc25fb9590e407bce092ee2e502"
	queryCheck(client, ctx, "snapshot", headerHash)

	number := "100800"
	queryCheck(client, ctx, "tally", number)

	txHash := "0x000278eee857fe590842b1a1bd088a159e15015b37c181be753db2a382467785"
	queryCheck(client, ctx, "txs", txHash)

	// query data from collection by different condition
	queryByCondition(client, ctx, "snapshot", "number", ">", 105813)

	queryByCondition(client, ctx, "txs", "Number", ">", 25418)

}

func queryCheck(client *firestore.Client, ctx context.Context, collectionName string, key string) {
	startTime := time.Now().UnixNano()
	if _, err := client.Collection(collectionName).Doc(key).Get(ctx); err != nil {
		log.Fatalf("Cannot query %s by key %s ", collectionName, key)
	} else {
		log.Printf("Query data from %s by key %s during %f", collectionName, key, float64(time.Now().UnixNano()-startTime)/1e+9)
		//log.Println(query.Data())
	}
}

func queryByCondition(client *firestore.Client, ctx context.Context, collectionName string, key string, op string, value interface{}) {
	startTime := time.Now().UnixNano()
	query := client.Collection(collectionName).Where(key, op, value)
	log.Printf("Query data from %s by key %s during %f", collectionName, key, float64(time.Now().UnixNano()-startTime)/1e+9)

	if doc, err := query.Documents(ctx).GetAll(); err != nil {
		log.Fatalf("Cannot query %s by key %s ,err = %s", collectionName, key, err)
	} else {
		log.Printf("Result count from %s by %s %s %s is %d", collectionName, key, op, value.(string), len(doc))
	}

}
