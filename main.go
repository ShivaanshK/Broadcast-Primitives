package main

import (
	authbc "broadcast-primitives/broadcast/cbc/authbc"
	"broadcast-primitives/helpers"
	"flag"
	"log"
	"sync"
)

const (
	AUTHENTICATED_BROADCAST int = iota
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
		authbc.StartBroadcastSimulation(*pid, serverAddr, peers, numNodes, &wg)
	}

	wg.Wait()
}
