package internal

import "mime/multipart"

type CSVService interface {
	CsvToEntities(file *multipart.File, entity interface{}) error
	EntitiesToCsv(entities interface{}) ([]byte, error)
}
