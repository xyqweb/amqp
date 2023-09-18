package amqp

import (
	"container/list"
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

var poolTick *time.Ticker

type Pool struct {
	list    *list.List                  // Available/idle items list.
	closed  *atomic.Value               // Whether the pool is closed.
	TTL     time.Duration               // Time To Live for pool items.
	NewFunc func() (interface{}, error) // Callback function to create pool item.
	// ExpireFunc is the for expired items destruction.
	// This function needs to be defined when the pool items
	// need to perform additional destruction operations.
	// Eg: net.Conn, os.File, etc.
	ExpireFunc func(interface{})
	mutex      *sync.RWMutex
}

// Pool item.
type poolItem struct {
	value    interface{} // Item value.
	expireAt int64       // Expire timestamp in milliseconds.
}

// NewFunc Creation function for object.
type NewFunc func() (interface{}, error)

// ExpireFunc Destruction function for object.
type ExpireFunc func(interface{})

// NewPool creates and returns a new object pool.
// To ensure execution efficiency, the expiration time cannot be modified once it is set.
//
// Note the expiration logic:
// ttl = 0 : not expired;
// ttl < 0 : immediate expired after use;
// ttl > 0 : timeout expired;
func NewPool(ttl time.Duration, newFunc NewFunc, expireFunc ...ExpireFunc) *Pool {
	r := &Pool{
		list:    list.New(),
		closed:  new(atomic.Value),
		TTL:     ttl,
		NewFunc: newFunc,
		mutex:   new(sync.RWMutex),
	}
	r.closed.Swap(false)
	if len(expireFunc) > 0 {
		r.ExpireFunc = expireFunc[0]
	}
	go HeartBeater(time.Second, r.checkExpireItems)
	return r
}

// Put puts an item to pool.
func (p *Pool) Put(value interface{}) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if p.closed.Load().(bool) {
		return errors.New("pool is closed")
	}
	item := &poolItem{
		value: value,
	}
	if p.TTL == 0 {
		item.expireAt = 0
	} else {
		item.expireAt = p.TimestampMilli() + p.TTL.Milliseconds()
	}
	p.list.PushBack(item)
	return nil
}

// Clear clears pool, which means it will remove all items from pool.
func (p *Pool) Clear() {
	if p.Size() > 0 && p.ExpireFunc != nil {
		for {
			if r := p.popFront(); r != nil {
				p.ExpireFunc(r.(*poolItem).value)
			} else {
				break
			}
		}
	} else {
		p.mutex.Lock()
		p.list = list.New()
		p.mutex.Unlock()
	}
}

// Get picks and returns an item from pool. If the pool is empty and NewFunc is defined,
// it creates and returns one from NewFunc.
func (p *Pool) Get() (interface{}, error) {
	for !p.closed.Load().(bool) {
		if r := p.popFront(); r != nil {
			f := r.(*poolItem)
			if f.expireAt == 0 || f.expireAt > p.TimestampMilli() {
				return f.value, nil
			} else if p.ExpireFunc != nil {
				p.ExpireFunc(f.value)
			}
		} else {
			break
		}
	}

	if p.NewFunc != nil {
		return p.NewFunc()
	}
	return nil, errors.New("pool is empty")
}

// Size returns the count of available items of pool.
func (p *Pool) Size() int {
	return p.list.Len()
}

// Close closes the pool. If `p` has ExpireFunc,
// then it automatically closes all items using this function before it's closed.
// Commonly you do not need to call this function manually.
func (p *Pool) Close() {
	p.Clear()
	p.closed.Swap(true)
}

// checkExpire removes expired items from pool in every second.
func (p *Pool) checkExpireItems() {
	if p.closed.Load().(bool) {
		// If p has ExpireFunc,
		// then it must close all items using this function.
		if p.ExpireFunc != nil {
			for {
				if r := p.popFront(); r != nil {
					p.ExpireFunc(r.(*poolItem).value)
				} else {
					break
				}
			}
		}
		panic("exit")
	}
	// All items do not expire.
	if p.TTL == 0 {
		return
	}
	// The latest item expire timestamp in milliseconds.
	var latestExpire int64 = -1
	// Retrieve the current timestamp in milliseconds, it expires the items
	// by comparing with this timestamp. It is not accurate comparison for
	// every item expired, but high performance.
	var timestampMilli = p.TimestampMilli()
	for {
		if latestExpire > timestampMilli {
			break
		}
		if r := p.popFront(); r != nil {
			item := r.(*poolItem)
			latestExpire = item.expireAt
			if item.expireAt > timestampMilli {
				p.mutex.Lock()
				p.list.PushFront(item)
				p.mutex.Unlock()
				break
			}
			if p.ExpireFunc != nil {
				p.ExpireFunc(item.value)
			}
		} else {
			break
		}
	}
}

func (p *Pool) popFront() (value interface{}) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if p.list == nil {
		p.list = list.New()
		return
	}

	if e := p.list.Front(); e != nil {
		value = p.list.Remove(e)
	}
	return
}

// TimestampMilli returns the timestamp in milliseconds.
func (p *Pool) TimestampMilli() int64 {
	return time.Now().UnixMilli()
}

// HeartBeater is a convenience function for add singleton mode job.
func HeartBeater(interval time.Duration, job func()) {
	var sendTicks <-chan time.Time
	if poolTick == nil {
		poolTick = time.NewTicker(interval)
		sendTicks = poolTick.C
	}
	for {
		if at := <-sendTicks; at.String() != "" {
			if err := Util.Try(context.Background(), func(ctx context.Context) error {
				job()
				return nil
			}); err != nil {
				poolTick.Stop()
				break
			}
		}
	}
}
