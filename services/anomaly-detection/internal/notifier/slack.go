package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type SlackNotifier struct {
	webhookURL string
	client     *http.Client
}

func NewSlack(webhookURL string) *SlackNotifier {
	return &SlackNotifier{
		webhookURL: webhookURL,
		client:     &http.Client{Timeout: 5 * time.Second},
	}
}

func (s *SlackNotifier) Notify(namespace, severity string, actual, expected, zscore float64) error {
	if s.webhookURL == "" {
		return nil // no webhook configured — skip silently
	}

	emoji := ":warning:"
	if severity == "critical" {
		emoji = ":rotating_light:"
	}

	msg := map[string]string{
		"text": fmt.Sprintf(
			"%s *FinOps Anomaly Detected* %s\n"+
				"*Namespace:* `%s`\n"+
				"*Severity:* %s\n"+
				"*Actual cost:* $%.4f/h\n"+
				"*Expected cost:* $%.4f/h\n"+
				"*Z-Score:* %.2f\n"+
				"*Time:* %s",
			emoji, emoji,
			namespace,
			severity,
			actual,
			expected,
			zscore,
			time.Now().Format("2006-01-02 15:04 UTC"),
		),
	}

	body, _ := json.Marshal(msg)
	resp, err := s.client.Post(s.webhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
