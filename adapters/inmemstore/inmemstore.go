package inmemstore

import (
	"github.com/dgraph-io/badger/v4"
)

func NewConnection() (*badger.DB, error) {
	return badger.Open(badger.DefaultOptions("").WithInMemory(true))
}
