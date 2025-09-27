package audit

import (
	"context"
	"encoding/json"
	"time"

	"github.com/corray333/backend-labs/order/internal/dal/rabbitmq"
	"github.com/corray333/backend-labs/order/internal/service/models/order"
	"github.com/streadway/amqp"
	"golang.org/x/sync/errgroup"
)

type AuditRabbitMQRepository struct {
	client *rabbitmq.Client
	queue  amqp.Queue
}

func NewAuditRabbitMQRepository(client *rabbitmq.Client) *AuditRabbitMQRepository {
	queue, err := client.DeclareQueue(rabbitmq.DeclareQueueConfig{
		Name:       "oms.iorderrepo.created",
		Durable:    false,
		Exclusive:  false,
		AutoDelete: false,
	})
	if err != nil {
		panic(err)
	}

	return &AuditRabbitMQRepository{
		client: client,
		queue:  queue,
	}
}

func (r *AuditRabbitMQRepository) LogBatchInsert(ctx context.Context, orders []order.Order) error {
	auditCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	g, ctx := errgroup.WithContext(auditCtx)
	g.SetLimit(3)

	for _, ord := range orders {
		g.Go(func() error {
			orderData, err := json.Marshal(ord)
			if err != nil {
				return err
			}

			return r.client.Channel().Publish(
				"",
				r.queue.Name,
				false,
				false,
				amqp.Publishing{
					ContentType: "application/json",
					Body:        orderData,
				},
			)
		})
	}

	return g.Wait()
}
