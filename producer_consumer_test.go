package amqp_test

import (
	"context"
	"errors"
	"github.com/rabbitmq/amqp091-go"
	"github.com/xyqweb/amqp"
	"testing"
	"time"
)

func Test_SetConf(t *testing.T) {
	//t.Skip()
	t.Run("set config yaml file", func(t *testing.T) {
		if err := amqp.RabbitmqConfig.SetYaml("_test.yaml"); err != nil {
			t.Errorf("TestSetConf SetYamlConf() error = %v", err)
		}
	})

	t.Run("set config json file", func(t *testing.T) {
		if err := amqp.RabbitmqConfig.SetJson("_test.json"); err != nil {
			t.Errorf("TestSetConf SetJsonConf() error = %v", err)
		}
	})
}

func Test_ProducerPublish(t *testing.T) {
	//t.Skip()
	t.Run("publish", func(t *testing.T) {
		if messageId, err := amqp.Producer.Publish(context.Background(), &amqp.Queue{
			QueueName: "queue",
			Type:      "publish",
			Data: map[string]interface{}{
				"time":  time.Now().Unix(),
				"delay": "no",
			},
		}); err != nil {
			t.Errorf("push queue error:%+v", err)
		} else if messageId == "" {
			t.Error("push queue error,message id is empty")
		}
	})
	t.Run("publish", func(t *testing.T) {
		if messageId, err := amqp.Producer.Publish(context.Background(), &amqp.Queue{
			QueueName: "queue",
			Type:      "testError",
			Data: map[string]interface{}{
				"time":  time.Now().Unix(),
				"delay": "no",
			},
		}); err != nil {
			t.Errorf("push queue error:%+v", err)
		} else if messageId == "" {
			t.Error("push queue error,message id is empty")
		}
	})
	t.Run("publish delay", func(t *testing.T) {
		if messageId, err := amqp.Producer.Publish(context.Background(), &amqp.Queue{
			QueueName: "queue",
			Type:      "publish",
			Data: map[string]interface{}{
				"time":  time.Now().Unix(),
				"delay": "yes",
			},
			Delay: 60,
		}); err != nil {
			t.Errorf("push queue error:%+v", err)
		} else if messageId == "" {
			t.Error("push queue error,message id is empty")
		}
	})
}

func Test_ProducerPublishByConsumer(t *testing.T) {
	//t.Skip()
	ctx := context.Background()
	t.Run("publishByConsumer", func(t *testing.T) {
		tempId := "6421428acfed96.73182736"
		if messageId, err := amqp.Producer.PublishByConsumer(ctx, &amqp.QueueData{
			MessageId: tempId,
			QueueName: "queue",
			Type:      "test",
		}, []byte(`{"data":{"publishByConsumer":"yes","time":1691805260},"type":"publish"}`)); err != nil {
			t.Errorf("push queue by customer error:%+v", err)
		} else if messageId != tempId {
			t.Errorf("push queue  by customer error,push message id:%s", messageId)
		}
	})
	t.Run("PublishByConsumerToErrorQueue", func(t *testing.T) {
		tempId := "6421428acfed96.83182736"
		if messageId, err := amqp.Producer.PublishByConsumer(ctx, &amqp.QueueData{
			MessageId: tempId,
			Headers: amqp091.Table{
				"attempt": 3,
			},
			QueueName: "queue",
			Type:      "test",
		}, []byte(`{"data":{"publishByConsumer":"toErrorQueue","time":1691805260},"type":"publish"}`)); err != nil {
			t.Errorf("push queue by customer error:%+v", err)
		} else if messageId != tempId {
			t.Errorf("push queue  by customer error,push message id:%s", messageId)
		}
	})
}
func Test_ConsumerStart(t *testing.T) {
	//t.Skip()
	t.Run("start Consumer", func(t *testing.T) {
		consume := amqp.Consumer
		go func() {
			time.Sleep(2 * time.Second)
			consume.Close(true)
			time.Sleep(8 * time.Second)
			consume.Close()
		}()
		if err := consume.Start(context.Background(), func(ctx context.Context, data *amqp.QueueData) error {
			if data.Type == "testError" {
				return errors.New("testError")
			}
			return nil
		}); err != nil {
			t.Errorf("TestConsumerStart Start() error = %v", err)
		}
	})
}
