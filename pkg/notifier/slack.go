package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	batchv1 "k8s.io/api/batch/v1"
)

// Compile-time check that SlackNotifier implements Notifier
var _ Notifier = (*SlackNotifier)(nil)

type SlackNotifier struct {
	webhookURL string
	httpClient *http.Client
}

type slackMessage struct {
	Text   string       `json:"text"`
	Blocks []slackBlock `json:"blocks,omitempty"`
}

type slackBlock struct {
	Type string         `json:"type"`
	Text *slackTextObj  `json:"text,omitempty"`
}

type slackTextObj struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func NewSlackNotifier(webhookURL string) *SlackNotifier {
	return &SlackNotifier{
		webhookURL: webhookURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *SlackNotifier) NotifyJobCompleted(ctx context.Context, job *batchv1.Job, succeeded bool) error {
	var emoji, status, title string
	var completionTime time.Time

	if succeeded {
		emoji = ":tada:"
		status = "Succeeded"
		title = "Job Completed Successfully"
		if job.Status.CompletionTime != nil {
			completionTime = job.Status.CompletionTime.Time
		}
	} else {
		emoji = ":x:"
		status = "Failed"
		title = "Job Failed"
		completionTime = time.Now()
	}

	message := fmt.Sprintf(`%s *%s*
━━━━━━━━━━━━━━━━━━━━━━━━━━
• *Name:* %s
• *Namespace:* %s
• *Status:* %s
• *Time:* %s`,
		emoji,
		title,
		job.Name,
		job.Namespace,
		status,
		completionTime.Format(time.RFC3339),
	)

	return s.send(ctx, message)
}

func (s *SlackNotifier) send(ctx context.Context, text string) error {
	msg := slackMessage{
		Blocks: []slackBlock{
			{
				Type: "section",
				Text: &slackTextObj{
					Type: "mrkdwn",
					Text: text,
				},
			},
		},
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal slack message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send slack notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack returned non-OK status: %d", resp.StatusCode)
	}

	return nil
}
