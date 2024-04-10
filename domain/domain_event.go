package domain

type BaseDomainEvent interface {
	EventName() string
}
