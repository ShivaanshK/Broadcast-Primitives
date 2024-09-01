package helpers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"golang.org/x/exp/rand"
)

// HostsConfig represents the structure of the JSON file.
type HostsConfig struct {
	Hosts []string `json:"hosts"`
}

// GetHostAndMapping reads the config file, parses the hosts, and returns the host at the given PID,
// a mapping of the remaining hosts, and the total number of hosts.
func GetHostAndMapping(filePath string, pid int) (string, map[string]int, int, error) {
	// Open the JSON file.
	file, err := os.Open(filePath)
	if err != nil {
		return "", nil, 0, fmt.Errorf("could not open file: %v", err)
	}
	defer file.Close()

	// Read the file's content.
	bytes, err := io.ReadAll(file)
	if err != nil {
		return "", nil, 0, fmt.Errorf("could not read file: %v", err)
	}

	// Unmarshal the JSON data into a HostsConfig struct.
	var config HostsConfig
	err = json.Unmarshal(bytes, &config)
	if err != nil {
		return "", nil, 0, fmt.Errorf("could not parse JSON: %v", err)
	}

	// Check if the PID is within the valid range.
	if pid < 0 || pid >= len(config.Hosts) {
		return "", nil, 0, fmt.Errorf("PID %d is out of range", pid)
	}

	// The host at the given index (PID).
	selectedHost := config.Hosts[pid]

	// Map to store the remaining hosts and their indices.
	hostMapping := make(map[string]int)

	for i, host := range config.Hosts {
		if i != pid {
			hostMapping[host] = i
		}
	}

	// Return the selected host, the host mapping, the number of hosts, and no error.
	return selectedHost, hostMapping, len(config.Hosts), nil
}

// GetKey checks if the key file exists for the given pid.
// If the file doesn't exist, it creates a new keypair, writes it to the file, and returns the private key.
// If the file exists, it reads the private key from the file and returns it.
func GetKey(pid int) (crypto.PrivKey, error) {
	// Create the key file name from the pid
	keyFilePath := generatePeerKeyFilePath(pid)

	// Check if the key file already exists
	if _, err := os.Stat(keyFilePath); err == nil {
		// File exists, read and return the key
		priv, err := readPrivateKeyFromFile(keyFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read key from file: %v", err)
		}
		log.Printf("Key file found at %s", keyFilePath)
		return priv, nil
	} else if !os.IsNotExist(err) {
		// Some other error occurred while checking the file
		return nil, err
	}

	// File doesn't exist, create a new keypair
	priv, _, err := crypto.GenerateKeyPair(crypto.Ed25519, -1)
	if err != nil {
		return nil, err
	}

	// Marshal the private key using the provided function
	privBytes, err := crypto.MarshalPrivateKey(priv)
	if err != nil {
		return nil, err
	}

	// Write the private key to the file
	err = os.WriteFile(keyFilePath, privBytes, 0600)
	if err != nil {
		return nil, err
	}

	log.Printf("New key file created at %s", keyFilePath)
	return priv, nil
}

// generatePeerKeyFilePath creates a file path using the pid.
func generatePeerKeyFilePath(pid int) string {
	return fmt.Sprintf("peer%d.key", pid)
}

// readPrivateKeyFromFile reads a private key from a file and returns it as a crypto.PrivKey.
func readPrivateKeyFromFile(filePath string) (crypto.PrivKey, error) {
	// Read the file contents
	keyData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Unmarshal the private key using the provided function
	priv, err := crypto.UnmarshalPrivateKey(keyData)
	if err != nil {
		return nil, err
	}

	return priv, nil
}

// ParseMultiaddress takes a multiaddress string and returns the part before the 5th "/".
func ParseMultiaddress(addr string) (string, error) {
	// Split the address by "/"
	parts := strings.Split(addr, "/")

	// Ensure there are enough parts to extract up to the 5th "/"
	if len(parts) < 5 {
		return "", fmt.Errorf("invalid multiaddress: %s", addr)
	}

	// Join the first five parts back together with "/"
	parsedAddr := strings.Join(parts[:5], "/")

	return parsedAddr, nil
}

// RandomString generates a random string of a given length.
// The charset defines the set of characters that can be included in the string.
func RandomMessage(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	seededRand := rand.New(rand.NewSource(uint64(time.Now().UnixNano())))

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func CalculateQuoromSize(numNodes int) (size int) {
	// n = 3f + 1 : solve for f
	faultTolerance := (numNodes - 1) / 3
	// quorom size is 2f + 1
	size = (2 * faultTolerance) + 1
	return
}
