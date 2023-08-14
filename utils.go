package amqp

import (
	"context"
	"fmt"
	"math/rand"
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
