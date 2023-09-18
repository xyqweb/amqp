package amqp

import amqp "github.com/rabbitmq/amqp091-go"

type config struct {
	Host           string                 `json:"host" yaml:"host"`
	Port           int                    `json:"port" yaml:"port"` // rabbitmq port default 5672
	User           string                 `json:"user" yaml:"user"`
	Password       string                 `json:"password" yaml:"password"`
	Vhost          string                 `json:"vhost" yaml:"vhost"`
	Exchange       string                 `json:"exchange" yaml:"exchange"`
	MaxFail        int32                  `json:"maxFail,omitempty" yaml:"maxFail"`               // Maximum number of consumable errors
	MaxExecuteTime int                    `json:"maxExecuteTime,omitempty" yaml:"maxExecuteTime"` // Maximum executable time of a single queue message
	Heartbeat      int                    `json:"heartbeat,omitempty" yaml:"heartbeat"`
	Kind           string                 `json:"kind,omitempty" yaml:"kind"`
	Queue          map[string]int         `json:"queue,omitempty" yaml:"queue"`           // string queue name,uint customers num
	HeaderArgs     map[string]interface{} `json:"headerArgs,omitempty" yaml:"headerArgs"` //
	PushArgs       map[string]interface{} `json:"pushArgs,omitempty" yaml:"pushArgs"`     //
	TxId           string                 `json:"txId" yaml:"txId"`
}

// QueueData 队列内容
type QueueData struct {
	MessageId string      `json:"messageId"`
	Headers   amqp.Table  `json:"headers"`
	QueueName string      `json:"queue_name"`
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Ttr       int         `json:"ttr"`
}

// queuePushItem 队列推送
type queuePushItem struct {
	QueueName string `json:"queue_name"`
	MessageId string `json:"MessageId"`
	Attempt   int32  `json:"attempt"`
	Delay     int    `json:"delay"`
	Ttr       int    `json:"ttr"`
	Body      []byte `json:"Body"`
}

type Queue struct {
	QueueName string      `json:"queue_name"`
	Type      string      `json:"type"`
	Data      interface{} `json:"data"` // Data和Body只需要传一个，Body的级别比Data更高
	Delay     int         `json:"delay"`
	Ttr       int         `json:"ttr"`
}

type queueDelivery struct {
	Headers      amqp.Table `json:"Headers"`
	DeliveryMode uint8      `json:"DeliveryMode"`
	MessageId    string     `json:"MessageId"`
	Exchange     string     `json:"Exchange"`
	RoutingKey   string     `json:"RoutingKey"`
	Expiration   string     `json:"Expiration"`
	Body         []byte     `json:"Body"`
}
