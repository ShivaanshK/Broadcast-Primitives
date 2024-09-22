package bracha

import (
	"broadcast-primitives/helpers"
	"broadcast-primitives/networking"
	"io"
	"log"
	"math/rand/v2"
	"os"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
)

var BroadcastState BrachaBroadcastState
var once sync.Once

func StartBroadcastSimulation(pid int, serverAddr string, peers map[string]int, numNodes int, wg *sync.WaitGroup) {
	initBroadcastState(numNodes, wg)

	networking.StartHost(pid, serverAddr, peers, handleIncomingMessages, wg)
	time.Sleep(7 * time.Second) // Give time to all peers to start their hosts
	networking.EstablishConnections()
	time.Sleep(3 * time.Second) // Give time to all peers to establish connections

	rand.New(rand.NewPCG(uint64(os.Getpid()), uint64(time.Now().Unix()))) // Seed the random number generator

	for {
		// Generate a random number between 0 and numNodes-1
		if rand.IntN(numNodes) == 0 {
			log.Print("Got picked to broadcast\n")
			randomMsg := helpers.RandomMessage(10)
			reliableBroadcast(randomMsg)
		}

		// Sleep for a random duration between 1 and 5 seconds
		sleepDuration := time.Duration(rand.IntN(5)+1) * time.Second
		log.Printf("Going to sleep for %v\n", sleepDuration)
		time.Sleep(sleepDuration)
	}
}

func initBroadcastState(numNodes int, wg *sync.WaitGroup) {
	once.Do(func() {
		BroadcastState = BrachaBroadcastState{
			OutgoingMessages:   make(chan *UnsignedMessage),
			NumNodes:           numNodes,
			MessageStates:      make(map[string]*MessageState),
			QuoromSize:         helpers.CalculateByzantineQuoromSize(numNodes),
			SingleHonestQuorom: helpers.CalculateSingleHonestQuoromSize(numNodes),
		}
		wg.Add(1)
		go handleOutgoingUnsignedMessage(wg)
	})
}

func reliableBroadcast(message string) {
	msg := newUnsignedSendMessage(message)
	BroadcastState.OutgoingMessages <- msg
	// Leaders echo is implicit on broadcasting a SEND
	BroadcastState.recordEcho(message, networking.NodeCtx.Pid)
}

func receivedSend(message string, pid int) {
	// Leaders echo is implicit on broadcasting a SEND
	BroadcastState.recordEcho(message, pid)
	// Record your own echo too
	echoesReceived := BroadcastState.recordEcho(message, networking.NodeCtx.Pid)

	// Broadcast an echo
	echo := newUnsignedEchoMessage(message)
	BroadcastState.OutgoingMessages <- echo

	// Assuming FIFO channels but doesnt mean others' echoes don't reach this node before leader's send
	if echoesReceived >= BroadcastState.QuoromSize {
		// Record your own ready
		readysReceived := BroadcastState.recordReady(message, networking.NodeCtx.Pid)
		// Broadcast a ready
		ready := newUnsignedReadyMessage(message)
		BroadcastState.OutgoingMessages <- ready

		// If ready quorom exists and haven't delivered, deliver message
		if readysReceived >= BroadcastState.QuoromSize && !BroadcastState.isDelivered(message) {
			deliverMessage(message)
		}
	}
}

func receivedEcho(message string, pid int) {
	// Record their echo
	echoesReceived := BroadcastState.recordEcho(message, pid)

	// If received echo quorom and didnt broadcast ready, do so
	if echoesReceived >= BroadcastState.QuoromSize && !BroadcastState.sentReady(message, networking.NodeCtx.Pid) {
		// Record your own ready
		BroadcastState.recordReady(message, networking.NodeCtx.Pid)

		// Broadcast a ready
		ready := newUnsignedReadyMessage(message)
		BroadcastState.OutgoingMessages <- ready
	}
}

func receivedReady(message string, pid int) {
	// Record their echo
	readysReceived := BroadcastState.recordReady(message, pid)

	// If received ready from at least 1 honest node and didnt broadcast ready, do so
	if readysReceived >= BroadcastState.SingleHonestQuorom && !BroadcastState.sentReady(message, networking.NodeCtx.Pid) {
		// Record your own ready
		readysReceived = BroadcastState.recordReady(message, networking.NodeCtx.Pid)

		// Broadcast a ready
		ready := newUnsignedReadyMessage(message)
		BroadcastState.OutgoingMessages <- ready
	}

	// If ready quorom exists and haven't delivered, then deliver
	if readysReceived >= BroadcastState.QuoromSize && !BroadcastState.isDelivered(message) {
		deliverMessage(message)
	}
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
			unmarshaledMessage, err := unmarshalUnsignedMessage(msg)
			if err != nil {
				log.Printf("Error unmarshaling message: %v", err)
			} else {
				log.Printf("Received message of type %v from process %v", unmarshaledMessage.Type, peerPid)
				if unmarshaledMessage.Type == SEND {
					receivedSend(unmarshaledMessage.Message, peerPid)
				} else if unmarshaledMessage.Type == ECHO {
					receivedEcho(unmarshaledMessage.Message, peerPid)
				} else if unmarshaledMessage.Type == READY {
					receivedReady(unmarshaledMessage.Message, peerPid)
				}
			}
		}
	}
}

func handleOutgoingUnsignedMessage(wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		msg := <-BroadcastState.OutgoingMessages
		marshaledMsg, err := marshalUnsignedMessage(msg)
		if err != nil {
			log.Panicf("Failed to marshal message: %v", err)
		}
		networking.BroadcastMessage(marshaledMsg)
	}
}
