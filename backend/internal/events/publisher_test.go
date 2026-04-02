package events

import (
	"context"
	"errors"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
)

func TestPublisherPublishUsesInjectedPublisher(t *testing.T) {
	originalPublish := publishWithContext
	t.Cleanup(func() {
		publishWithContext = originalPublish
	})

	called := false
	publishWithContext = func(channel *amqp.Channel, _ context.Context, exchange string, routingKey string, publishing amqp.Publishing) error {
		called = true
		require.Nil(t, channel)
		require.Equal(t, ExchangeName, exchange)
		require.Equal(t, "ticket.created", routingKey)
		require.Equal(t, "application/json", publishing.ContentType)
		require.Equal(t, []byte(`{"ok":true}`), publishing.Body)
		return nil
	}

	publisher := &Publisher{exchange: ExchangeName}
	require.NoError(t, publisher.Publish(context.Background(), "ticket.created", []byte(`{"ok":true}`)))
	require.True(t, called)

	publishWithContext = func(*amqp.Channel, context.Context, string, string, amqp.Publishing) error {
		return errors.New("publish failed")
	}
	require.ErrorContains(t, publisher.Publish(context.Background(), "ticket.created", []byte(`{"ok":true}`)), "publish failed")
	require.NoError(t, (*Publisher)(nil).Publish(context.Background(), "ignored", nil))
}

func TestNewPublisherCoversDialChannelAndExchangePaths(t *testing.T) {
	originalDial := amqpDial
	originalOpenChannel := openChannel
	originalDeclareExchange := declareExchange
	originalCloseChannel := closeChannel
	originalCloseConnection := closeConnection
	t.Cleanup(func() {
		amqpDial = originalDial
		openChannel = originalOpenChannel
		declareExchange = originalDeclareExchange
		closeChannel = originalCloseChannel
		closeConnection = originalCloseConnection
	})

	amqpDial = func(url string) (*amqp.Connection, error) {
		require.Equal(t, "amqp://broker", url)
		return nil, errors.New("dial failed")
	}

	publisher, err := NewPublisher("amqp://broker")
	require.Nil(t, publisher)
	require.ErrorContains(t, err, "dial failed")

	connectionClosed := 0
	amqpDial = func(_ string) (*amqp.Connection, error) {
		return nil, nil
	}
	openChannel = func(_ *amqp.Connection) (*amqp.Channel, error) {
		return nil, errors.New("channel failed")
	}
	closeConnection = func(_ *amqp.Connection) error {
		connectionClosed++
		return nil
	}

	publisher, err = NewPublisher("amqp://broker")
	require.Nil(t, publisher)
	require.ErrorContains(t, err, "channel failed")
	require.Equal(t, 1, connectionClosed)

	channelClosed := 0
	connectionClosed = 0
	openChannel = func(_ *amqp.Connection) (*amqp.Channel, error) {
		return nil, nil
	}
	declareExchange = func(_ *amqp.Channel, exchange string) error {
		require.Equal(t, ExchangeName, exchange)
		return errors.New("exchange failed")
	}
	closeChannel = func(_ *amqp.Channel) error {
		channelClosed++
		return nil
	}

	publisher, err = NewPublisher("amqp://broker")
	require.Nil(t, publisher)
	require.ErrorContains(t, err, "exchange failed")
	require.Equal(t, 1, channelClosed)
	require.Equal(t, 1, connectionClosed)

	declareExchange = func(_ *amqp.Channel, exchange string) error {
		require.Equal(t, ExchangeName, exchange)
		return nil
	}

	publisher, err = NewPublisher("amqp://broker")
	require.NoError(t, err)
	require.NotNil(t, publisher)
	require.Equal(t, ExchangeName, publisher.exchange)
}

func TestPublisherCloseCoversSuccessAndFailure(t *testing.T) {
	originalCloseChannel := closeChannel
	originalCloseConnection := closeConnection
	t.Cleanup(func() {
		closeChannel = originalCloseChannel
		closeConnection = originalCloseConnection
	})

	channelClosed := false
	connectionClosed := false
	closeChannel = func(_ *amqp.Channel) error {
		channelClosed = true
		return nil
	}
	closeConnection = func(_ *amqp.Connection) error {
		connectionClosed = true
		return nil
	}

	publisher := &Publisher{}
	require.NoError(t, publisher.Close())
	require.True(t, channelClosed)
	require.True(t, connectionClosed)
	require.NoError(t, (*Publisher)(nil).Close())

	channelClosed = false
	connectionClosed = false
	closeChannel = func(_ *amqp.Channel) error {
		channelClosed = true
		return errors.New("channel close failed")
	}
	closeConnection = func(_ *amqp.Connection) error {
		connectionClosed = true
		return nil
	}

	err := publisher.Close()
	require.ErrorContains(t, err, "channel close failed")
	require.True(t, channelClosed)
	require.True(t, connectionClosed)

	channelClosed = false
	connectionClosed = false
	closeChannel = func(_ *amqp.Channel) error {
		channelClosed = true
		return nil
	}
	closeConnection = func(_ *amqp.Connection) error {
		connectionClosed = true
		return errors.New("connection close failed")
	}

	err = publisher.Close()
	require.ErrorContains(t, err, "connection close failed")
	require.True(t, channelClosed)
	require.True(t, connectionClosed)
}
