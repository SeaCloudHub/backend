package inmemstore

import (
	"github.com/SeaCloudHub/backend/domain/book"

	"github.com/dgraph-io/badger/v4"
)

type BookStore struct {
	db *badger.DB
}

func NewBookStore(db *badger.DB) *BookStore {
	return &BookStore{db}
}

func (b *BookStore) Save(data *book.Book) error {
	key := []byte(data.ISBN)
	value := []byte(data.Name)
	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, value)
	})
}

func (b *BookStore) FindByISBN(isbn string) (*book.Book, error) {
	result := book.Book{}
	err := b.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(isbn))
		if err != nil {
			return err
		}

		_ = item.Value(func(val []byte) error {
			result.ISBN = string(item.Key())
			result.Name = string(val)
			return nil
		})

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &result, nil
}
