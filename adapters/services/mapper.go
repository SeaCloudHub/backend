package services

import (
	"sync"

	"github.com/SeaCloudHub/backend/adapters/httpserver/model"
	"github.com/SeaCloudHub/backend/domain/file"
	"github.com/SeaCloudHub/backend/domain/identity"
)

var (
	mapperInstance *mapper
	onceMapper     sync.Once
)

type mapper struct{}

func NewMapperService() *mapper {
	onceMapper.Do(func() {
		mapperInstance = &mapper{}
	})
	return mapperInstance
}

func (s *mapper) ToIdentity(request model.CreateIdentityRequest) identity.SimpleIdentity {
	return identity.SimpleIdentity{
		Email:    request.Email,
		Password: request.Password,
	}
}

func (s *mapper) ToIdentities(request []model.CreateIdentityRequest) ([]identity.SimpleIdentity, error) {
	var identities []identity.SimpleIdentity
	for _, r := range request {
		if err := r.Validate(); err != nil {
			return nil, err
		}
		identities = append(identities, s.ToIdentity(r))
	}
	return identities, nil
}

func (s *mapper) FileWithParents(files []file.File, parents []file.SimpleFile) []file.File {
	var parentMap = make(map[string]file.SimpleFile)
	for _, parent := range parents {
		parentMap[parent.FullPath()] = parent
	}

	var newFiles []file.File

	for _, f := range files {
		if parent, ok := parentMap[f.Path]; ok {
			f.Parent = &parent

			newFiles = append(newFiles, f)
		}
	}

	return newFiles
}
