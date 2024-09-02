package main

import (
	authcbc "broadcast-primitives/broadcast/cbc"
	"broadcast-primitives/helpers"
	"broadcast-primitives/types"
	"flag"
	"log"
	"sync"
)

func main() {
	configPath := flag.String("config_path", "./config.json", "Path to config file")
	broadcastPrimitive := flag.Int("type", 0, "Broadcast primitve to use")
	pid := flag.Int("pid", 0, "Process ID")
	flag.Parse()

	// Get broadcast primitive to use and parse config file
	broadcastPrimitiveToUse := types.BroadcastPrimitive(*broadcastPrimitive)
	serverAddr, peers, numNodes, err := helpers.GetHostAndMapping(*configPath, *pid)
	if err != nil {
		log.Panicf("Error parsing config file: %v", err)
	}

	var wg sync.WaitGroup

	// Start broadcast depending on flag
	switch broadcastPrimitiveToUse {
	case types.AUTH_CBC:
		authcbc.StartBroadcast(*pid, serverAddr, peers, numNodes, &wg)
	}

	wg.Wait()
}
