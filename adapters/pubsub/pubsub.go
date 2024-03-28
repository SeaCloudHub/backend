package pubsub

import "context"

type Topic string

type Pubsub interface {
	Publish(ctx context.Context)
}
