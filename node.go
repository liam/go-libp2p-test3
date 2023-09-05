package main

import (
	"bufio"
	"context"
	"log"
	"os"
	"strings"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"

	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
)

const protocolId = "/peermsg/1.0.0"
const bitcoinProtocolId = "/bitcoin/1.0.0"
const sendAck = false

type Node struct {
	ctx               context.Context
	host              host.Host
	topics            map[string]*pubsub.Topic
	subscriptions     map[string]*pubsub.Subscription
	peerMetadataStore *PeerMetadataStore
}

func NewNode(ctx context.Context, h host.Host, tn []string) (*Node, error) {

	n := &Node{ctx: ctx, host: h}
	go n.discoverPeers(ctx, tn)

	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		return nil, err
	}
	topics := map[string]*pubsub.Topic{}
	subscriptions := map[string]*pubsub.Subscription{}

	for _, topicName := range tn {
		topic, err := ps.Join(topicName)
		if err != nil {
			return nil, err
		}
		topics[topicName] = topic

		sub, err := topic.Subscribe()
		if err != nil {
			return nil, err
		}
		subscriptions[topicName] = sub
	}
	n.topics = topics
	n.subscriptions = subscriptions
	n.peerMetadataStore = NewPeerMetadataStore()

	h.SetStreamHandler(protocolId, func(s network.Stream) {
		go readPeerMsg(s)
	})
	h.SetStreamHandler(bitcoinProtocolId, handleBlockchainMessage)
	return n, nil
}

func (n *Node) discoverPeers(ctx context.Context, tn []string) {
	kademliaDHT := initDHT(ctx, n.host)
	routingDiscovery := drouting.NewRoutingDiscovery(kademliaDHT)
	for _, topicName := range tn {
		dutil.Advertise(ctx, routingDiscovery, topicName)
	}

	// Look for others who have announced and attempt to connect to them
	anyConnected := false
	for !anyConnected {
		for _, topicName := range tn {
			log.Printf("Searching for peers for topic %s..\n", topicName)

			peerChan, err := routingDiscovery.FindPeers(ctx, topicName)
			if err != nil {
				panic(err)
			}

			for peer := range peerChan {
				if peer.ID == n.host.ID() {
					continue // No self connection
				}
				err := n.host.Connect(ctx, peer)
				if err != nil {
					//  we fail to connect to a lot of peers. Just ignore it for now.
					// log.Println("Failed connecting to ", peer.ID.Pretty(), ", error:", err)
				} else {
					log.Println("Connected to:", peer.ID.Pretty())
					anyConnected = true
				}
			}
		}
	}
	log.Println("Peer discovery complete")
	log.Printf("connected to %d peers\n", len(n.host.Network().Peers()))
	log.Printf("peerstore has %d peers\n", len(n.host.Peerstore().Peers()))

}

func (n *Node) StartHeartbeatMessaging(ctx context.Context) {
	// ticker := time.NewTicker(time.Second * 30)
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := n.topics[heartbeatTopicName].Publish(ctx, []byte("heartbeat")); err != nil {
				log.Println("heartbeat publish error:", err)
			}
		}
	}
}

func (n *Node) printHeartbeatFrom(ctx context.Context) {
	for {
		m, err := n.subscriptions[heartbeatTopicName].Next(ctx)
		if err != nil {
			panic(err)
		}
		if m.ReceivedFrom != n.host.ID() {
			n.peerMetadataStore.receiveHeartbeat(m.ReceivedFrom)
		}
	}
}

func (n *Node) StartBlockchainMessaging(ctx context.Context) {

	b := NewBlockchain()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if len(b.Subtrees) < 10 {
				subtreeHash := b.GenerateSubtree()
				if err := n.topics[blockTopicName].Publish(ctx, []byte(subtreeHash)); err != nil {
					log.Println("block publish error:", err)
				}
			} else {
				b.GenerateBlock()
				blockString, _ := b.ToJson()
				if err := n.topics[blockTopicName].Publish(ctx, blockString); err != nil {
					log.Println("block publish error:", err)
				}
				b.Reset()
			}
		}
	}
}

func (n *Node) streamConsoleTo(ctx context.Context) {
	reader := bufio.NewReader(os.Stdin)
	for {
		s, err := reader.ReadString('\n')
		if err != nil {
			panic(err)
		}
		if err := n.topics[chatTopicName].Publish(ctx, []byte(s)); err != nil {
			log.Println("publish error:", err)
		}
	}
}

func (n *Node) printChatMessagesFrom(ctx context.Context) {
	for {
		m, err := n.subscriptions[chatTopicName].Next(ctx)
		if err != nil {
			panic(err)
		}
		if m.ReceivedFrom != n.host.ID() {
			log.Printf("topic: %s - from: %s - message: %s\n", *m.Message.Topic, m.ReceivedFrom.ShortString(), strings.TrimSpace(string(m.Message.Data)))

			h2pi := n.host.Peerstore().PeerInfo(m.ReceivedFrom)
			log.Printf("dialing %s\n", h2pi.Addrs)
			if err := n.host.Connect(ctx, h2pi); err != nil {
				log.Printf("Failed to connect: %+v\n", err)
			}

			s, err := n.host.NewStream(
				ctx,
				m.ReceivedFrom,
				protocolId,
			)
			if err != nil {
				log.Printf("failed to create stream: %+v\n", err)
				return
			}
			err = writePeerMsg(s, "Acknowledge\n")
			if err != nil {
				log.Printf("failed to write to stream: %+v\n", err)
				return
			}
		}

	}
}

func (n *Node) printMessagesFrom(ctx context.Context) {
	for {
		m, err := n.subscriptions[blockTopicName].Next(ctx)
		if err != nil {
			panic(err)
		}
		if m.ReceivedFrom != n.host.ID() {
			log.Printf("topic: %s - from: %s - message: %s\n", *m.Message.Topic, m.ReceivedFrom.ShortString(), strings.TrimSpace(string(m.Message.Data)))

			if sendAck {
				h2pi := n.host.Peerstore().PeerInfo(m.ReceivedFrom)
				log.Printf("dialing %s\n", h2pi.Addrs)
				if err := n.host.Connect(ctx, h2pi); err != nil {
					log.Printf("Failed to connect: %+v\n", err)
				}

				s, err := n.host.NewStream(
					ctx,
					m.ReceivedFrom,
					protocolId,
				)
				if err != nil {
					log.Printf("failed to create stream: %+v\n", err)
					return
				}
				err = writePeerMsg(s, "Acknowledge\n")
				if err != nil {
					log.Printf("failed to write to stream: %+v\n", err)
					return
				}
			}
		}
	}
}

func readPeerMsg(s network.Stream) error {
	buf := bufio.NewReader(s)
	str, err := buf.ReadString('\n')
	if err != nil {
		return err
	}

	log.Printf("read: p2p message %s from: %s\n", strings.TrimSpace(str), s.Conn().RemotePeer().ShortString())
	return err
}

func writePeerMsg(s network.Stream, msg string) error {
	_, err := s.Write([]byte(msg))
	return err
}
