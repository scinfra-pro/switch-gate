package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Event represents a webhook event payload
type Event struct {
	Name      string                 `json:"event"`
	Timestamp time.Time              `json:"timestamp"`
	Source    string                 `json:"source"`
	Payload   map[string]interface{} `json:"payload"`
}

// Webhook sends events to a remote endpoint
type Webhook struct {
	url    string
	secret string
	source string
	client *http.Client
}

// New creates a new Webhook client
func New(url, secret, source string) *Webhook {
	return &Webhook{
		url:    url,
		secret: secret,
		source: source,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// Send sends an event asynchronously with retry
func (w *Webhook) Send(eventName string, payload map[string]interface{}) {
	go w.sendWithRetry(eventName, payload)
}

// sendWithRetry attempts to send with exponential backoff
func (w *Webhook) sendWithRetry(eventName string, payload map[string]interface{}) {
	backoff := []time.Duration{0, 1 * time.Second, 3 * time.Second}

	var lastErr error
	for i, delay := range backoff {
		if delay > 0 {
			time.Sleep(delay)
		}

		if err := w.send(eventName, payload); err == nil {
			log.Printf("INFO: Webhook sent: %s", eventName)
			return
		} else {
			lastErr = err
			if i < len(backoff)-1 {
				log.Printf("WARN: Webhook attempt %d failed: %v, retrying...", i+1, err)
			}
		}
	}

	log.Printf("ERROR: Webhook failed after %d attempts: %s - %v", len(backoff), eventName, lastErr)
}

// send performs the actual HTTP request
func (w *Webhook) send(eventName string, payload map[string]interface{}) error {
	event := Event{
		Name:      eventName,
		Timestamp: time.Now().UTC(),
		Source:    w.source,
		Payload:   payload,
	}

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	req, err := http.NewRequest("POST", w.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Secret", w.secret)

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("server returned %d", resp.StatusCode)
	}

	return nil
}
