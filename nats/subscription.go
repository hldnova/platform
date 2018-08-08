package nats

import stan "github.com/nats-io/go-nats-streaming"

type Subscription interface {
	// Pending returns the number of queued messages and queued bytes for this subscription.
	Pending() (int64, int64, error)

	// Delivered returns the number of delivered messages for this subscription.
	Delivered() (int64, error)

	// Close removes this subscriber
	Close() error
}

type subscription struct {
	sub stan.Subscription
}

func (s subscription) Pending() (int64, int64, error) {
	messages, bytes, err := s.sub.Pending()
	return int64(messages), int64(bytes), err
}

func (s subscription) Delivered() (int64, error) {
	return s.sub.Delivered()
}

func (s subscription) Close() error {
	return s.sub.Close()
}

// Durable subscriptions:
// sc, _ := stan.Connect("test-cluster", "client-123")
//
// // Subscribe with durable name
// sc.Subscribe("foo", func(m *stan.Msg) {
//     fmt.Printf("Received a message: %s\n", string(m.Data))
// }, stan.DurableName("my-durable"))
// ...
// // client receives message sequence 1-40
// ...
// // client disconnects for an hour
// ...
// // client reconnects with same clientID "client-123"
// sc, _ := stan.Connect("test-cluster", "client-123")
//
// // client re-subscribes to "foo" with same durable name "my-durable"
// sc.Subscribe("foo", func(m *stan.Msg) {
//     fmt.Printf("Received a message: %s\n", string(m.Data))
// }, stan.DurableName("my-durable"))
// ...
// // client receives messages 41-current
