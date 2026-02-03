package notify

import (
	"fmt"
	"log/slog"
)

// Notifier defines the interface for sending notifications
type Notifier interface {
	SendILLStatusUpdate(toEmail string, title string, status string, comment string) error
}

// LogNotifier is a stub implementation that logs notifications to stdout
type LogNotifier struct{}

func NewLogNotifier() *LogNotifier {
	return &LogNotifier{}
}

func (n *LogNotifier) SendILLStatusUpdate(toEmail string, title string, status string, comment string) error {
	// In a real implementation, this would use smtp.SendMail or an API like SendGrid
	slog.Info("Sending Notification",
		"type", "email",
		"to", toEmail,
		"subject", fmt.Sprintf("ILL Request Update: %s", title),
		"body", fmt.Sprintf("Your request for '%s' has been %s. Note: %s", title, status, comment),
	)
	return nil
}
