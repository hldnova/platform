type Subscription interface {
    // Ack acknowledges a message.
    Ack() error

    // Pending returns the number of queued messages and queued bytes for this subscription.
    Pending() (int64, int64, error)

    // Delivered returns the number of delivered messages for this subscription.
    Delivered() (int64, error)

    // Close removes this subscriber
    Close() error
}
