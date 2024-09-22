package echo

import (
	"broadcast-primitives/helpers"
	"broadcast-primitives/networking"
	"io"
	"log"
	"math/rand/v2"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
)

var BroadcastState EchoBroadcastState
var once sync.Once

func StartBroadcastSimulation(pid int, serverAddr string, peers map[string]int, numNodes int, privKey crypto.PrivKey, peerPublicKeys []crypto.PubKey, wg *sync.WaitGroup) {
	initBroadcastState(numNodes, privKey, peerPublicKeys, wg)

	log.Println(peerPublicKeys)

	networking.StartHost(pid, serverAddr, peers, handleIncomingMessages, wg)
	time.Sleep(7 * time.Second) // Give time to all peers to start their hosts
	networking.EstablishConnections()
	time.Sleep(3 * time.Second) // Give time to all peers to establish connections

	rand.New(rand.NewPCG(uint64(os.Getpid()), uint64(time.Now().Unix()))) // Seed the random number generator

	for {
		// Generate a random number between 0 and numNodes-1
		if rand.IntN(numNodes) == 0 {
			randomMsg := helpers.RandomMessage(10)
			consistentBroadcast(randomMsg)
		}

		// Sleep for a random duration between 1 and 5 seconds
		sleepDuration := time.Duration(rand.IntN(5)+1) * time.Second
		time.Sleep(sleepDuration)
	}
}

func initBroadcastState(numNodes int, privKey crypto.PrivKey, peerPublicKeys []crypto.PubKey, wg *sync.WaitGroup) {
	once.Do(func() {
		BroadcastState = EchoBroadcastState{
			OutgoingMessages:  make(chan *OutgoingMessage),
			NumNodes:          numNodes,
			HostPrivateKey:    privKey,
			PeerPublicKeys:    peerPublicKeys,
			EchoesReceived:    make(map[string][]Signature),
			DeliveredMessages: make(map[string]bool),
			QuoromSize:        helpers.CalculateByzantineQuoromSize(numNodes),
		}
		wg.Add(1)
		go handleOutgoingUnsignedMessage(wg)
	})
}

func consistentBroadcast(message string) {
	// Record your own ECHO for the SEND message. It is implicit
	msgToSign := []byte("ECHO" + strconv.Itoa(networking.NodeCtx.Pid) + message)
	signature, err := BroadcastState.HostPrivateKey.Sign(msgToSign)
	if err != nil {
		log.Panicf("Failed to sign my own ECHO for message %v", message)
	}
	BroadcastState.recordEcho(message, signature, networking.NodeCtx.Pid)

	// Broadcast SEND message
	msg := newSendMessage(message)
	outgoingMsg := newOutgoingMessage(msg, "")
	BroadcastState.OutgoingMessages <- outgoingMsg
}

func receivedSend(message *Message, peerMultiAddr string) {
	leaderPid := networking.NodeCtx.PeersPids[peerMultiAddr]

	// Sign an echo message
	msgToSign := []byte("ECHO" + strconv.Itoa(leaderPid) + message.Message)
	signature, err := BroadcastState.HostPrivateKey.Sign(msgToSign)
	if err != nil {
		log.Panicf("Failed to sign ECHO for message %v", message)
	}

	// Create echo message with signature at first index
	echo := newEchoMessage(message.Message, signature)

	// Unicast echo back to the leader
	outgoingEcho := newOutgoingMessage(echo, peerMultiAddr)
	BroadcastState.OutgoingMessages <- outgoingEcho
}

func receivedEcho(message *Message, peerMultiAddr string) {
	peerPid := networking.NodeCtx.PeersPids[peerMultiAddr]

	// Verify the signature of the peer
	dataSigned := []byte("ECHO" + strconv.Itoa(networking.NodeCtx.Pid) + message.Message)
	peerSig := message.Signatures[0]
	peerPubKey := BroadcastState.PeerPublicKeys[peerPid]
	valid, err := peerPubKey.Verify(dataSigned, peerSig)
	if err != nil {
		log.Panicf("Failed to verify ECHO sig from peer %v on message %v", peerPid, message.Message)
	}
	if !valid {
		return
	}

	// Record echo signature for sending in the final
	numEchoes := BroadcastState.recordEcho(message.Message, message.Signatures[0], peerPid)
	// Broadcast FINAL message once quorom reached
	if numEchoes == BroadcastState.QuoromSize {
		echoSigs := BroadcastState.EchoesReceived[message.Message]
		finalMsg := newFinalMessage(message.Message, echoSigs)
		outgoingMsg := newOutgoingMessage(finalMsg, "")
		BroadcastState.OutgoingMessages <- outgoingMsg
		deliverMessage(finalMsg.Message)
	}
}

func receivedFinal(message *Message, peerMultiAddr string) {
	leaderPid := networking.NodeCtx.PeersPids[peerMultiAddr]

	validEchoSigs := 0
	for peerPid, peerSig := range message.Signatures {
		if len(peerSig) > 0 {
			dataSigned := []byte("ECHO" + strconv.Itoa(leaderPid) + message.Message)
			peerPubKey := BroadcastState.PeerPublicKeys[peerPid]
			valid, err := peerPubKey.Verify(dataSigned, peerSig)
			if err != nil {
				log.Panicf("Failed to verify ECHO sig from peer %v on message %v", peerPid, message.Message)
			}
			if valid {
				validEchoSigs++
			}
		}
	}

	if validEchoSigs >= BroadcastState.QuoromSize {
		// Update echo sigs for DA
		BroadcastState.EchoesReceived[message.Message] = message.Signatures
		// Deliver message
		deliverMessage(message.Message)
	}
}

func deliverMessage(message string) {
	BroadcastState.deliver(message)
	log.Printf("Delivered Message: %v", message)
	BroadcastState.PrintMessageStates()
}

func handleIncomingMessages(stream network.Stream) {
	defer func() {
		networking.RemoveStream(stream)
		stream.Close()
	}()

	fullMultiAddr := stream.Conn().RemoteMultiaddr().String() + "/p2p/" + stream.Conn().RemotePeer().String()
	peerPid := networking.NodeCtx.PeersPids[fullMultiAddr]
	log.Printf("Stream opened to peer %v with multiaddr %v", peerPid, fullMultiAddr)

	buffer := make([]byte, 1024) // Buffer to hold incoming data

	for {
		n, err := stream.Read(buffer)
		if err != nil {
			if err != io.EOF {
				log.Panicf("Error reading from stream with %v: %v", stream.Conn().RemoteMultiaddr().String(), err)
			}
			break
		}

		if n > 0 {
			msg := buffer[:n]
			unmarshaledMessage, err := unmarshalMessage(msg)
			if err != nil {
				log.Printf("Error unmarshaling message: %v", err)
			} else {
				log.Printf("Received message of type %v from process %v", unmarshaledMessage.Type, fullMultiAddr)
				if unmarshaledMessage.Type == SEND {
					receivedSend(unmarshaledMessage, fullMultiAddr)
				} else if unmarshaledMessage.Type == ECHO {
					receivedEcho(unmarshaledMessage, fullMultiAddr)
				} else if unmarshaledMessage.Type == FINAL {
					receivedFinal(unmarshaledMessage, fullMultiAddr)
				}
			}
		}
	}
}

func handleOutgoingUnsignedMessage(wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		msg := <-BroadcastState.OutgoingMessages
		marshaledMsg, err := marshalMessage(msg.Message)
		if err != nil {
			log.Panicf("Failed to marshal unsigned message")
		}
		if msg.Recipient == "" {
			networking.BroadcastMessage(marshaledMsg)
		} else {
			networking.UnicastMessage(marshaledMsg, msg.Recipient)
		}
	}
}
