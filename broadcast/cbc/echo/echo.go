package echo

import (
	"broadcast-primitives/helpers"
	"broadcast-primitives/networking"
	"io"
	"log"
	"math/rand/v2"
	"os"
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
			OutgoingMessages:  make(chan *Message),
			NumNodes:          numNodes,
			HostPrivateKey:    privKey,
			PeerPublicKeys:    peerPublicKeys,
			DeliveredMessages: make(map[string]bool),
			QuoromSize:        helpers.CalculateByzantineQuoromSize(numNodes),
		}
		wg.Add(1)
		go handleOutgoingUnsignedMessage(wg)
	})
}

func consistentBroadcast(message string) {
	msg := newSendMessage(message)
	BroadcastState.OutgoingMessages <- msg
}

func receivedSend(message string, pid int) {

}

func receivedEcho(message string, pid int) {

}

func receivedFinal(message string, pid int) {

}

func deliverMessage(message string) {
	BroadcastState.deliver(message)
	log.Printf("Delivered Message: %v", message)
	BroadcastState.printMessageStates()
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
				log.Printf("Received message of type %v from process %v", unmarshaledMessage.Type, peerPid)
				if unmarshaledMessage.Type == SEND {
					receivedSend(unmarshaledMessage.Message, peerPid)
				} else if unmarshaledMessage.Type == ECHO {
					receivedEcho(unmarshaledMessage.Message, peerPid)
				} else if unmarshaledMessage.Type == FINAL {
					receivedFinal(unmarshaledMessage.Message, peerPid)
				}
			}
		}
	}
}

func handleOutgoingUnsignedMessage(wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		msg := <-BroadcastState.OutgoingMessages
		marshaledMsg, err := marshalMessage(msg)
		if err != nil {
			log.Panicf("Failed to marshal unsigned message")
		}
		networking.BroadcastMessage(marshaledMsg)
	}
}
