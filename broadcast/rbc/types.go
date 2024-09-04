package bracha

import (
	"encoding/json"
	"log"
	"sync"
)

type MessageType int

const (
	SEND MessageType = iota
	ECHO
	READY
)

type UnsignedMessage struct {
	Type    MessageType
	Message string
}

// NewSendMessagereturns a new SendMessage with Type set to SEND
func newUnsignedSendMessage(message string) *UnsignedMessage {
	return &UnsignedMessage{
		Type:    SEND,
		Message: message,
	}
}

// NewEchoMessage returns a new EchoMessage with Type set to ECHO
func newUnsignedEchoMessage(message string) *UnsignedMessage {
	return &UnsignedMessage{
		Type:    ECHO,
		Message: message,
	}
}

// NewEchoMessage returns a new EchoMessage with Type set to ECHO
func newUnsignedReadyMessage(message string) *UnsignedMessage {
	return &UnsignedMessage{
		Type:    READY,
		Message: message,
	}
}

type BrachaBroadcastState struct {
	OutgoingMessages   chan *UnsignedMessage
	NumNodes           int                      // number of nodes
	MessageStates      map[string]*MessageState // message -> message state
	QuoromSize         int                      // size of Byzantine Quorom
	SingleHonestQuorom int                      // size of a single honest Quorom
	sync.Mutex
}

type MessageState struct {
	Echoes    []bool // array indicating if others have sent echoes
	Readys    []bool // array indicating if others have sent readys
	Delivered bool
}

// RecordEcho records the echo for a process on a message
func (state *BrachaBroadcastState) recordEcho(message string, pid int) int {
	state.Lock()
	defer state.Unlock()
	_, exists := state.MessageStates[message]

	if !exists {
		state.MessageStates[message] = &MessageState{
			Echoes:    make([]bool, state.NumNodes),
			Readys:    make([]bool, state.NumNodes),
			Delivered: false,
		}
	}

	state.MessageStates[message].Echoes[pid] = true

	return state.getEchoCount(message)
}

// GetEchoCount returns the echo count array for a specific message in a thread-safe manner.
func (state *BrachaBroadcastState) getEchoCount(message string) (count int) {
	_, exists := state.MessageStates[message]
	if exists {
		echoArray := state.MessageStates[message].Echoes
		for _, echoed := range echoArray {
			if echoed {
				count++
			}
		}
	}
	return
}

// RecordEcho records the echo for a process on a message
func (state *BrachaBroadcastState) recordReady(message string, pid int) int {
	state.Lock()
	defer state.Unlock()
	_, exists := state.MessageStates[message]

	if !exists {
		state.MessageStates[message] = &MessageState{
			Echoes:    make([]bool, state.NumNodes),
			Readys:    make([]bool, state.NumNodes),
			Delivered: false,
		}
	}

	state.MessageStates[message].Readys[pid] = true

	return state.getReadyCount(message)
}

// GetEchoCount returns the echo count array for a specific message in a thread-safe manner.
func (state *BrachaBroadcastState) getReadyCount(message string) (count int) {
	_, exists := state.MessageStates[message]
	if exists {
		readyArray := state.MessageStates[message].Readys
		for _, ready := range readyArray {
			if ready {
				count++
			}
		}
	}
	return
}

func (state *BrachaBroadcastState) isDelivered(message string) bool {
	state.Lock()
	defer state.Unlock()
	msgState, exists := state.MessageStates[message]
	if exists {
		return msgState.Delivered
	}
	return false
}

func (state *BrachaBroadcastState) deliver(message string) {
	state.Lock()
	defer state.Unlock()
	_, exists := state.MessageStates[message]
	if exists {
		state.MessageStates[message].Delivered = true
	}
}

func (state *BrachaBroadcastState) printMessageStates() {
	state.Lock()
	defer state.Unlock()

	log.Println("Current Message States:")
	for message, msgState := range state.MessageStates {
		log.Printf("Message: %s\n", message)
		log.Printf("Delivered: %v\n", msgState.Delivered)
		log.Printf("Echoes: %v\n", msgState.Echoes)
		log.Printf("Readys: %v\n", msgState.Readys)
		log.Println()
	}
}

func marshalUnsignedMessage(msg *UnsignedMessage) ([]byte, error) {
	return json.Marshal(msg)
}

func unmarshalUnsignedMessage(data []byte) (*UnsignedMessage, error) {
	var msg UnsignedMessage
	err := json.Unmarshal(data, &msg)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}
