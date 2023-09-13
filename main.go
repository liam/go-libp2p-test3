package main

import (
	"context"
	"flag"
	"fmt"
)

var (
	heartbeatTopicName = "heartbeat"

	chatTopicName    = "chat"
	blockTopicName   = "block"
	subtreeTopicName = "subtree"
)

func main() {
	instanceId := flag.String("instance-id", "", "Unique identifier for this instance")
	flag.Parse()

	if *instanceId == "" {
		fmt.Println("Please provide an instance Id, eg  --instance-id=1")
		return
	}

	topicNames := []string{heartbeatTopicName, chatTopicName, blockTopicName, subtreeTopicName}

	flag.Parse()
	ctx := context.Background()

	node, err := NewNode(*instanceId, ctx, topicNames)
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
