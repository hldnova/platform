package nats

type Subscriber interface {
    // Similar to Subscribe in nats
    Subscribe(subject, group string, handler Handler) error
}
