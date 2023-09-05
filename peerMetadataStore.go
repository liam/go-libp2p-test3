package main

import (
	"log"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

type PeerMetadata struct {
	LastHeartbeat        time.Time
	PenultimateHeartbeat time.Time

	Score int
}

const (
	heartbeatInterval = 30 * time.Second
	acceptableDelay   = 100 * time.Millisecond
	minScore          = 0
	maxScore          = 100
)

func (p *PeerMetadata) calculateScore() {
	actualInterval := p.LastHeartbeat.Sub(p.PenultimateHeartbeat)
	delta := actualInterval - heartbeatInterval

	if delta > acceptableDelay {
		p.Score-- // Decrease the score
	}

	// Bound the score within the [minScore, maxScore] range
	if p.Score < minScore {
		p.Score = minScore
	} else if p.Score > maxScore {
		p.Score = maxScore
	}
}

type PeerMetadataStore struct {
	metadata map[peer.ID]*PeerMetadata
	lock     sync.RWMutex
}

func NewPeerMetadataStore() *PeerMetadataStore {
	return &PeerMetadataStore{
		metadata: make(map[peer.ID]*PeerMetadata),
	}
}

func (s *PeerMetadataStore) receiveHeartbeat(id peer.ID) {
	s.lock.Lock()
	defer s.lock.Unlock()

	metadata, exists := s.metadata[id]
	if !exists {
		metadata = &PeerMetadata{Score: 100} // initialise score to 100
		s.metadata[id] = metadata
	}

	metadata.PenultimateHeartbeat = metadata.LastHeartbeat
	metadata.LastHeartbeat = time.Now()
	metadata.calculateScore()

	log.Printf("Received heartbeat from %s, Score = %d\n", id, metadata.Score)
}
