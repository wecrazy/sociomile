package events

import (
	"context"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

var (
	amqpDial        = amqp.Dial
	openChannel     = func(connection *amqp.Connection) (*amqp.Channel, error) { return connection.Channel() }
	declareExchange = func(channel *amqp.Channel, exchange string) error {
		return channel.ExchangeDeclare(exchange, "topic", true, false, false, false, nil)
	}
	publishWithContext = func(channel *amqp.Channel, ctx context.Context, exchange string, routingKey string, publishing amqp.Publishing) error {
		return channel.PublishWithContext(ctx, exchange, routingKey, false, false, publishing)
	}
	closeChannel = func(channel *amqp.Channel) error {
		if channel == nil {
			return nil
		}

		return channel.Close()
	}
	closeConnection = func(connection *amqp.Connection) error {
		if connection == nil {
			return nil
		}

		return connection.Close()
	}
)

// ExchangeName is the RabbitMQ exchange used for Sociomile domain events.
const ExchangeName = "sociomile.events"

// Publisher publishes domain events to RabbitMQ.
type Publisher struct {
	connection *amqp.Connection
	channel    *amqp.Channel
	exchange   string
}

// NewPublisher creates a RabbitMQ publisher and declares the event exchange.
func NewPublisher(url string) (*Publisher, error) {
	connection, err := amqpDial(url)
	if err != nil {
		return nil, err
	}

	channel, err := openChannel(connection)
	if err != nil {
		_ = closeConnection(connection)
		return nil, err
	}

	if err := declareExchange(channel, ExchangeName); err != nil {
		_ = closeChannel(channel)
		_ = closeConnection(connection)
		return nil, err
	}

	return &Publisher{
		connection: connection,
		channel:    channel,
		exchange:   ExchangeName,
	}, nil
}

// Publish sends a message to the configured exchange using the routing key.
func (p *Publisher) Publish(ctx context.Context, routingKey string, payload []byte) error {
	if p == nil {
		return nil
	}

	return publishWithContext(p.channel, ctx, p.exchange, routingKey, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now(),
		Body:         payload,
	})
}

// Close closes the AMQP channel and connection.
func (p *Publisher) Close() error {
	if p == nil {
		return nil
	}

	if err := closeChannel(p.channel); err != nil {
		_ = closeConnection(p.connection)
		return err
	}

	return closeConnection(p.connection)
}
