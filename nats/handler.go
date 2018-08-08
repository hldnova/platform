package nats

import "go.uber.org/zap"

type Handler interface {
	// Mashup of golang http handler and nat's Subscription
	// The subscription is used to send back acks and such similar
	// to the http response writer
	//
	// When a handler believes the message has been handled, it needs
	// to run m.Ack()
	//
	// If the handler can no longer accept data (like error conditions and such)
	// it needs to run s.Close()
	//
	// If the message is bad, corrupted or whatever, the handler should likely
	// run m.Ack() and log about it.
	Process(s Subscription, m Message)
}

type LogHandler struct {
	Logger *zap.Logger
	Name   string
}

func (lh *LogHandler) Process(s Subscription, m Message) {
	lh.Logger.Info(lh.Name + "|" + string(m.Data()))
	m.Ack()
}
