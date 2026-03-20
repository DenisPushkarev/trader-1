package nats

import (
	"context"
	"fmt"

	natspkg "github.com/nats-io/nats.go"
)

// ConsumeMessages subscribes to a JetStream subject via push consumer and calls handler for each message.
// It blocks until ctx is cancelled.
func (c *Client) ConsumeMessages(ctx context.Context, cfg ConsumerConfig, handler func(data []byte) error) error {
	sub, err := c.js.SubscribeSync(cfg.Subject,
		natspkg.Durable(cfg.Durable),
		natspkg.ManualAck(),
		natspkg.BindStream(cfg.Stream),
	)
	if err != nil {
		return fmt.Errorf("subscribe %s/%s: %w", cfg.Stream, cfg.Durable, err)
	}
	defer sub.Unsubscribe() //nolint:errcheck

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		msg, err := sub.NextMsgWithContext(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			c.logger.Warn().Err(err).Msg("next message error")
			continue
		}

		if err := handler(msg.Data); err != nil {
			c.logger.Error().Err(err).Msg("handler error — nacking message")
			msg.Nak() //nolint:errcheck
			continue
		}
		msg.Ack() //nolint:errcheck
	}
}
