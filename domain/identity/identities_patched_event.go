package identity

type IdentitiesPatchedEvent struct {
	IDs []string
}

func NewIdentitiesPatchedEvent(IDs []string) IdentitiesPatchedEvent {
	return IdentitiesPatchedEvent{
		IDs: IDs,
	}
}

func (e IdentitiesPatchedEvent) EventName() string {
	return "IdentitiesPatched"
}
