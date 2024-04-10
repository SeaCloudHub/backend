package services

import (
	"github.com/gocarina/gocsv"
	"mime/multipart"
	"sync"
)

var (
	csvServiceInstance *csvService
	once               sync.Once
)

type csvService struct{}

func NewCSVService() *csvService {
	once.Do(func() {
		csvServiceInstance = &csvService{}
	})
	return csvServiceInstance
}

func (c *csvService) CsvToEntities(file *multipart.File, entity interface{}) error {
	return gocsv.UnmarshalMultipartFile(file, entity)
}

func (c *csvService) EntitiesToCsv(entities interface{}) ([]byte, error) {
	return gocsv.MarshalBytes(entities)
}
