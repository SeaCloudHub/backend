package notificationhub

import "github.com/SeaCloudHub/backend/domain/notification"

type NotificationRequest struct {
	From          string                      `json:"from"`
	Notifications []notification.Notification `json:"notifications"`
}
