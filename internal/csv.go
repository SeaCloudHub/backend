package internal

import "mime/multipart"

type CSVService interface {
	CsvToEntities(file *multipart.File, entity interface{}) error
}
