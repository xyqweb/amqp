package amqp

import (
	"context"
	"errors"
	"testing"
)

func Test_UtilRandomString(t *testing.T) {
	t.Run("randomString", func(t *testing.T) {
		random := Util.RandomString(10)
		if len(random) != 10 {
			t.Error("TestUtilRandomString fail")
		}
	})
}

func Test_UtilTry(t *testing.T) {
	ctx := context.Background()
	t.Run("try", func(t *testing.T) {
		err := Util.Try(ctx, func(ctx context.Context) error {
			panic("tryPanic")
		})
		if err.Error() != "tryPanic" {
			t.Errorf("TestUtilTry tryPanic fail %+v", err)
		}
	})
	t.Run("try", func(t *testing.T) {
		err := Util.Try(ctx, func(ctx context.Context) error {
			return errors.New("tryErr")
		})
		if err.Error() != "tryErr" {
			t.Errorf("TestUtilTry tryErr fail %+v", err)
		}
	})
}
