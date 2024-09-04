package authenticated

import (
	"encoding/json"
	"log"
	"sync"
)

type MessageType int

const (
	SEND MessageType = iota
	ECHO
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

type AuthBroadcastState struct {
	OutgoingMessages chan *UnsignedMessage
	NumNodes         int                      // number of nodes
	MessageStates    map[string]*MessageState // message -> message state
	QuoromSize       int                      // size of Byzantine Quorom
	sync.Mutex
}

type MessageState struct {
	Echoes    []bool // array indicating if others have sent echoes
	Delivered bool
}

// RecordEcho records the echo for a process on a message
func (state *AuthBroadcastState) recordEcho(message string, pid int) int {
	state.Lock()
	defer state.Unlock()
	_, exists := state.MessageStates[message]

	if !exists {
		state.MessageStates[message] = &MessageState{
			Echoes:    make([]bool, state.NumNodes),
			Delivered: false,
		}
	}

	state.MessageStates[message].Echoes[pid] = true

	return state.getEchoCount(message)
}

func (state *AuthBroadcastState) isDelivered(message string) bool {
	state.Lock()
	defer state.Unlock()
	msgState, exists := state.MessageStates[message]
	if exists {
		return msgState.Delivered
	}
	return false
}

func (state *AuthBroadcastState) deliver(message string) {
	state.Lock()
	defer state.Unlock()
	_, exists := state.MessageStates[message]
	if exists {
		state.MessageStates[message].Delivered = true
	}
}

func (state *AuthBroadcastState) printMessageStates() {
	state.Lock()
	defer state.Unlock()

	log.Println("Current Message States:")
	for message, msgState := range state.MessageStates {
		log.Printf("Message: %s\n", message)
		log.Printf("Delivered: %v\n", msgState.Delivered)
		log.Printf("Echoes: %v\n", msgState.Echoes)
		log.Println()
	}
}

// GetEchoCount returns the echo count array for a specific message in a thread-safe manner.
func (state *AuthBroadcastState) getEchoCount(message string) (count int) {
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
