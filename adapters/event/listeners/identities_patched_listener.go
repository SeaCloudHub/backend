package listeners

import (
	"context"
	"github.com/SeaCloudHub/backend/domain"
	"github.com/SeaCloudHub/backend/domain/file"
	"github.com/SeaCloudHub/backend/domain/identity"
)

type IdentitiesPatchedListener struct {
	fileService file.Service
}

func NewIdentitiesPatchedListener(fileService file.Service) IdentitiesPatchedListener {
	return IdentitiesPatchedListener{fileService: fileService}
}

func (l IdentitiesPatchedListener) EventHandler(event domain.BaseDomainEvent) error {
	identitiesPatchedEvent, ok := event.(identity.IdentitiesPatchedEvent)
	if !ok {
		return nil
	}

	for _, id := range identitiesPatchedEvent.IDs {
		dirpath := "/" + id + "/"
		if err := l.fileService.CreateDirectory(context.Background(), dirpath); err != nil {
			// If an error occurs while creating the directory, delete previously created directories
			for _, delID := range identitiesPatchedEvent.IDs[:len(identitiesPatchedEvent.IDs)] {
				_ = l.fileService.Delete(context.Background(), "/"+delID+"/")
			}
			return err
		}
	}

	return nil
}
