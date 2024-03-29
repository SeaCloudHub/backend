package services

import (
	"sync"

	"github.com/SeaCloudHub/backend/adapters/httpserver/model"
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
		Email:     request.Email,
		Password:  request.Password,
		FirstName: request.FirstName,
		LastName:  request.LastName,
		AvatarURL: request.AvatarURL,
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
