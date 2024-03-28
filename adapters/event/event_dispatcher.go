package event

import (
	"github.com/SeaCloudHub/backend/domain"
	"sync"
)

type eventDispatcher struct {
	listeners map[string][]func(event domain.BaseDomainEvent) error
	mutex     sync.Mutex
}

func NewEventDispatcher() *eventDispatcher {
	return &eventDispatcher{}
}

func (ed *eventDispatcher) Dispatch(event domain.BaseDomainEvent) error {
	ed.mutex.Lock()
	defer ed.mutex.Unlock()
	listeners, ok := ed.listeners[event.EventName()]
	if !ok {
		return nil
	}
	for _, listener := range listeners {
		if err := listener(event); err != nil {
			return err
		}
	}

	ed.listeners[event.EventName()] = nil

	return nil
}

func (ed *eventDispatcher) Register(eventName string, listener func(event domain.BaseDomainEvent) error) {
	ed.mutex.Lock()
	defer ed.mutex.Unlock()
	if ed.listeners == nil {
		ed.listeners = make(map[string][]func(event domain.BaseDomainEvent) error)
	}
	ed.listeners[eventName] = append(ed.listeners[eventName], listener)
}
