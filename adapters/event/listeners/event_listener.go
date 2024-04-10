package listeners

import "github.com/SeaCloudHub/backend/domain"

type EventListener interface {
	EventHandler(event domain.BaseDomainEvent) error
}
