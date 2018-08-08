package nats

import (
	stan "github.com/nats-io/go-nats-streaming"
)

type Subscriber interface {
	// Similar to Subscribe in nats
	Subscribe(subject, group string, handler Handler) error
}

type subscriber struct {
	Connection stan.Conn
}

func NewSubscriber(clientID string) (Subscriber, error) {
	sc, err := stan.Connect(ServerName, clientID)
	if err != nil {
		return nil, err
	}

	return &subscriber{Connection: sc}, nil
}

type messageHandler struct {
	handler Handler
	sub     subscription
}

func (mh *messageHandler) handle(m *stan.Msg) {
	mh.handler.Process(mh.sub, &message{m: m})
}

func (s *subscriber) Subscribe(subject, group string, handler Handler) error {
	mh := messageHandler{handler: handler}
	sub, err := s.Connection.QueueSubscribe(subject, group, mh.handle, stan.DurableName(group), stan.SetManualAckMode(), stan.MaxInflight(25))
	if err != nil {
		return err
	}
	mh.sub = subscription{sub: sub}
	return nil
}
