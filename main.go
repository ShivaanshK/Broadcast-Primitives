package main

import (
	"broadcast-primitives/broadcast/cbc/authenticated"
	"broadcast-primitives/broadcast/cbc/echo"
	"broadcast-primitives/broadcast/rbc/bracha"
	"broadcast-primitives/helpers"
	"flag"
	"log"
	"sync"
)

const (
	AUTHENTICATED_BROADCAST int = iota
	ECHO_BROADCAST
	BRACHA_BROADCAST
)

func main() {
	configPath := flag.String("config_path", "./config.json", "Path to config file")
	broadcastPrimitive := flag.Int("type", 0, "Broadcast primitve to use")
	pid := flag.Int("pid", 0, "Process ID")
	flag.Parse()

	// Get broadcast primitive to use and parse config file
	serverAddr, peers, numNodes, err := helpers.GetHostAndMapping(*configPath, *pid)
	if err != nil {
		log.Panicf("Error parsing config file: %v", err)
	}

	var wg sync.WaitGroup

	// Start broadcast depending on flag
	switch *broadcastPrimitive {
	case AUTHENTICATED_BROADCAST:
		authenticated.StartBroadcastSimulation(*pid, serverAddr, peers, numNodes, &wg)
	case ECHO_BROADCAST:
		privKey, _ := helpers.ReadPrivateKeyFromFile(helpers.GeneratePeerKeyFilePath(*pid))
		peerPublicKeys, err := helpers.GetPeerPublicKeys(*configPath)
		if err != nil {
			log.Panicf(err.Error())
		}
		echo.StartBroadcastSimulation(*pid, serverAddr, peers, numNodes, privKey, peerPublicKeys, &wg)
	case BRACHA_BROADCAST:
		bracha.StartBroadcastSimulation(*pid, serverAddr, peers, numNodes, &wg)
	}

	wg.Wait()
}
