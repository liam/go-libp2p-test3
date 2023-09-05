package main

import (
	"io"
	"log"

	"github.com/libp2p/go-libp2p/core/network"
)

func handleBlockchainMessage(s network.Stream) {
	buf, err := io.ReadAll(s)
	if err != nil {
		s.Reset()
		log.Println(err)
		return
	}
	s.Close()
	log.Printf("Received: %s", string(buf))
}
