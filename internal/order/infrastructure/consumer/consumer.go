package consumer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hmmm42/gorder-v2/common/broker"
	"github.com/hmmm42/gorder-v2/order/app"
	"github.com/hmmm42/gorder-v2/order/app/command"
	domain "github.com/hmmm42/gorder-v2/order/domain/order"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
)

type Consumer struct {
	app app.Application
}

func NewConsumer(app app.Application) *Consumer {
	return &Consumer{
		app: app,
	}
}

func (c *Consumer) Listen(ch *amqp.Channel) {
	q, err := ch.QueueDeclare(broker.EventOrderPaid, true, false, true, false, nil)
	if err != nil {
		logrus.Fatal(err)
	}

	err = ch.QueueBind(q.Name, "", broker.EventOrderPaid, false, nil)
	if err != nil {
		logrus.Fatal(err)
	}

	msgs, err := ch.Consume(q.Name, "", false, false, false, false, nil)
	if err != nil {
		logrus.Fatal(err)
	}

	var forever chan struct{}
	go func() {
		for msg := range msgs {
			c.handleMessage(ch, msg, q)
		}
	}()
	//goland:noinspection GoDfaNilDereference
	<-forever
}

func (c *Consumer) handleMessage(ch *amqp.Channel, msg amqp.Delivery, q amqp.Queue) {
	ctx := broker.ExtractRabbitMQHeaders(context.Background(), msg.Headers)
	t := otel.Tracer("rabbitmq")
	_, span := t.Start(ctx, fmt.Sprintf("rabbitmq.%s.consume", q.Name))
	defer span.End()

	var err error
	defer func() {
		if err != nil {
			_ = msg.Nack(false, false)
		} else {
			_ = msg.Ack(false)
		}
	}()

	o := &domain.Order{}
	if err = json.Unmarshal(msg.Body, o); err != nil {
		logrus.Infof("error unmarshal msg.body into domain.order, err=%v", err)
		return
	}

	_, err = c.app.Commands.UpdateOrder.Handle(ctx, command.UpdateOrder{
		Order: o,
		UpdateFn: func(context.Context, *domain.Order) (*domain.Order, error) {
			if err := o.IsPaid(); err != nil {
				return nil, err
			}
			return o, nil
		},
	})
	if err != nil {
		logrus.Infof("error updating order, orderID = %s, err = %v", o.ID, err)
		if err = broker.HandleRetry(ctx, ch, &msg); err != nil {
			logrus.Warnf("retry_error, error handling retry, messageID=%s, err=%v", msg.MessageId, err)
		}
		return
	}

	span.AddEvent("order.updated")
	logrus.Info("order consume paid event success!")
}
