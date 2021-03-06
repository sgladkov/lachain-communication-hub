package main

import (
	"github.com/ethereum/go-ethereum/log"
	"github.com/juju/loggo"
	"lachain-communication-hub/config"
	server "lachain-communication-hub/grpc"
	"lachain-communication-hub/peer"
	"lachain-communication-hub/relay"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	loggo.ConfigureLoggers("<root>=INFO")
	if len(os.Args) <= 1 {
		localPeer := peer.New("_h1")
		s := server.New(config.GRPCPort, localPeer)
		go s.Serve()
	} else {
		if os.Args[1] == "-relay" {
			relay.Run()
		}
		if os.Args[1] == "-nolookup" {
			config.DisableIpLookup()
			localPeer := peer.New("_h1")
			server.New(config.GRPCPort, localPeer)
		}
	}

	// wait for a SIGINT or SIGTERM signal
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	log.Info("Received signal, shutting down...")
}
