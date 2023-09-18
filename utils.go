package amqp

import (
	"context"
	"fmt"
	amqp "github.com/rabbitmq/amqp091-go"
	"math/rand"
	"strconv"
	"strings"
)

var Util = util{}

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

type util struct {
}

// Try implements try... logistics using internal panic...recover.
// It returns error if any exception occurs or return err, or else it returns nil.
func (u *util) Try(ctx context.Context, try func(ctx context.Context) error) (err error) {
	defer func() {
		if exception := recover(); exception != nil {
			if v, ok := exception.(error); ok {
				err = v
			} else {
				err = fmt.Errorf(`%+v`, exception)
			}
		}
	}()
	err = try(ctx)
	return
}

// RandomString create random string
func (u *util) RandomString(n int) string {
	sb := strings.Builder{}
	sb.Grow(n)
	for i := 0; i < n; i++ {
		sb.WriteByte(charset[rand.Intn(len(charset))])
	}
	return sb.String()
}

// SearchArray search whether string `s` in slice `a`.
func (u *util) SearchArray(a []string, s string) int {
	for i, v := range a {
		if s == v {
			return i
		}
	}
	return NotFoundIndex
}

// InArray checks whether string `s` in slice `a`.
func (u *util) InArray(a []string, s string) bool {
	return u.SearchArray(a, s) != NotFoundIndex
}

// fillArgs fill args
func (u *util) fillArgs(args amqp.Table) amqp.Table {
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
func (u *util) GetPushArgs(queueName string, delay int) amqp.Table {
	args := u.fillArgs(Config.PushArgs)
	if delay > 0 {
		args[DeadLetterExchange] = Config.Exchange
		args[DeadLetterRoutingKey] = u.GetRoutingKey(queueName)
		args[amqp.QueueMessageTTLArg] = delay * 1000
	}
	return args
}

// GetHeaderArgs 获取推送头
func (u *util) GetHeaderArgs(queueName string, attempt int32, ttr, delay int) (amqp.Table, string, int) {
	args := u.fillArgs(Config.HeaderArgs)
	args[AttemptName] = attempt
	if ttr <= 0 {
		args[TtrKey] = Config.MaxExecuteTime
	} else {
		args[TtrKey] = ttr
	}
	if attempt > Config.MaxFail {
		delay = 0
		queueName += "Error"
	}
	if delay > 0 {
		args[DelayKey] = delay
		queueName = fmt.Sprintf("%s.%s.%d.delay", Config.Exchange, queueName, delay*1000)
	}
	return args, queueName, delay
}

// GetRoutingKey get queue routing key
func (u *util) GetRoutingKey(queueName string) string {
	return queueName + "Key"
}

// CoverInt converts `any` to int.
func (u *util) CoverInt(any interface{}) int {
	if any == nil {
		return 0
	}
	switch value := any.(type) {
	case int:
		return value
	case int8:
		return int(value)
	case int16:
		return int(value)
	case int32:
		return int(value)
	case int64:
		return int(value)
	case uint:
		return int(value)
	case uint8:
		return int(value)
	case uint16:
		return int(value)
	case uint32:
		return int(value)
	case uint64:
		return int(value)
	case float32:
		return int(value)
	case float64:
		return int(value)
	case bool:
		if value {
			return 1
		}
		return 0
	case []byte:
		s := string(value)
		if val, err := strconv.Atoi(s); err != nil {
			return 0
		} else {
			return val
		}
	case string:
		if val, err := strconv.Atoi(value); err != nil {
			return 0
		} else {
			return val
		}
	default:
		return 0
	}
}
