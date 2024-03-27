package identity

type IdentityCreatedEvent struct {
	ID string
}

func NewIdentityCreatedEvent(ID string) IdentityCreatedEvent {
	return IdentityCreatedEvent{
		ID: ID,
	}
}

func (e IdentityCreatedEvent) EventName() string {
	return "IdentityCreated"
}

func (e IdentityCreatedEvent) GetID() string {
	return e.ID
}
