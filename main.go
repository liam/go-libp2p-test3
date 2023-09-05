package main

import (
	"context"
	"flag"
	"log"

	"github.com/libp2p/go-libp2p"
)

var (
	heartbeatTopicName = "heartbeat"

	chatTopicName    = "chat"
	blockTopicName   = "block"
	subtreeTopicName = "subtree"
)

func main() {
	topicNames := []string{heartbeatTopicName, chatTopicName, blockTopicName, subtreeTopicName}

	flag.Parse()
	ctx := context.Background()

	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	if err != nil {
		panic(err)
	}
	log.Printf("peer ID: %s", h.ID().Pretty())
	log.Printf("Connect to me on:")
	for _, addr := range h.Addrs() {
		log.Printf("  %s/p2p/%s", addr, h.ID().Pretty())
	}

	node, err := NewNode(ctx, h, topicNames)
	if err != nil {
		panic(err)
	}

	go node.StartHeartbeatMessaging(ctx)
	go node.printHeartbeatFrom(ctx)

	go node.streamConsoleTo(ctx)
	go node.printChatMessagesFrom(ctx)

	go node.StartBlockchainMessaging(ctx)

	go node.printMessagesFrom(ctx)

	select {}
}
