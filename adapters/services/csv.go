package services

import (
	"encoding/csv"
	"fmt"
	"io"
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

func (c *csvService) CsvToEntities(file multipart.File,
	entityMapper func(record []string) interface{}) ([]interface{}, error) {
	csvReader := csv.NewReader(file)

	// Skip header
	_, err := csvReader.Read()
	if err != nil {
		return nil, err
	}

	var entityList []interface{}
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read CSV record: %v", err)
		}

		// Map CSV record to entity
		entity := entityMapper(record)
		entityList = append(entityList, entity)
	}

	return entityList, nil
}
