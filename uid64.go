package uid64

import (
	"errors"
	"fmt"
	"hash/fnv"
	"math"
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"
)

// Distributed Sequence Generator.
// Inspired by Twitter snowflake: https://github.com/twitter/snowflake/tree/snowflake-2010

const (
	unusedBits   = 1
	epochBits    = 41
	nodeIDBits   = 10
	sequenceBits = 12
)

var (
	maxNodeID   = int(math.Pow(2, nodeIDBits) - 1)
	maxSequence = int(math.Pow(2, sequenceBits) - 1)
	// Custom Epoch (January 1, 2015 Midnight UTC = 2015-01-01T00:00:00Z)
	customEpoch = int64(1420070400000)
)

var (
	ErrInvalidState     = errors.New("the system clock is invalid")
	ErrOutOfBoundNodeID = fmt.Errorf("nodeID must be between 0 and %d", maxNodeID)
)

type Generator struct {
	nodeID        int
	lock          sync.Mutex
	lastTimestamp int64
	sequence      int64
}

func New() *Generator {
	return &Generator{
		lastTimestamp: -1,
	}
}

func NewWithNodeID(nodeID int) (*Generator, error) {
	if nodeID < 0 || nodeID > maxNodeID {
		return nil, ErrOutOfBoundNodeID
	}

	return &Generator{
		lastTimestamp: -1,
		nodeID:        nodeID,
	}, nil
}

func (g *Generator) NextID() (int64, error) {
	g.lock.Lock()
	defer g.lock.Unlock()
	if g.nodeID == 0 {
		nid, err := createNodeID()
		if err != nil {
			return 0, err
		}
		g.nodeID = nid
	}

	currentTimestamp := timestamp()

	switch {
	case currentTimestamp < g.lastTimestamp:
		return 0, ErrInvalidState
	case currentTimestamp == g.lastTimestamp:
		g.sequence = (g.sequence + 1) & int64(maxSequence)
		if g.sequence == 0 {
			// Sequence Exhausted, wait till next millisecond.
			currentTimestamp = g.blockWaitToNextMillisecond(currentTimestamp)
		}
	default:
		g.sequence = 0
	}

	g.lastTimestamp = currentTimestamp
	id := currentTimestamp << (nodeIDBits + sequenceBits)
	id |= int64(g.nodeID) << sequenceBits
	id |= g.sequence
	return id, nil

}

func (g *Generator) blockWaitToNextMillisecond(currentTimestamp int64) int64 {
	for g.lastTimestamp == currentTimestamp {
		currentTimestamp = timestamp()
	}
	return currentTimestamp
}

func createNodeID() (int, error) {
	var nodeID int
	ifaces, err := net.Interfaces()
	if err != nil {
		return nodeID, err
	}
	var sb strings.Builder
	for i := 0; i < len(ifaces); i++ {
		mac := ifaces[i].HardwareAddr
		for _, b := range mac {
			sb.WriteString(fmt.Sprintf("%02X", b))
		}
	}
	if sb.Len() == 0 {
		return rand.Int(), nil
	}
	h := fnv.New32a()
	h.Write([]byte(sb.String()))
	return int(h.Sum32()) & maxNodeID, nil
}

func timestamp() int64 {
	return (time.Now().UnixNano() / int64(time.Millisecond)) - customEpoch
}
