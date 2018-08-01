package nats

type Publisher interface {
    // Similar to Publish in nats
    Publish(subject string, data []byte) error
}
