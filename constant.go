package amqp

const (
	VERSION                    = "1.0.1"
	AttemptName                = "attempt"
	DefaultExchange            = "exchange"
	NotFoundIndex              = -1
	DefaultAttempt       int32 = 1  // default exec times
	ErrDelay                   = 60 // default exec queue fail push delay time,unit second
	DeadLetterExchange         = "x-dead-letter-exchange"
	DeadLetterRoutingKey       = "x-dead-letter-routing-key"
	DelayKey                   = "amqp-delay"
)
