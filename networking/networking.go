package networking

import (
	"broadcast-primitives/helpers"
	"broadcast-primitives/types"
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

const PROTCOL_ID = "/broadcast/1.0.0"

// NodeCtx is the singleton instance of NodeCtx.
var NodeCtx *types.NodeCtx

// once is used to ensure that initCtx is only called once.
var once sync.Once

func initCtx(host host.Host, pid int, peers map[string]int) {
	once.Do(func() {
		NodeCtx = &types.NodeCtx{
			Pid:       pid,
			PeersPids: peers,
			Host:      host,
			Streams:   make([]network.Stream, 0),
		}
	})
}

// StartHost starts the host and starts listening on the provided address
func StartHost(pid int, hostAddr string, peers map[string]int, handleStream network.StreamHandler, wg *sync.WaitGroup) {
	priv, err := helpers.GetKey(pid)
	if err != nil {
		log.Panicf("Error getting private key for peer %v: %v", pid, err)
	}

	multiAddr, err := helpers.ParseMultiaddress(hostAddr)
	if err != nil {
		log.Panicf("Error parsing multiaddress for %v: %v", pid, err)
	}
	host, err := libp2p.New(
		// Use the keypair
		libp2p.Identity(priv),
		// Multiple listen addresses
		libp2p.ListenAddrStrings(multiAddr),
	)
	if err != nil {
		log.Panicf("Error starting the host: %v", err)
	}

	initCtx(host, pid, peers)

	NodeCtx.Host.SetStreamHandler(PROTCOL_ID, handleStream)
	log.Println("---SUCCESSFULLY INITIALIZED HOST---")
	log.Printf("Host Peer ID: %v", NodeCtx.Host.ID().String())

	wg.Add(1)
	go waitForShutdownSignal(wg)
}

// EstablishConnections establishes connections with the given peers.
func EstablishConnections() {
	if NodeCtx == nil {
		log.Panic("NodeCtx is not initialized")
	}

	peers := NodeCtx.PeersPids
	NodeCtx.Lock()
	defer NodeCtx.Unlock()

	for peerAddr := range peers {
		peerInfo, err := peer.AddrInfoFromString(peerAddr)
		if err != nil {
			log.Panicf("Error getting multiaddr info: %v", err)
		}
		if err := NodeCtx.Host.Connect(context.Background(), *peerInfo); err != nil {
			log.Panicf("Failed to connect to peer %v: %v", peerAddr, err)
		}
		log.Printf("Successfully connected to %v", peerInfo.Addrs[0])
		stream, err := NodeCtx.Host.NewStream(context.Background(), peerInfo.ID, PROTCOL_ID)
		if err != nil {
			log.Panicf("Error creating new stream with %v: %v", peerAddr, err)
		}
		log.Printf("Successfully created stream with %v", peerInfo.Addrs[0])
		NodeCtx.Streams = append(NodeCtx.Streams, stream)
	}
}

func HandleOutgoingUnsignedMessage(outgoingMessages chan *types.UnsignedMessage, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		msg := <-outgoingMessages
		marshaledMsg, err := MarshalUnsignedMessage(msg)
		if err != nil {
			log.Panicf("Failed to marshal message: %v", err)
		}
		for _, stream := range NodeCtx.Streams {
			n, err := stream.Write(marshaledMsg)
			if err != nil {
				log.Panicf("Failed to write operation to stream: %v", err)
			} else if n != len(marshaledMsg) {
				log.Panicf("Failed to write entire operation to stream: %v", err)
			} else {
				log.Printf("Sent message of type %v to %v", msg.Type, stream.Conn().RemoteMultiaddr())
			}
		}
	}
}

func RemoveStream(stream network.Stream) {
	NodeCtx.Lock()
	defer NodeCtx.Unlock()

	for i, currSteam := range NodeCtx.Streams {
		if stream.ID() == currSteam.ID() {
			NodeCtx.Streams = append(NodeCtx.Streams[:i], NodeCtx.Streams[i+1:]...)
			break
		}
	}
}

func MarshalUnsignedMessage(msg *types.UnsignedMessage) ([]byte, error) {
	return json.Marshal(msg)
}

func UnmarshalUnsignedMessage(data []byte) (*types.UnsignedMessage, error) {
	var msg types.UnsignedMessage
	err := json.Unmarshal(data, &msg)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

func waitForShutdownSignal(wg *sync.WaitGroup) {
	defer wg.Done()
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	log.Println("Received signal, shutting down...")
	if err := NodeCtx.Host.Close(); err != nil {
		panic(err)
	}
	os.Exit(1)
}
