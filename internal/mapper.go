package internal

import (
	"github.com/SeaCloudHub/backend/adapters/httpserver/model"
	"github.com/SeaCloudHub/backend/domain/file"
	"github.com/SeaCloudHub/backend/domain/identity"
)

type Mapper interface {
	ToIdentity(request model.CreateIdentityRequest) identity.SimpleIdentity
	ToIdentities(request []model.CreateIdentityRequest) ([]identity.SimpleIdentity, error)
	FileWithParents(files []file.File, parents []file.SimpleFile) []file.File
}
