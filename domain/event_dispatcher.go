package domain

type EventDispatcher interface {
	Dispatch(event BaseDomainEvent) error
	Register(eventName string, listener func(event BaseDomainEvent) error)
}
