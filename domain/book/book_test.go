package book_test

import (
	"testing"

	"github.com/SeaCloudHub/backend/domain/book"
	"github.com/stretchr/testify/assert"
)

func TestNewBook(t *testing.T) {
	b := book.NewBook("9781804617007", "Microservices with Go")
	assert.Equal(t, b.ISBN, "9781804617007")
	assert.Equal(t, b.Name, "Microservices with Go")
}
