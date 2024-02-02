package book

type Storage interface {
	Save(book *Book) error
	FindByISBN(isbn string) (*Book, error)
}

type Book struct {
	ISBN string
	Name string
}

func NewBook(isbn string, name string) Book {
	return Book{ISBN: isbn, Name: name}
}
