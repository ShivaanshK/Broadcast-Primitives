package authcbc

import (
	"broadcast-primitives/helpers"
	"broadcast-primitives/networking"
	"broadcast-primitives/types"
	"io"
	"log"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
)

var BroadcastState types.AuthBroadcastState
var once sync.Once

func StartBroadcast(pid int, serverAddr string, peers map[string]int, numNodes int, wg *sync.WaitGroup) {
	initBroadcastState(numNodes, wg)

	networking.StartHost(pid, serverAddr, peers, handleIncomingMessages, wg)
	time.Sleep(7 * time.Second) // Give time to all peers to start their hosts
	networking.EstablishConnections()
	time.Sleep(3 * time.Second) // Give time to all peers to establish connections

	if pid == BroadcastState.GetLeader() {
		randomMsg := helpers.RandomMessage(10)
		consistentBroadcast(randomMsg)
	}
}

func initBroadcastState(numNodes int, wg *sync.WaitGroup) {
	once.Do(func() {
		BroadcastState = types.AuthBroadcastState{
			OutgoingMessages: make(chan *types.UnsignedMessage),
			NumNodes:         numNodes,
			Leader:           0,
			EchoCount:        make(map[string][]bool),
			QuoromSize:       helpers.CalculateQuoromSize(numNodes),
		}
		wg.Add(1)
		go networking.HandleOutgoingUnsignedMessage(BroadcastState.OutgoingMessages, wg)
	})
}

func consistentBroadcast(message string) {
	msg := types.NewUnsignedSendMessage(message)
	BroadcastState.OutgoingMessages <- msg
	// Leaders echo is implicit on broadcasting a SEND
	BroadcastState.RecordEcho(message, networking.NodeCtx.Pid)
}

func receivedSend(message string, pid int) {
	// Leaders echo is implicit on broadcasting a SEND
	BroadcastState.RecordEcho(message, pid)
	// Broadcast an echo
	echo := types.NewUnsignedEchoMessage(message)
	BroadcastState.OutgoingMessages <- echo
	// Record your own echo too
	echoesReceived := BroadcastState.RecordEcho(message, networking.NodeCtx.Pid)
	// Assuming FIFO channels but doesnt mean others' echoes don't reach this node before leader's send
	if echoesReceived >= BroadcastState.QuoromSize {
		deliverMessage(message)
	}
}

func receivedEcho(message string, pid int) {
	echoesReceived := BroadcastState.RecordEcho(message, pid)
	if echoesReceived >= BroadcastState.QuoromSize {
		// Deliver upon quorom
		deliverMessage(message)
	}
}

func deliverMessage(message string) {
	log.Printf("Delivered Message: %v", message)
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
			unmarshaledMessage, err := networking.UnmarshalUnsignedMessage(msg)
			if err != nil {
				log.Printf("Error unmarshaling message: %v", err)
			} else {
				log.Printf("Received message of type %v from process %v", unmarshaledMessage.Type, peerPid)
				currLeader := BroadcastState.GetLeader()
				if unmarshaledMessage.Type == types.SEND && peerPid == currLeader {
					// Received Send from leader
					receivedSend(unmarshaledMessage.Message, peerPid)
				} else if unmarshaledMessage.Type == types.ECHO {
					receivedEcho(unmarshaledMessage.Message, peerPid)
				}
			}
		}
	}
}
