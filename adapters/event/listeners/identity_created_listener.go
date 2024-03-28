package listeners

import (
	"context"
	"github.com/SeaCloudHub/backend/domain"
	"github.com/SeaCloudHub/backend/domain/file"
	"github.com/SeaCloudHub/backend/domain/identity"
	"path"
)

type IdentityCreatedEventListener struct {
	fileService file.Service
}

func NewIdentityCreatedEventListener(fileService file.Service) IdentityCreatedEventListener {
	return IdentityCreatedEventListener{fileService: fileService}
}

func (l IdentityCreatedEventListener) EventHandler(event domain.BaseDomainEvent) error {
	identityCreatedEvent, ok := event.(identity.IdentityCreatedEvent)
	if !ok {
		return nil
	}

	dirpath := path.Join("/", identityCreatedEvent.ID) + "/"

	err := l.fileService.CreateDirectory(context.Background(), dirpath)
	if err != nil {
		return err
	}

	return nil
}
