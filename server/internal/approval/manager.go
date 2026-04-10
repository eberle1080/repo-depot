// Package approval implements human-in-the-loop approval for gated operations.
// It publishes question signals to signal-supervisor via RabbitMQ and blocks
// until the user approves, denies, or the request times out.
package approval

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/amp-labs/amp-common/logger"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	signalsQueue    = "cassandra.signals"
	responsesQueue  = "cassandra.responses"
	approvalTimeout = 5 * time.Minute
)

// Manager publishes approval requests to signal-supervisor and waits for responses.
type Manager struct {
	conn    *amqp.Connection
	pubCh   *amqp.Channel
	pending sync.Map // id → chan string
}

// signal matches the Signal struct expected by signal-supervisor.
type signal struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"`
	Title    string   `json:"title,omitempty"`
	Body     string   `json:"body"`
	Choices  []string `json:"choices,omitempty"`
	Priority string   `json:"priority,omitempty"`
}

type signalResponse struct {
	ID     string `json:"id"`
	Answer string `json:"answer"`
}

// Connect establishes a connection to RabbitMQ and starts the response consumer.
// Returns nil if url is empty (approval disabled).
func Connect(ctx context.Context, url string) (*Manager, error) {
	if url == "" {
		return nil, nil
	}

	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("connect to RabbitMQ: %w", err)
	}

	pubCh, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("open publish channel: %w", err)
	}

	m := &Manager{conn: conn, pubCh: pubCh}

	if err := m.startConsumer(ctx); err != nil {
		pubCh.Close()
		conn.Close()
		return nil, err
	}

	logger.Get(ctx).Info("approval manager connected to RabbitMQ")

	return m, nil
}

// Close shuts down the RabbitMQ connection.
func (m *Manager) Close() {
	m.pubCh.Close()
	m.conn.Close()
}

func (m *Manager) startConsumer(ctx context.Context) error {
	ch, err := m.conn.Channel()
	if err != nil {
		return fmt.Errorf("open consume channel: %w", err)
	}

	deliveries, err := ch.Consume(
		responsesQueue,
		"repo-depot-approval", // consumer tag
		true,                  // auto-ack
		false,                 // exclusive
		false,                 // no-local
		false,                 // no-wait
		nil,
	)
	if err != nil {
		ch.Close()
		return fmt.Errorf("consume %s: %w", responsesQueue, err)
	}

	go func() {
		defer ch.Close()

		for {
			select {
			case <-ctx.Done():
				return
			case d, ok := <-deliveries:
				if !ok {
					return
				}

				var resp signalResponse
				if err := json.Unmarshal(d.Body, &resp); err != nil {
					logger.Get(ctx).Warn("approval: unparseable response", "error", err)
					continue
				}

				if pending, ok := m.pending.Load(resp.ID); ok {
					pending.(chan string) <- resp.Answer
				} else {
					logger.Get(ctx).Debug("approval: no pending request for response", "id", resp.ID)
				}
			}
		}
	}()

	return nil
}

// Request sends an approval question to signal-supervisor and blocks until the
// user responds or the request times out (5 minutes → auto-deny).
//
// title appears as the question heading in the menu bar popover.
// body provides supporting context (PR URL, branch names, etc.)
func (m *Manager) Request(ctx context.Context, title, body string) error {
	id, err := newID()
	if err != nil {
		return fmt.Errorf("generate request id: %w", err)
	}

	ch := make(chan string, 1)
	m.pending.Store(id, ch)

	defer m.pending.Delete(id)

	sig := signal{
		ID:       id,
		Type:     "question",
		Title:    title,
		Body:     body,
		Choices:  []string{"Approve", "Deny"},
		Priority: "high",
	}

	b, err := json.Marshal(sig)
	if err != nil {
		return fmt.Errorf("marshal signal: %w", err)
	}

	if err := m.pubCh.PublishWithContext(ctx,
		"", // default exchange
		signalsQueue,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        b,
		},
	); err != nil {
		return fmt.Errorf("publish approval request: %w", err)
	}

	select {
	case answer := <-ch:
		if answer != "Approve" {
			return fmt.Errorf("denied")
		}

		return nil
	case <-time.After(approvalTimeout):
		return fmt.Errorf("timed out after 5 minutes — auto-denied")
	case <-ctx.Done():
		return ctx.Err()
	}
}

func newID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}
