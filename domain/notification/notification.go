package notification

import "context"

type Notification struct {
	UserID  string `json:"user_id"`
	Content string `json:"content"`
}

type Service interface {
	SendNotification(ctx context.Context, notifications []Notification,
		userId, token string) error
}
