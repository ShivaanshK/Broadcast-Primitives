package echo

import (
	"crypto"
	"encoding/json"
	"log"
	"math/big"
	"sync"
)

type MessageType int

const (
	SEND MessageType = iota
	ECHO
	FINAL
)

type Signature struct {
	// Signature fields
	R big.Int
	S big.Int
}

// Message struct handles both unsigned and signed messages
type Message struct {
	Type       MessageType
	Message    string
	Signatures []Signature // If empty, it indicates an unsigned message
}

// NewSendMessage returns a new Message with Type set to SEND and no signatures (unsigned)
func newSendMessage(message string) *Message {
	return &Message{
		Type:    SEND,
		Message: message,
	}
}

// NewEchoMessage returns a new Message with Type set to ECHO and includes signatures
func newEchoMessage(message string, signatures []Signature) *Message {
	return &Message{
		Type:       ECHO,
		Message:    message,
		Signatures: signatures,
	}
}

// EchoBroadcastState manages the state of the echo broadcast system
type EchoBroadcastState struct {
	OutgoingMessages  chan *Message
	NumNodes          int // number of nodes
	NodePublicKeys    []crypto.PublicKey
	DeliveredMessages map[string]bool // message -> bool indicating delivery of message
	QuoromSize        int             // size of Byzantine Quorum
	sync.Mutex
}

// isDelivered checks if a message has already been delivered
func (state *EchoBroadcastState) isDelivered(message string) bool {
	state.Lock()
	defer state.Unlock()
	isDelivered, exists := state.DeliveredMessages[message]
	if exists {
		return isDelivered
	}
	return false
}

// deliver marks a message as delivered
func (state *EchoBroadcastState) deliver(message string) {
	state.Lock()
	defer state.Unlock()
	state.DeliveredMessages[message] = true
}

// printMessageStates logs the current state of message deliveries
func (state *EchoBroadcastState) printMessageStates() {
	state.Lock()
	defer state.Unlock()

	log.Println("Current Message States:")
	for message, isDelivered := range state.DeliveredMessages {
		log.Printf("Message: %s\n", message)
		log.Printf("Delivered: %v\n", isDelivered)
		log.Println()
	}
}

// MarshalMessage serializes a Message struct to JSON
func marshalMessage(msg *Message) ([]byte, error) {
	return json.Marshal(msg)
}

// UnmarshalMessage deserializes JSON data into a Message struct
func unmarshalMessage(data []byte) (*Message, error) {
	var msg Message
	err := json.Unmarshal(data, &msg)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}
