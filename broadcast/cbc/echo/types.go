package echo

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/libp2p/go-libp2p/core/crypto"
)

type MessageType int

const (
	SEND MessageType = iota
	ECHO
	FINAL
)

type Signature []byte

// Message struct handles both unsigned and signed messages
type Message struct {
	Type       MessageType
	Message    string
	Signatures []Signature // If empty, it indicates an unsigned message
}

type OutgoingMessage struct {
	Message   *Message
	Recipient string // multiaddr to unicast to. If "", will broadcast to every peer
}

// NewSendMessage returns a new Message with Type set to SEND and no signatures (unsigned)
func newSendMessage(message string) *Message {
	return &Message{
		Type:    SEND,
		Message: message,
	}
}

// NewEchoMessage returns a new Message with Type set to ECHO and includes signatures
func newEchoMessage(message string, sig Signature) (echo *Message) {
	echo = &Message{
		Type:       ECHO,
		Message:    message,
		Signatures: make([]Signature, 1),
	}
	echo.Signatures[0] = sig
	return
}

// NewEchoMessage returns a new Message with Type set to ECHO and includes signatures
func newFinalMessage(message string, sigs []Signature) *Message {
	return &Message{
		Type:       FINAL,
		Message:    message,
		Signatures: sigs,
	}
}

// NewSendMessage returns a new Message with Type set to SEND and no signatures (unsigned)
func newOutgoingMessage(message *Message, recipient string) *OutgoingMessage {
	return &OutgoingMessage{
		Message:   message,
		Recipient: recipient,
	}
}

// EchoBroadcastState manages the state of the echo broadcast system
type EchoBroadcastState struct {
	OutgoingMessages  chan *OutgoingMessage
	NumNodes          int // number of nodes
	HostPrivateKey    crypto.PrivKey
	PeerPublicKeys    []crypto.PubKey
	EchoesReceived    map[string][]Signature
	DeliveredMessages map[string]bool // message -> bool indicating delivery of message
	QuoromSize        int             // size of Byzantine Quorum
	sync.Mutex
}

// record echo
func (state *EchoBroadcastState) recordEcho(message string, sig Signature, echoerPid int) int {
	state.Lock()
	defer state.Unlock()
	_, exists := state.EchoesReceived[message]

	if !exists {
		state.EchoesReceived[message] = make([]Signature, state.NumNodes)
	}

	state.EchoesReceived[message][echoerPid] = sig

	return state.countEchoes(message)
}

// countEchoes returns the number of non-empty echoes (non-empty signatures) for a given message
func (state *EchoBroadcastState) countEchoes(message string) int {
	echoes, exists := state.EchoesReceived[message]
	if !exists {
		return 0
	}

	// Count non-empty echoes (non-empty byte slices)
	count := 0
	for _, sig := range echoes {
		if len(sig) > 0 { // Only count non-empty byte arrays
			count++
		}
	}

	return count
}

// deliver marks a message as delivered
func (state *EchoBroadcastState) deliver(message string) {
	state.Lock()
	defer state.Unlock()
	state.DeliveredMessages[message] = true
}

// printMessageStates logs the current state of message deliveries
func (state *EchoBroadcastState) PrintMessageStates() {
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
