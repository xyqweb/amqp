package amqp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
	"sync"
	"time"
)

var Producer = producer{mux: new(sync.RWMutex)}

type (
	producer struct {
		interceptTxQueue map[string][]*queuePushItem // intercept database transaction queue
		mux              *sync.RWMutex
	}
)

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

// getTxId get database transaction id
func (p *producer) getTxId(ctx context.Context, transactionId ...string) string {
	if len(transactionId) > 0 && transactionId[0] != "" {
		return transactionId[0]
	}
	if Config.TxId != "" {
		if ctxTxId, ok := ctx.Value(Config.TxId).(string); ok && ctxTxId != "" {
			return ctxTxId
		}
	}
	return ""
}

// TxBegin database transaction begin hook
// transactionId database transaction id
func (p *producer) TxBegin(ctx context.Context, transactionId ...string) error {
	p.mux.Lock()
	defer p.mux.Unlock()
	txId := p.getTxId(ctx, transactionId...)
	if txId == "" {
		return nil
	}
	if p.interceptTxQueue == nil {
		p.interceptTxQueue = map[string][]*queuePushItem{txId: nil}
	}
	return nil
}

// TxCommit database transaction commit hook
// transactionId database transaction id
func (p *producer) TxCommit(ctx context.Context, transactionId ...string) (err error) {
	if p.interceptTxQueue == nil {
		return
	}
	txId := p.getTxId(ctx, transactionId...)
	if txId == "" {
		return nil
	}
	if queueList, ok := p.interceptTxQueue[txId]; ok {
		defer delete(p.interceptTxQueue, txId)
		if err = Util.Try(ctx, func(ctx context.Context) error {
			return p.txPush(ctx, queueList)
		}); err != nil {
			return
		}
	}
	return
}

// TxRollback database transaction rollback hook
// transactionId database transaction id
func (p *producer) TxRollback(ctx context.Context, transactionId ...string) (err error) {
	p.mux.Lock()
	defer p.mux.Unlock()
	if p.interceptTxQueue == nil {
		return
	}
	txId := p.getTxId(ctx, transactionId...)
	if txId == "" {
		return nil
	}
	if _, ok := p.interceptTxQueue[txId]; ok {
		delete(p.interceptTxQueue, txId)
	}
	return
}

// Publish publish queue
func (p *producer) Publish(ctx context.Context, queue *Queue, txId ...string) (messageId string, err error) {
	var (
		needIntercept = false
		queueBody     []byte
	)
	transactionId := p.getTxId(ctx, txId...)
	if transactionId != "" {
		if _, ok := p.interceptTxQueue[transactionId]; ok {
			needIntercept = true
		}
	}
	if queueBody, err = json.Marshal(map[string]interface{}{
		"type": queue.Type,
		"data": queue.Data,
	}); err != nil {
		return
	}
	pushItem := &queuePushItem{
		QueueName: queue.QueueName,
		MessageId: Util.RandomString(32),
		Body:      queueBody,
		Delay:     queue.Delay,
		Ttr:       queue.Ttr,
	}
	if !needIntercept {
		return p.publish(ctx, pushItem)
	} else {
		p.mux.Lock()
		p.interceptTxQueue[transactionId] = append(p.interceptTxQueue[transactionId], pushItem)
		p.mux.Unlock()
		return pushItem.MessageId, nil
	}
}

// PublishByConsumer publish queue for Consumer->startSingleQueueConsume exec fail
func (p *producer) PublishByConsumer(ctx context.Context, data *QueueData, body []byte) (messageId string, err error) {
	var attempt int32
	if headerAttempt, ok := data.Headers[AttemptName]; ok {
		if attempt, ok = headerAttempt.(int32); !ok {
			attempt = DefaultAttempt
		} else if attempt < 1 {
			attempt = DefaultAttempt
		}
	}
	var maxExecuteTime int
	if amqpTtr, ok := data.Headers[TtrKey]; ok {
		maxExecuteTime = Util.CoverInt(amqpTtr)
	} else {
		maxExecuteTime = Config.MaxExecuteTime
	}
	return p.publish(ctx, &queuePushItem{
		QueueName: data.QueueName,
		MessageId: data.MessageId,
		Body:      body,
		Attempt:   attempt,
		Delay:     ErrDelay,
		Ttr:       maxExecuteTime,
	})
}

// openChannel open rabbitmq connection channel
func (p *producer) openChannel(rabbit *rabbitmq) (channel *amqp.Channel, err error) {
retryConn:
	if conn := rabbit.OpenConn(); conn == nil {
		err = errors.New("open Rabbitmq connection fail")
		log.Printf("【Producer】open Rabbitmq connection fail")
		return
	} else {
		if channel, err = conn.Channel(); err != nil {
			if err.Error() == amqp.ErrChannelMax.Error() {
				rabbit.Close()
				goto retryConn
			}
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
func (p *producer) queueBind(channel *amqp.Channel, item *queuePushItem) (delivery *queueDelivery, err error) {
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
	headers, queueName, delay = Util.GetHeaderArgs(item.QueueName, attempt, item.Ttr, item.Delay)
	pushArgs := Util.GetPushArgs(item.QueueName, delay)
	if delay > 0 {
		expiration = fmt.Sprintf("%d", delay*1000)
		routingKey = queueName
	} else {
		routingKey = Util.GetRoutingKey(queueName)
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
	if delivery, err = p.queueBind(channel, item); err != nil {
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

// txPush after database commit push intercept queue
func (p *producer) txPush(ctx context.Context, queueList []*queuePushItem) (err error) {
	var (
		rabbit   *rabbitmq
		channel  *amqp.Channel
		delivery *queueDelivery
	)
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
	p.mux.Lock()
	defer p.mux.Unlock()
	for i := 0; i < 3; i++ {
		if err = Util.Try(ctx, func(ctx context.Context) error {
			if err = channel.Tx(); err != nil {
				log.Printf("create transaction err:%+v", err)
				return err
			}
			for _, item := range queueList {
				if delivery, err = p.queueBind(channel, item); err != nil {
					log.Printf("【Producer】queueBind bind err:%+v", err)
					return err
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
					return err
				}
			}
			return nil
		}); err != nil {
			if txErr := channel.TxRollback(); txErr != nil {
				log.Printf("【Producer】publish queue err:%+v rabbitmq transaction rollback err:%+v", err, txErr)
			} else {
				log.Printf("【Producer】publish queue err:%+v rabbitmq transaction rollback success", err)
			}
		} else if err = channel.TxCommit(); err != nil {
			log.Printf("【Producer】rabbitmq transaction commit err:%+v", err)
		} else {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	return
}
