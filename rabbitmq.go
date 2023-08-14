package amqp

import (
	"encoding/json"
	"errors"
	"fmt"
	amqp "github.com/rabbitmq/amqp091-go"
	"gopkg.in/yaml.v3"
	"log"
	"os"
	"sync"
	"time"
)

const attemptName = "attempt"

var (
	Config         *config
	RabbitmqConfig = rabbitmqConfig{}
	rabbitmqPool   *Pool
)

type (
	rabbitmqConfig struct{}
	rabbitmq       struct {
		conn *amqp.Connection
		mux  *sync.RWMutex
	}
)

// SetYaml 设置yaml文件配置
// file full path to yaml file
func (r *rabbitmqConfig) SetYaml(file string) error {
	conf := new(config)
	if data, err := os.ReadFile(file); err != nil {
		return err
	} else if err = yaml.Unmarshal(data, &conf); err != nil {
		return err
	}
	return r.verify(conf)
}

// SetJson 设置json文件配置
// file full path to json file
func (r *rabbitmqConfig) SetJson(file string) error {
	if data, err := os.ReadFile(file); err != nil {
		return err
	} else {
		return r.SetJsonData(string(data))
	}
}

// SetJsonData 设置json配置
// data rabbitmq config params
func (r *rabbitmqConfig) SetJsonData(data string) error {
	conf := new(config)
	if err := json.Unmarshal([]byte(data), &conf); err != nil {
		return err
	}
	return r.verify(conf)
}

// verify 校验配置
func (r *rabbitmqConfig) verify(conf *config) error {
	if conf.Host == "" || conf.User == "" || conf.Password == "" || conf.Vhost == "" || len(conf.Queue) == 0 {
		return errors.New("rabbitmq conf error")
	}
	if conf.Kind == "" || !r.InArray([]string{amqp.ExchangeDirect, amqp.ExchangeTopic, amqp.ExchangeHeaders, amqp.ExchangeFanout}, conf.Kind) {
		conf.Kind = amqp.ExchangeDirect
	}
	if conf.Exchange == "" {
		conf.Exchange = "exchange"
	}
	if len(conf.PushArgs) > 0 {
		for k, v := range conf.PushArgs {
			if val, ok := v.(float64); ok {
				conf.PushArgs[k] = int(val)
			}
		}
	}
	if len(conf.HeaderArgs) > 0 {
		for k, v := range conf.HeaderArgs {
			if val, ok := v.(float64); ok {
				conf.HeaderArgs[k] = int(val)
			}
		}
	}
	Config = conf
	return nil
}

// SearchArray search whether string `s` in slice `a`.
func (r *rabbitmqConfig) SearchArray(a []string, s string) int {
	for i, v := range a {
		if s == v {
			return i
		}
	}
	return -1
}

// InArray checks whether string `s` in slice `a`.
func (r *rabbitmqConfig) InArray(a []string, s string) bool {
	return r.SearchArray(a, s) != -1
}

// initiate rabbitmq pool
func init() {
	rabbitmqPool = NewPool(10*time.Second, func() (interface{}, error) {
		rabbit := rabbitmq{mux: new(sync.RWMutex)}
		return &rabbit, nil
	}, func(i interface{}) {
		i.(*rabbitmq).Close()
	})
}

// newConn create Rabbitmq connection
func (r *rabbitmq) newConn() error {
	if Config == nil {
		return errors.New("no rabbitmq config")
	}
	r.mux.Lock()
	defer r.mux.Unlock()
	amqpUrl := fmt.Sprintf("amqp://%s:%s@%s:%d/%s", Config.User, Config.Password, Config.Host, Config.Port, Config.Vhost)
	conn, err := amqp.DialConfig(amqpUrl, amqp.Config{Heartbeat: time.Duration(Config.Heartbeat) * time.Second})
	if err != nil {
		return err
	}
	r.conn = conn
	return nil
}

// OpenConn get rabbitmq connection
func (r *rabbitmq) OpenConn(notifyClose ...chan *amqp.Error) *amqp.Connection {
	if r.conn == nil || r.conn.IsClosed() {
		if err := r.newConn(); err != nil {
			log.Printf("【Rabbitmq】create Connection fail:%+v", err)
			return nil
		}
	}
	if len(notifyClose) > 0 && notifyClose[0] != nil {
		r.conn.NotifyClose(notifyClose[0])
	}
	return r.conn
}

// ReopenConn reopen rabbitmq connection
func (r *rabbitmq) ReopenConn(notifyClose chan *amqp.Error) *amqp.Connection {
	if r.conn != nil && !r.conn.IsClosed() {
		r.Close()
	}

	isContinue := true
	for isContinue {
		if err := r.newConn(); err != nil {
			log.Printf("【Rabbitmq】create Connection fail:%+v", err)
			time.Sleep(2 * time.Second)
		} else {
			isContinue = false
		}
	}
	r.conn.NotifyClose(notifyClose)
	return r.conn
}

// fillArgs fill args
func (r *rabbitmq) fillArgs(args amqp.Table) amqp.Table {
	argTable := amqp.Table{}
	for k, v := range args {
		if vInt, isInt := v.(int); isInt {
			argTable[k] = vInt
		} else if vStr, isStr := v.(string); isStr {
			argTable[k] = vStr
		}
	}
	return argTable
}

// GetPushArgs 获取args
func (r *rabbitmq) GetPushArgs(queueName string, delay int) amqp.Table {
	args := r.fillArgs(Config.PushArgs)
	if delay > 0 {
		args["x-dead-letter-exchange"] = Config.Exchange
		args["x-dead-letter-routing-key"] = r.GetRoutingKey(queueName)
		args[amqp.QueueMessageTTLArg] = delay * 1000
	}
	return args
}

// GetHeaderArgs 获取推送头
func (r *rabbitmq) GetHeaderArgs(queueName string, attempt int32, delay int) (amqp.Table, string, int) {
	args := r.fillArgs(Config.HeaderArgs)
	args[attemptName] = attempt
	if attempt > Config.MaxFail {
		delay = 0
		queueName += "Error"
	}
	if delay > 0 {
		args["delay"] = delay
		queueName = fmt.Sprintf("%s.%s.%d.delay", Config.Exchange, queueName, delay*1000)
	}
	return args, queueName, delay
}

// GetRoutingKey get queue routing key
func (r *rabbitmq) GetRoutingKey(queueName string) string {
	return queueName + "Key"
}

// Close rabbitmq connection
func (r *rabbitmq) Close() {
	fmt.Println("rabbitmq close ", r)
	r.mux.Lock()
	defer r.mux.Unlock()
	if r.conn != nil && !r.conn.IsClosed() {
		_ = r.conn.Close()
	}
}
