package permission

import "context"

type Service interface {
	IsManager(ctx context.Context, userID string) (bool, error)
}
