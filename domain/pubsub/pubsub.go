package pubsub

import "context"

type Message struct {
	Channel string
	Payload string
}

type PubSub interface {
	ReceiveMessage(ctx context.Context) (Message, error)
	Close() error
}

type Service interface {
	Publish(ctx context.Context, channel string, message interface{}) error
	Subscribe(ctx context.Context, channel string) PubSub
}
