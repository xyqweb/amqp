package amqp

import (
	"context"
	"encoding/json"
	"errors"
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

var Consumer = consumer{gracefulShutdown: new(atomic.Bool), mux: new(sync.RWMutex)}

type (
	ConsumerHandler func(ctx context.Context, data *QueueData) error
	consumer        struct {
		conn             *amqp.Connection
		rabbit           *rabbitmq
		gracefulShutdown *atomic.Bool
		mux              *sync.RWMutex
		handler          ConsumerHandler
		//rabbitList       map[string]*consumerRabbit
	}
	/*consumerRabbit struct {
		conn   *amqp.Connection
		rabbit *rabbitmq
		status *atomic.Bool
	}*/
)

// get rabbitmq pool
func (c *consumer) getPool() error {
	c.mux.Lock()
	defer c.mux.Unlock()
	//if c.rabbit != nil {
	//	return nil
	//}
	if rabbitItem, err := rabbitmqPool.Get(); err != nil {
		return err
	} else if rabbit, ok := rabbitItem.(*rabbitmq); ok {
		/*memoryAdd := fmt.Sprintf("%p", rabbit)
		if conn := rabbit.OpenConn(); conn != nil {
			c.rabbitList[memoryAdd] = &consumerRabbit{
				conn:   conn,
				rabbit: rabbit,
				status: new(atomic.Bool),
			}
			return nil
		} else {
			return errors.New("cannot open rabbitmq connection")
		}*/
		c.rabbit = rabbit
		return nil
	} else {
		return errors.New("get rabbitmq pool fail")
	}
}

/*
func (c *consumer) getRabbit() *consumerRabbit {
	var instance *consumerRabbit
	if len(c.rabbitList) == 0 {
		if err := c.getPool(); err != nil {
			return nil
		}
	}
	for key, rabbit := range c.rabbitList {
		if !rabbit.status.Load() {
			c.rabbitList[key].status.Swap(true)
			instance = rabbit
		}
	}
	if instance == nil {
		if err := c.getPool(); err != nil {
			return nil
		}
		for key, rabbit := range c.rabbitList {
			if !rabbit.status.Load() {
				c.rabbitList[key].status.Swap(true)
				instance = rabbit
			}
		}
	}
	return instance
}*/

// Start queue listen
func (c *consumer) Start(ctx context.Context, handle ConsumerHandler) error {
	var (
		notifyClose   = make(chan *amqp.Error, 1)
		connectIsDone = make(chan bool, 1)
		isContinue    = true
	)
	notifyConsumerChan := make(chan string, 1)
	c.handler = handle
	//c.rabbitList = make(map[string]*consumerRabbit)
	if err := c.getPool(); err != nil {
		return err
	}
	if conn := c.rabbit.OpenConn(notifyClose); conn == nil {
		return errors.New("cannot open rabbitmq connection")
	} else {
		c.conn = conn
		connectIsDone <- true
	}
	for isContinue {
		select {
		case notifyCloseMsg := <-notifyClose:
			log.Printf("【Consumer】rabbitmq connection notify close %+v,gracefulShutdown %t", notifyCloseMsg, c.gracefulShutdown.Load())
			if c.gracefulShutdown.Load() {
				isContinue = false
				break
			}
			notifyClose = make(chan *amqp.Error, 1)
			if conn := c.rabbit.ReopenConn(notifyClose); conn == nil {
				isContinue = false
				break
			} else {
				c.conn = conn
				connectIsDone <- true
			}
		case <-connectIsDone:
			c.createQueueListen(ctx, notifyConsumerChan)
		case notifyQueueName := <-notifyConsumerChan:
			if !c.conn.IsClosed() {
				c.createQueueListen(ctx, notifyConsumerChan, notifyQueueName)
			}
		}
	}
	close(connectIsDone)
	close(notifyConsumerChan)
	return nil
}

// createQueueListen create queue Consume
func (c *consumer) createQueueListen(ctx context.Context, notifyConsumerChan chan string, queueName ...string) {
	notifyQueueName := ""
	if len(queueName) > 0 && queueName[0] != "" {
		notifyQueueName = queueName[0]
	}
	for name, num := range Config.Queue {
		// When the notification queue is not empty, only a single listener is created for the notification queue
		if notifyQueueName != "" {
			if name != notifyQueueName {
				continue
			}
			num = 1
		}
		for i := 0; i < num; i++ {
			go func(name string) {
				if err := c.createSingleQueueListen(ctx, name); err != nil {
					log.Printf("【Consumer】create %s consumer error %+v", name, err)
					// If gracefulShutdown is false, send queue name to chan
					if !c.gracefulShutdown.Load() {
						notifyConsumerChan <- name
					}
				}
			}(name)
		}
	}
}

// openChannel open rabbitmq connection channel and bind queue
func (c *consumer) openChannel(queueName string, channelNotifyClose chan *amqp.Error) (channel *amqp.Channel, err error) {
	if channel, err = c.conn.Channel(); err != nil {
		if err.Error() == amqp.ErrChannelMax.Error() {
			c.Close(true)
		}
		log.Printf("【Consumer】Create Channel err:%+v", err)
		return
	}
	channel.NotifyClose(channelNotifyClose)
	routingKey := Util.GetRoutingKey(queueName)
	if err = channel.ExchangeDeclare(
		Config.Exchange, // name
		Config.Kind,     // type
		true,            // durable
		false,           // auto-deleted
		false,           // internal
		false,           // noWait
		nil,             // arguments
	); err != nil {
		log.Printf("【Consumer】ExchangeDeclare err:%+v", err)
		return
	}
	args := Util.GetPushArgs(queueName, 0)
	if _, err = channel.QueueDeclare(
		queueName, // name of the queue
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // noWait
		args,
	); err != nil {
		log.Printf("【Consumer】QueueDeclare err:%+v", err)
		return
	}
	if err = channel.QueueBind(
		queueName,       // name of the queue
		routingKey,      // bindingKey
		Config.Exchange, // sourceExchange
		false,           // noWait
		args,            // arguments
	); err != nil {
		log.Printf("【Consumer】QueueBind err:%+v", err)
		return
	}
	if err = channel.Qos(1, 0, false); err != nil {
		log.Printf("【Consumer】queueName %s Qos set err:%+v", queueName, err)
		return
	}
	return
}

// newSingleQueueListen create listen channel
func (c *consumer) createSingleQueueListen(ctx context.Context, queueName string) (err error) {
	var (
		channel            *amqp.Channel
		channelNotifyClose = make(chan *amqp.Error, 1)
		isConnect          = make(chan bool, 1)
		isContinue         = true
	)
	if channel, err = c.openChannel(queueName, channelNotifyClose); err != nil {
		return
	}
	isConnect <- true
	for isContinue {
		select {
		case notifyCloseMsg := <-channelNotifyClose:
			if c.conn.IsClosed() {
				// If connection is closed,break cycle and return
				isContinue = false
				log.Printf("【Consumer】channel is closed,notify message :%+v,connection is close status:true", notifyCloseMsg)
				err = errors.New("connection is closed")
				break
			}
			time.Sleep(1 * time.Second)
			// If connection is not close reopen consume channel
			channelNotifyClose = make(chan *amqp.Error, 1)
			if channel, err = c.openChannel(queueName, channelNotifyClose); err != nil {
				log.Printf("【Consumer】channel reopen fail,notify message :%+v, err:%+v", notifyCloseMsg, err)
				channelNotifyClose <- &amqp.Error{Code: 504, Reason: err.Error()}
			} else {
				isConnect <- true
			}
		case _, ok := <-isConnect:
			if !ok {
				isContinue = false
				err = errors.New("channel isConnect is closed,need notify to restart this queue")
				break
			}
			if err = Util.Try(ctx, func(ctx context.Context) error {
				return c.startSingleQueueConsume(ctx, queueName, channel)
			}); err != nil {
				if c.conn.IsClosed() {
					isContinue = false
					err = errors.New("rabbitmq connection is closed，need reopen connection")
					break
				} else {
					log.Printf("【Consumer】start queue %s listen error:%+v", queueName, err)
					err = nil
				}
			}
		}
	}
	close(isConnect)
	return
}

// startSingleQueueConsume Consume channel
func (c *consumer) startSingleQueueConsume(ctx context.Context, queueName string, channel *amqp.Channel) error {
	if channel.IsClosed() {
		return errors.New("channel is closed")
	}
	deliveries, err := channel.Consume(
		queueName,                             // name
		"amqp-go.ctag-"+Util.RandomString(20), // consumerTag,
		false,                                 // autoAck
		false,                                 // exclusive
		false,                                 // noLocal
		false,                                 // noWait
		nil,                                   // arguments
	)
	if err != nil {
		return err
	}
	for d := range deliveries {
		if err = Util.Try(ctx, func(ctx context.Context) error {
			queueData := new(QueueData)
			execErr := json.Unmarshal(d.Body, &queueData)
			if execErr != nil {
				return errors.New("Queue content cannot be assigned to struct QueueData， %s" + execErr.Error())
			}
			if d.MessageId == "" {
				d.MessageId = Util.RandomString(23)
			}

			// test channel close code,Only opening up for local development
			/*if queueData.Type == "xyqWebTestChannelClose" {
				_ = d.Ack(true)
				_ = channel.Close()
				return nil
			}*/

			queueData.Headers = d.Headers
			queueData.MessageId = d.MessageId
			queueData.QueueName = queueName
			cancelCtx, cancel := context.WithTimeout(ctx, time.Duration(Config.MaxExecuteTime)*time.Second)
			defer cancel()
			if execErr = c.handler(cancelCtx, queueData); execErr != nil {
				if _, pushErr := Producer.PublishByConsumer(ctx, queueData, d.Body); pushErr != nil {
					log.Printf("【Consumer】queue:%+v，consume error：%+v, repush error:%+v", queueData, execErr, pushErr)
				} else {
					log.Printf("【Consumer】queue :%+v，consume error：%+v,already repush", queueData, execErr)
				}
			}
			return execErr
		}); err != nil {
			log.Printf("【Consumer】queue consume error:%+v", err)
		}
		_ = d.Ack(true)
	}
	return err
}

// Close reset connection,
func (c *consumer) Close(gracefulShutdown bool) {
	c.mux.Lock()
	defer c.mux.Unlock()
	_ = rabbitmqPool.Put(c.rabbit)
	c.gracefulShutdown.Swap(gracefulShutdown)
	if gracefulShutdown {
		rabbitmqPool.Close()
	}
	_ = c.conn.Close()
}
