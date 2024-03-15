package book

import "context"

type Storage interface {
	Save(ctx context.Context, book *Book) error
	FindByISBN(ctx context.Context, isbn string) (*Book, error)
}

type Book struct {
	ISBN string
	Name string
}

func NewBook(isbn string, name string) Book {
	return Book{ISBN: isbn, Name: name}
}
