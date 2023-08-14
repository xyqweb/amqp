package amqp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
	"time"
)

var Producer = producer{}

type producer struct{}

// get rabbitmq pool
func (p *producer) getPool() (*rabbitmq, error) {
	if rabbitItem, err := rabbitmqPool.Get(); err != nil {
		return nil, err
	} else if rabbit, ok := rabbitItem.(*rabbitmq); ok {
		return rabbit, nil
	} else {
		return nil, errors.New("get rabbitmq pool fail")
	}
}

// Publish publish queue
func (p *producer) Publish(ctx context.Context, queue *Queue) (messageId string, err error) {
	var queueBody []byte
	if queueBody, err = json.Marshal(map[string]interface{}{
		"type": queue.Type,
		"data": queue.Data,
	}); err != nil {
		return
	}
	return p.publish(ctx, &queuePushItem{
		QueueName: queue.QueueName,
		MessageId: Util.RandomString(32),
		Body:      queueBody,
		Delay:     queue.Delay,
	})
}

// PublishByConsumer publish queue form customer fail
func (p *producer) PublishByConsumer(ctx context.Context, data *QueueData, body []byte) (messageId string, err error) {
	var attempt int32
	if headerAttempt, ok := data.Headers[attemptName]; ok {
		if attempt, ok = headerAttempt.(int32); !ok {
			attempt = 1
		} else if attempt < 1 {
			attempt = 1
		}
	}
	return p.publish(ctx, &queuePushItem{
		QueueName: data.QueueName,
		MessageId: data.MessageId,
		Body:      body,
		Attempt:   attempt,
		Delay:     60,
	})
}

// openChannel open rabbitmq connection channel
func (p *producer) openChannel(rabbit *rabbitmq) (channel *amqp.Channel, err error) {
	if conn := rabbit.OpenConn(); conn == nil {
		err = errors.New("open Rabbitmq connection fail")
		log.Printf("【Producer】open Rabbitmq connection fail")
		return
	} else {
		if channel, err = conn.Channel(); err != nil {
			log.Printf("【Producer】create channel err:%+v", err)
			return
		}
		if err = channel.ExchangeDeclare(
			Config.Exchange, // name
			Config.Kind,     // type
			true,            // durable
			false,           // auto-deleted
			false,           // internal
			false,           // noWait
			nil,             // arguments
		); err != nil {
			log.Printf("【Producer】 ExchangeDeclare：%s error：%+v", Config.Exchange, err)
			return
		}
	}
	return
}

// queueBind bind rabbitmq queue
func (p *producer) queueBind(rabbit *rabbitmq, channel *amqp.Channel, item *queuePushItem) (delivery *queueDelivery, err error) {
	var (
		delay      = 0
		queueName  string
		headers    amqp.Table
		expiration string
		routingKey string
	)
	attempt := item.Attempt
	if attempt <= 0 {
		attempt = 0
	}
	attempt++
	headers, queueName, delay = rabbit.GetHeaderArgs(item.QueueName, attempt, item.Delay)
	pushArgs := rabbit.GetPushArgs(item.QueueName, delay)
	if delay > 0 {
		expiration = fmt.Sprintf("%d", delay*1000)
		routingKey = queueName
	} else {
		routingKey = rabbit.GetRoutingKey(queueName)
	}
	if _, err = channel.QueueDeclare(
		queueName, // name of the queue
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // noWait
		pushArgs,
	); err != nil {
		log.Printf("【Producer】 queue name：%s，QueueDeclare error：%+v", queueName, err)
		return
	}
	if err = channel.QueueBind(
		queueName,       // name of the queue
		routingKey,      // bindingKey
		Config.Exchange, // sourceExchange
		false,           // noWait
		pushArgs,        // arguments
	); err != nil {
		log.Printf("【Producer】 queue：%s，QueueBind error：%+v", queueName, err)
		return
	}
	delivery = &queueDelivery{
		DeliveryMode: 1,
		Headers:      headers,
		MessageId:    item.MessageId,
		Exchange:     Config.Exchange,
		RoutingKey:   routingKey,
		Expiration:   expiration,
	}
	return
}

// publish queue to rabbitmq
func (p *producer) publish(ctx context.Context, item *queuePushItem) (messageId string, err error) {
	var (
		delivery *queueDelivery
		channel  *amqp.Channel
		rabbit   *rabbitmq
	)
	messageId = item.MessageId
	if rabbit, err = p.getPool(); err != nil {
		return
	}
	if channel, err = p.openChannel(rabbit); err != nil {
		return
	}
	defer func(channel *amqp.Channel) {
		_ = channel.Close()
		_ = rabbitmqPool.Put(rabbit)
	}(channel)
	if delivery, err = p.queueBind(rabbit, channel, item); err != nil {
		log.Printf("【Producer】queueBind bind err:%+v", err)
		return
	}
	if err = channel.PublishWithContext(
		ctx,
		delivery.Exchange,   // publish to an exchange
		delivery.RoutingKey, // routing to 0 or more queues
		false,               // mandatory
		false,               // immediate
		amqp.Publishing{
			Headers:      delivery.Headers,
			Body:         item.Body,
			MessageId:    delivery.MessageId,
			DeliveryMode: delivery.DeliveryMode, // 1=non-persistent, 2=persistent
			Expiration:   delivery.Expiration,
			Timestamp:    time.Now(),
		}); err != nil {
		log.Printf("【Producer】 PublishWithContext err:%+v", err)
	}
	return
}
