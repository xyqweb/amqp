package amqp

const (
	VERSION               = "1.0.0"
	AttemptName           = "attempt"
	DefaultExchange       = "exchange"
	NotFoundIndex         = -1
	DefaultAttempt  int32 = 1  // default exec times
	ErrDelay              = 60 // default exec queue fail push delay time,unit second
)
