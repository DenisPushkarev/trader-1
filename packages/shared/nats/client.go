package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
)

// Client wraps a NATS connection with JetStream support.
type Client struct {
	conn   *nats.Conn
	js     nats.JetStreamContext
	logger zerolog.Logger
}

// NewClient creates a new NATS client and connects to the given URL.
func NewClient(url string, logger zerolog.Logger) (*Client, error) {
	conn, err := nats.Connect(url,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(10),
		nats.ReconnectWait(2*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}

	js, err := conn.JetStream()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("nats jetstream: %w", err)
	}

	return &Client{conn: conn, js: js, logger: logger}, nil
}

// Publish publishes a message to a NATS subject.
func (c *Client) Publish(_ context.Context, subject string, data []byte) error {
	if err := c.conn.Publish(subject, data); err != nil {
		return fmt.Errorf("publish to %s: %w", subject, err)
	}
	return nil
}

// PublishJSON serializes v to JSON and publishes to subject.
func (c *Client) PublishJSON(_ context.Context, subject string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if err := c.conn.Publish(subject, data); err != nil {
		return fmt.Errorf("publish to %s: %w", subject, err)
	}
	return nil
}

// Subscribe creates a plain NATS subscription.
func (c *Client) Subscribe(subject string, handler func(msg *nats.Msg)) (*nats.Subscription, error) {
	sub, err := c.conn.Subscribe(subject, handler)
	if err != nil {
		return nil, fmt.Errorf("subscribe to %s: %w", subject, err)
	}
	return sub, nil
}

// JetStream returns the JetStream context.
func (c *Client) JetStream() nats.JetStreamContext {
	return c.js
}

// EnsureStream creates a JetStream stream if it does not already exist.
func (c *Client) EnsureStream(cfg *nats.StreamConfig) error {
	_, err := c.js.StreamInfo(cfg.Name)
	if err == nats.ErrStreamNotFound {
		_, err = c.js.AddStream(cfg)
		if err != nil {
			return fmt.Errorf("add stream %s: %w", cfg.Name, err)
		}
		c.logger.Info().Str("stream", cfg.Name).Msg("created JetStream stream")
		return nil
	}
	if err != nil {
		return fmt.Errorf("stream info %s: %w", cfg.Name, err)
	}
	return nil
}

// ConsumerConfig holds durable consumer parameters.
type ConsumerConfig struct {
	Stream     string
	Durable    string
	Subject    string
	MaxDeliver int
	AckWait    time.Duration
}

// EnsureConsumer creates a durable push consumer if it does not already exist.
func (c *Client) EnsureConsumer(cfg ConsumerConfig) error {
	ackWait := cfg.AckWait
	if ackWait == 0 {
		ackWait = 30 * time.Second
	}
	maxDeliver := cfg.MaxDeliver
	if maxDeliver == 0 {
		maxDeliver = 5
	}

	_, err := c.js.ConsumerInfo(cfg.Stream, cfg.Durable)
	if err == nats.ErrConsumerNotFound {
		_, err = c.js.AddConsumer(cfg.Stream, &nats.ConsumerConfig{
			Durable:        cfg.Durable,
			FilterSubject:  cfg.Subject,
			AckPolicy:      nats.AckExplicitPolicy,
			MaxDeliver:     maxDeliver,
			AckWait:        ackWait,
			DeliverPolicy:  nats.DeliverAllPolicy,
			DeliverSubject: nats.NewInbox(),
		})
		if err != nil {
			return fmt.Errorf("add consumer %s/%s: %w", cfg.Stream, cfg.Durable, err)
		}
		c.logger.Info().Str("stream", cfg.Stream).Str("consumer", cfg.Durable).Msg("created durable consumer")
		return nil
	}
	return err
}

// Close drains and closes the NATS connection.
func (c *Client) Close() {
	if err := c.conn.Drain(); err != nil {
		c.logger.Warn().Err(err).Msg("nats drain error")
	}
	c.conn.Close()
}

// Conn returns the underlying nats.Conn.
func (c *Client) Conn() *nats.Conn {
	return c.conn
}
