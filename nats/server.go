package nats

import "github.com/nats-io/gnatsd/server"

func RunServer(opts *server.Options) *server.Server {
  s := server.New(opts)
  if s == nil {
    panic("No NATS Server object returned.")
  }

  go s.Start()

  if !s.ReadyForConnections(10 * time.Second) {
    panic("Unable to start NATS Server in Go Routine")
  }
  return s
}
