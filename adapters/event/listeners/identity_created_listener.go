package listeners

import (
	"context"
	"github.com/SeaCloudHub/backend/domain"
	"github.com/SeaCloudHub/backend/domain/file"
	"github.com/SeaCloudHub/backend/domain/identity"
	"github.com/SeaCloudHub/backend/pkg/util"
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
	
	err := l.fileService.CreateDirectory(context.Background(),
		util.GetIdentityDirPath(identityCreatedEvent.ID))
	if err != nil {
		return err
	}

	return nil
}
