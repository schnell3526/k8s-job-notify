package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

// NotificationLevel determines which job events trigger notifications.
type NotificationLevel string

const (
	// NotificationLevelAll sends notifications for both succeeded and failed jobs.
	NotificationLevelAll NotificationLevel = "all"
	// NotificationLevelFailed sends notifications only for failed jobs.
	NotificationLevelFailed NotificationLevel = "failed"
)

// ShouldNotifySuccess returns true if success notifications should be sent.
func (l NotificationLevel) ShouldNotifySuccess() bool {
	return l == NotificationLevelAll
}

// ShouldNotifyFailure returns true if failure notifications should be sent.
func (l NotificationLevel) ShouldNotifyFailure() bool {
	return true // Always notify on failure
}

type Config struct {
	SlackWebhookURL   string
	Namespace         string
	InCluster         bool
	ResyncPeriod      time.Duration
	NotificationLevel NotificationLevel
}

func Load() (*Config, error) {
	webhookURL := os.Getenv("SLACK_WEBHOOK_URL")
	if webhookURL == "" {
		return nil, errors.New("SLACK_WEBHOOK_URL is required")
	}

	inCluster := true
	if v := os.Getenv("IN_CLUSTER"); v != "" {
		parsed, err := strconv.ParseBool(v)
		if err != nil {
			return nil, errors.New("IN_CLUSTER must be a boolean value")
		}
		inCluster = parsed
	}

	resyncPeriod := 30 * time.Second
	if v := os.Getenv("RESYNC_PERIOD"); v != "" {
		seconds, err := strconv.Atoi(v)
		if err != nil {
			return nil, errors.New("RESYNC_PERIOD must be an integer (seconds)")
		}
		resyncPeriod = time.Duration(seconds) * time.Second
	}

	notificationLevel, err := parseNotificationLevel(os.Getenv("NOTIFICATION_LEVEL"))
	if err != nil {
		return nil, err
	}

	return &Config{
		SlackWebhookURL:   webhookURL,
		Namespace:         os.Getenv("NAMESPACE"),
		InCluster:         inCluster,
		ResyncPeriod:      resyncPeriod,
		NotificationLevel: notificationLevel,
	}, nil
}

func parseNotificationLevel(value string) (NotificationLevel, error) {
	if value == "" {
		return NotificationLevelAll, nil // default
	}

	switch NotificationLevel(value) {
	case NotificationLevelAll, NotificationLevelFailed:
		return NotificationLevel(value), nil
	default:
		return "", fmt.Errorf("NOTIFICATION_LEVEL must be 'all' or 'failed', got: %s", value)
	}
}
