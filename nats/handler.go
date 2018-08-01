package nats

type Handler interface {
    // Mashup of golang http handler and nat's Subscription
    // The subscription is used to send back acks and such similar
    // to the http response writer
    //
    // When a handler believes the data has been handled, it needs
    // to run s.Ack()
    //
    // If the handler can no longer accept data (like error conditions and such)
    // it needs to run s.Close()
    //
    // If the message is bad, corrupted or whatever, the handler should likely
    // run s.Ack() and log about it.
    Process(s Subscription, data []byte)
}
