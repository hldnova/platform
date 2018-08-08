package nats

import (
	stand "github.com/nats-io/nats-streaming-server/server"
	stores "github.com/nats-io/nats-streaming-server/stores"
)

const ServerName = "platform"

type Server struct {
	Server *stand.StanServer
}

// this really doesn't serve any purpose atm
func CreateServer() (*Server, error) {
	opts := stand.GetDefaultOptions()
	opts.StoreType = stores.TypeFile
	opts.ID = ServerName
	opts.FilestoreDir = "datastore"
	opts.IOSleepTime = stand.DefaultIOSleepTime
	server, err := stand.RunServerWithOpts(opts, nil)
	s := &Server{Server: server}

	return s, err
}
