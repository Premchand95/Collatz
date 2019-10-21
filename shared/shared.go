package shared

var RabbitMQUrl = "amqp://guest:guest@localhost:5672/"

type CollatzTask struct {
	Number uint64
}
