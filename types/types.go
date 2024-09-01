package types

import (
	"sync"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
)

type BroadcastPrimitive int

type MessageType int

const (
	AUTH_CBC BroadcastPrimitive = iota
)

const (
	SEND MessageType = iota
	ECHO
	FINAL
)

type UnsignedMessage struct {
	Type    MessageType
	Message string
}

// NewSendMessagereturns a new SendMessage with Type set to SEND
func NewUnsignedSendMessage(message string) *UnsignedMessage {
	return &UnsignedMessage{
		Type:    SEND,
		Message: message,
	}
}

// NewEchoMessage returns a new EchoMessage with Type set to ECHO
func NewUnsignedEchoMessage(message string) *UnsignedMessage {
	return &UnsignedMessage{
		Type:    ECHO,
		Message: message,
	}
}

// NodeCtx holds the host address, peer addresses, and connections.
type NodeCtx struct {
	Pid       int
	PeersPids map[string]int
	Host      host.Host
	Streams   []network.Stream
	sync.Mutex
}

type AuthBroadcastState struct {
	OutgoingMessages chan *UnsignedMessage
	NumNodes         int               // number of nodes
	Leader           int               // pid
	EchoCount        map[string][]bool // message -> bool array indicating if others have sent echoes
	QuoromSize       int               // size of Byzantine Quorom
	sync.Mutex
}

// SetLeader sets the Leader field in a thread-safe manner.
func (state *AuthBroadcastState) SetLeader(leader int) {
	state.Lock()
	defer state.Unlock()
	state.Leader = leader
}

// GetLeader returns the Leader field in a thread-safe manner.
func (state *AuthBroadcastState) GetLeader() int {
	state.Lock()
	defer state.Unlock()
	return state.Leader
}

// RecordEcho records the echo for a process on a message
func (state *AuthBroadcastState) RecordEcho(message string, pid int) int {
	state.Lock()
	defer state.Unlock()
	_, exists := state.EchoCount[message]

	if exists {
		state.EchoCount[message][pid] = true
	} else {
		state.EchoCount[message] = make([]bool, state.NumNodes)
		state.EchoCount[message][pid] = true
	}

	return state.getEchoCount(message)
}

// GetEchoCount returns the echo count array for a specific message in a thread-safe manner.
func (state *AuthBroadcastState) getEchoCount(message string) (count int) {
	echoArray, exists := state.EchoCount[message]
	if exists {
		for _, echoed := range echoArray {
			if echoed {
				count++
			}
		}
	}
	return
}
