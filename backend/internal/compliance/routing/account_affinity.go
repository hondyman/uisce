package routing

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	DefaultVNodesPerNode = 100
	DefaultDrainTimeout  = 5 * time.Second
)

// NodeState tracks the lifecycle of a gateway replica node
type NodeState string

const (
	NodeStateActive   NodeState = "ACTIVE"
	NodeStateDraining NodeState = "DRAINING"
	NodeStateStopped  NodeState = "STOPPED"
)

type nodeInfo struct {
	state      NodeState
	drainUntil time.Time
}

// ConsistentHashRouter routes evaluation requests by AccountID to gateway replicas
type ConsistentHashRouter struct {
	mu           sync.RWMutex
	vnodes       int
	ring         []uint32
	vnodeToNode  map[uint32]string
	nodes        map[string]*nodeInfo
	drainTimeout time.Duration
}

// NewConsistentHashRouter creates a new ring router
func NewConsistentHashRouter(vnodes int) *ConsistentHashRouter {
	if vnodes <= 0 {
		vnodes = DefaultVNodesPerNode
	}
	return &ConsistentHashRouter{
		vnodes:       vnodes,
		ring:         make([]uint32, 0),
		vnodeToNode:  make(map[uint32]string),
		nodes:        make(map[string]*nodeInfo),
		drainTimeout: DefaultDrainTimeout,
	}
}

// SetDrainTimeout configures the graceful drain window duration
func (r *ConsistentHashRouter) SetDrainTimeout(d time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.drainTimeout = d
}

// AddNode adds a gateway pod replica to the hash ring
func (r *ConsistentHashRouter) AddNode(nodeID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.nodes[nodeID] = &nodeInfo{state: NodeStateActive}
	for i := 0; i < r.vnodes; i++ {
		vnodeKey := nodeID + "#" + strconv.Itoa(i)
		h := hashKey(vnodeKey)
		r.ring = append(r.ring, h)
		r.vnodeToNode[h] = nodeID
	}
	sort.Slice(r.ring, func(i, j int) bool { return r.ring[i] < r.ring[j] })
}

// DrainNode transitions a node to draining status with a graceful timeout
func (r *ConsistentHashRouter) DrainNode(nodeID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if info, exists := r.nodes[nodeID]; exists {
		info.state = NodeStateDraining
		info.drainUntil = time.Now().UTC().Add(r.drainTimeout)
	}
}

// IsNodeDraining checks if a node is in graceful drain window
func (r *ConsistentHashRouter) IsNodeDraining(nodeID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if info, exists := r.nodes[nodeID]; exists {
		if info.state == NodeStateDraining && time.Now().UTC().Before(info.drainUntil) {
			return true
		}
	}
	return false
}

// RemoveNode removes a node completely from the ring after drain timeout expires
func (r *ConsistentHashRouter) RemoveNode(nodeID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.nodes, nodeID)
	newRing := make([]uint32, 0, len(r.ring))
	for _, h := range r.ring {
		if r.vnodeToNode[h] == nodeID {
			delete(r.vnodeToNode, h)
		} else {
			newRing = append(newRing, h)
		}
	}
	r.ring = newRing
}

// RouteAccount returns the replica node assigned to process evaluations for the given AccountID
func (r *ConsistentHashRouter) RouteAccount(accountID uuid.UUID) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.ring) == 0 {
		return "", errors.New("routing: no active gateway nodes available in ring")
	}

	h := hashKey(accountID.String())
	idx := sort.Search(len(r.ring), func(i int) bool {
		return r.ring[i] >= h
	})

	if idx == len(r.ring) {
		idx = 0
	}

	nodeID := r.vnodeToNode[r.ring[idx]]
	return nodeID, nil
}

func hashKey(key string) uint32 {
	hasher := sha256.New()
	hasher.Write([]byte(key))
	digest := hasher.Sum(nil)
	return binary.BigEndian.Uint32(digest[:4])
}
