package internal

import (
	"mime/multipart"
)

type CSVService interface {
	CsvToEntities(file multipart.File,
		entityMapper func(record []string) interface{}) ([]interface{}, error)
}
