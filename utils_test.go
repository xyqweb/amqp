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

func Test_SearchArray(t *testing.T) {
	t.Run("searchArray", func(t *testing.T) {
		if index := Util.SearchArray([]string{"a", "b"}, "c"); index != NotFoundIndex {
			t.Error("testSearchArray fail")
		}
	})
}

func Test_UtilTry(t *testing.T) {
	ctx := context.Background()
	t.Run("try", func(t *testing.T) {
		if err := Util.Try(ctx, func(ctx context.Context) error {
			panic("tryPanic")
		}); err.Error() != "tryPanic" {
			t.Errorf("TestUtilTry tryPanic fail %+v", err)
		}
	})

	t.Run("try", func(t *testing.T) {
		if err := Util.Try(ctx, func(ctx context.Context) error {
			return errors.New("tryErr")
		}); err.Error() != "tryErr" {
			t.Errorf("TestUtilTry tryErr fail %+v", err)
		}
	})
}

func Test_CoverInt(t *testing.T) {
	t.Run("coverInt", func(t *testing.T) {
		if val := Util.CoverInt(1); val != 1 {
			t.Errorf("TestUtilTry coverInt fail")
		}
		if val := Util.CoverInt(uint(1)); val != 1 {
			t.Errorf("TestUtilTry coverInt fail")
		}
		if val := Util.CoverInt(int8(1)); val != 1 {
			t.Errorf("TestUtilTry coverInt fail")
		}
		if val := Util.CoverInt(uint8(1)); val != 1 {
			t.Errorf("TestUtilTry coverInt fail")
		}
		if val := Util.CoverInt(int16(1)); val != 1 {
			t.Errorf("TestUtilTry coverInt fail")
		}
		if val := Util.CoverInt(uint16(1)); val != 1 {
			t.Errorf("TestUtilTry coverInt fail")
		}
		if val := Util.CoverInt(int32(1)); val != 1 {
			t.Errorf("TestUtilTry coverInt fail")
		}
		if val := Util.CoverInt(uint32(1)); val != 1 {
			t.Errorf("TestUtilTry coverInt fail")
		}
		if val := Util.CoverInt(int64(1)); val != 1 {
			t.Errorf("TestUtilTry coverInt fail")
		}
		if val := Util.CoverInt(uint64(1)); val != 1 {
			t.Errorf("TestUtilTry coverInt fail")
		}
		if val := Util.CoverInt(float32(1)); val != 1 {
			t.Errorf("TestUtilTry coverInt fail")
		}
		if val := Util.CoverInt(float64(1)); val != 1 {
			t.Errorf("TestUtilTry coverInt fail")
		}
		if val := Util.CoverInt(true); val != 1 {
			t.Errorf("TestUtilTry coverInt fail")
		}
		if val := Util.CoverInt(false); val != 0 {
			t.Errorf("TestUtilTry coverInt fail")
		}
		if val := Util.CoverInt(nil); val != 0 {
			t.Errorf("TestUtilTry coverInt fail")
		}
		if val := Util.CoverInt("1"); val != 1 {
			t.Errorf("TestUtilTry coverInt fail")
		}
		if val := Util.CoverInt("0"); val != 0 {
			t.Errorf("TestUtilTry coverInt fail")
		}
		if val := Util.CoverInt("a"); val != 0 {
			t.Errorf("TestUtilTry coverInt fail")
		}
		if val := Util.CoverInt([]byte("1")); val != 1 {
			t.Errorf("TestUtilTry coverInt fail")
		}
		if val := Util.CoverInt([]byte("0")); val != 0 {
			t.Errorf("TestUtilTry coverInt fail")
		}
		if val := Util.CoverInt([]byte("a")); val != 0 {
			t.Errorf("TestUtilTry coverInt fail")
		}
		if val := Util.CoverInt(struct{}{}); val != 0 {
			t.Errorf("TestUtilTry coverInt fail")
		}
	})
}
