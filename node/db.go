package main

import (
	"log"

	"github.com/cockroachdb/pebble"
)

// InitializeDB initializes and returns a PebbleDB instance.
func InitializeDB(path string) *pebble.DB {
    db, err := pebble.Open(path, &pebble.Options{})
    if err != nil {
        log.Fatalf("Failed to open database: %v", err)
    }
    return db
}
