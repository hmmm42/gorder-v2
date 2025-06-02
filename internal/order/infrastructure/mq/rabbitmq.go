package mq

import (
	"context"

	"github.com/hmmm42/gorder-v2/common/broker"
	domain "github.com/hmmm42/gorder-v2/order/domain/order"
	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQEventPublisher struct {
	Channel *amqp.Channel
}

func NewRabbitMQEventPublisher(channel *amqp.Channel) *RabbitMQEventPublisher {
	return &RabbitMQEventPublisher{Channel: channel}
}

// Publish 传入的Exchange为空, 直接路由到Queue对应的队列, 即 EventOrderCreated
func (r RabbitMQEventPublisher) Publish(ctx context.Context, event domain.DomainEvent) error {
	return broker.PublishEvent(ctx, broker.PublishEventReq{
		Channel:  r.Channel,
		Routing:  broker.Direct,
		Queue:    event.Dest,
		Exchange: "",
		Body:     event.Data,
	})
}

func (r RabbitMQEventPublisher) Broadcast(ctx context.Context, event domain.DomainEvent) error {
	return broker.PublishEvent(ctx, broker.PublishEventReq{
		Channel:  r.Channel,
		Routing:  broker.FanOut,
		Queue:    event.Dest,
		Exchange: "",
		Body:     event.Data,
	})
}
