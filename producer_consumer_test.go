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

func Test_ProducerTxPushContext(t *testing.T) {
	//t.Skip()
	t.Run("push queue by transaction commit", func(t *testing.T) {
		producer := amqp.Producer
		txId := "rK1y4WEeT8mDThVITiFO"
		ctx := context.WithValue(context.Background(), "txId", txId)
		if err := producer.TxBegin(ctx); err != nil {
			t.Errorf("tx begin err:%+v", err)
		}
		if messageId, err := producer.Publish(ctx, &amqp.Queue{
			QueueName: "queue",
			Type:      "txPush",
			Data: map[string]interface{}{
				"time": time.Now().UnixMilli(),
			},
			Delay: 0,
		}); err != nil {
			t.Errorf("tx push err:%+v", err)
		} else if messageId == "" {
			t.Error("tx push err messageId is empty")
		} else {
			t.Logf("tx push success messageId:%s", messageId)
		}
		if messageId, err := producer.Publish(ctx, &amqp.Queue{
			QueueName: "queue",
			Type:      "txPush",
			Data: map[string]interface{}{
				"time": time.Now().UnixMilli(),
			},
			Delay: 0,
		}); err != nil {
			t.Errorf("tx push err:%+v", err)
		} else if messageId == "" {
			t.Error("tx push err messageId is empty")
		} else {
			t.Logf("tx push success messageId:%s", messageId)
		}
		if err := producer.TxCommit(ctx); err != nil {
			t.Errorf("tx commit err:%+v", err)
		}
	})
	t.Run("push queue by transaction rollback", func(t *testing.T) {
		producer := amqp.Producer
		txId := "V8tUmZW0wSpSNORIvh2r"
		ctx := context.WithValue(context.Background(), "txId", txId)
		if err := producer.TxBegin(ctx); err != nil {
			t.Errorf("tx begin err:%+v", err)
		}
		if messageId, err := producer.Publish(ctx, &amqp.Queue{
			QueueName: "queue",
			Type:      "txPush",
			Data: map[string]interface{}{
				"time": time.Now().UnixMilli(),
			},
			Delay: 0,
		}); err != nil {
			t.Errorf("tx push err:%+v", err)
		} else if messageId == "" {
			t.Error("tx push err messageId is empty")
		} else {
			t.Logf("tx push success messageId:%s", messageId)
		}
		if messageId, err := producer.Publish(ctx, &amqp.Queue{
			QueueName: "queue",
			Type:      "txPush",
			Data: map[string]interface{}{
				"time": time.Now().UnixMilli(),
			},
			Delay: 0,
		}); err != nil {
			t.Errorf("tx push err:%+v", err)
		} else if messageId == "" {
			t.Error("tx push err messageId is empty")
		} else {
			t.Logf("tx push success messageId:%s", messageId)
		}
		if err := producer.TxRollback(ctx); err != nil {
			t.Errorf("tx rollback err:%+v", err)
		}
	})
}

func Test_ProducerTxPushParams(t *testing.T) {
	//t.Skip()
	t.Run("push queue by transaction commit", func(t *testing.T) {
		amqp.Config.TxId = ""
		producer := amqp.Producer
		txId := "rK1y4WEeT8mDThVITiFO"
		ctx := context.Background()
		if err := producer.TxBegin(ctx, txId); err != nil {
			t.Errorf("tx begin err:%+v", err)
		}
		if messageId, err := producer.Publish(ctx, &amqp.Queue{
			QueueName: "queue",
			Type:      "txPush",
			Data: map[string]interface{}{
				"time": time.Now().UnixMilli(),
			},
			Delay: 0,
		}, txId); err != nil {
			t.Errorf("tx push err:%+v", err)
		} else if messageId == "" {
			t.Error("tx push err messageId is empty")
		} else {
			t.Logf("tx push success messageId:%s", messageId)
		}
		if messageId, err := producer.Publish(ctx, &amqp.Queue{
			QueueName: "queue",
			Type:      "txPush",
			Data: map[string]interface{}{
				"time": time.Now().UnixMilli(),
			},
			Delay: 0,
		}, txId); err != nil {
			t.Errorf("tx push err:%+v", err)
		} else if messageId == "" {
			t.Error("tx push err messageId is empty")
		} else {
			t.Logf("tx push success messageId:%s", messageId)
		}
		if err := producer.TxCommit(ctx, txId); err != nil {
			t.Errorf("tx commit err:%+v", err)
		}
	})
	t.Run("push queue by transaction rollback", func(t *testing.T) {
		producer := amqp.Producer
		txId := "V8tUmZW0wSpSNORIvh2r"
		ctx := context.WithValue(context.Background(), "txId", txId)
		if err := producer.TxBegin(ctx, txId); err != nil {
			t.Errorf("tx begin err:%+v", err)
		}
		if messageId, err := producer.Publish(ctx, &amqp.Queue{
			QueueName: "queue",
			Type:      "txPush",
			Data: map[string]interface{}{
				"time": time.Now().UnixMilli(),
			},
			Delay: 0,
		}); err != nil {
			t.Errorf("tx push err:%+v", err)
		} else if messageId == "" {
			t.Error("tx push err messageId is empty")
		} else {
			t.Logf("tx push success messageId:%s", messageId)
		}
		if messageId, err := producer.Publish(ctx, &amqp.Queue{
			QueueName: "queue",
			Type:      "txPush",
			Data: map[string]interface{}{
				"time": time.Now().UnixMilli(),
			},
			Delay: 0,
		}, txId); err != nil {
			t.Errorf("tx push err:%+v", err)
		} else if messageId == "" {
			t.Error("tx push err messageId is empty")
		} else {
			t.Logf("tx push success messageId:%s", messageId)
		}
		if err := producer.TxRollback(ctx, txId); err != nil {
			t.Errorf("tx rollback err:%+v", err)
		}
	})
}

func Test_ConsumerStart(t *testing.T) {
	//t.Skip()
	t.Run("start Consumer", func(t *testing.T) {
		consume := amqp.Consumer
		go func() {
			time.Sleep(2 * time.Second)
			consume.Close(false)

			/*_, _ = amqp.Producer.Publish(context.Background(), &amqp.Queue{
				QueueName: "queue",
				Type:      "xyqWebTestChannelClose",
				Data: map[string]interface{}{
					"isCloseChannel": "yes",
				},
			})*/
			t.Log("test connection close and reopen")

			time.Sleep(6 * time.Second)
			consume.Close(true)
		}()
		if err := consume.Start(context.Background(), func(ctx context.Context, data *amqp.QueueData) error {
			if data.Type == "testError" {
				return errors.New("testError")
			}
			return nil
		}); err != nil {
			t.Errorf("TestConsumerStart Start() error = %v", err)
		}
		time.Sleep(2 * time.Second)
	})
}
