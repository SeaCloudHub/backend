package notificationhub

import (
	"context"
	"fmt"
	"github.com/SeaCloudHub/backend/domain/notification"
	"github.com/SeaCloudHub/backend/pkg/config"
	"github.com/go-resty/resty/v2"
	"net/url"
)

type NotificationHub struct {
	host   *url.URL
	client *resty.Client
}

func NewNotificationHub(cfg *config.Config) (*NotificationHub, error) {
	u, err := url.Parse(cfg.NotificationHub.Endpoint)
	if err != nil {
		return nil, err
	}

	return &NotificationHub{
		host:   u,
		client: resty.New().SetBaseURL(u.String()),
	}, nil
}

func (n *NotificationHub) pushNotification(ctx context.Context, notificationReq NotificationRequest, token string) error {
	resp, err := n.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", token)).
		SetBody(notificationReq).
		Post("/api/internal/notifications")

	if err != nil {
		return err
	}

	if resp.StatusCode() != 200 {
		return fmt.Errorf("failed to push notification: %s", resp.Status())
	}

	return nil
}

func (n *NotificationHub) SendNotification(ctx context.Context,
	notifications []notification.Notification, userId, token string) error {
	notificationReq := NotificationRequest{
		Notifications: notifications,
		From:          userId,
	}
	return n.pushNotification(ctx, notificationReq, token)
}
