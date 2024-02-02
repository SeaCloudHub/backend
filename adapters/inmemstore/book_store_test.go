package inmemstore_test

import (
	"testing"

	"github.com/SeaCloudHub/backend/adapters/inmemstore"
	"github.com/SeaCloudHub/backend/domain/book"

	"github.com/dgraph-io/badger/v4"
	"github.com/stretchr/testify/assert"
)

func TestBookStore(t *testing.T) {
	db, err := inmemstore.NewConnection()
	assert.NoError(t, err)

	store := inmemstore.NewBookStore(db)

	t.Run("save book", func(t *testing.T) {
		want := book.NewBook("9781680500745", "Clojure Applied")

		err := store.Save(&want)
		assert.NoError(t, err)

		assertBookStored(t, db, want)
	})

	t.Run("get book", func(t *testing.T) {
		want := book.NewBook("9781680500745", "Clojure Applied")

		err := store.Save(&want)
		assert.NoError(t, err)

		got, err := store.FindByISBN(want.ISBN)
		assert.NoError(t, err)
		assertFoundBook(t, got, &want)
	})
}

func assertFoundBook(t testing.TB, got, want *book.Book) {
	t.Helper()

	assert.NotNil(t, got)
	assert.Equal(t, want.ISBN, got.ISBN)
	assert.Equal(t, want.Name, got.Name)
}

func assertBookStored(t testing.TB, db *badger.DB, data book.Book) {
	t.Helper()

	_ = db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(data.ISBN))
		assert.NoError(t, err)
		assert.NotNil(t, item)
		return nil
	})
}
