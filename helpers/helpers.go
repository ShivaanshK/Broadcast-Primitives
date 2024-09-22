package helpers

import (
	"encoding/base64"
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
	Hosts          []string `json:"hosts"`
	HostPublicKeys []string `json:"hostPublicKeys"`
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

// GetPeerPublicKeys reads the config file and returns an array of crypto.PubKey corresponding to each host.
// If the HostPublicKeys array is entirely empty, it reads all the public keys from the peer key files and updates the config.
func GetPeerPublicKeys(filePath string) ([]crypto.PubKey, error) {
	// Open the config file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not open file: %v", err)
	}
	defer file.Close()

	// Read the file's content
	bytes, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("could not read file: %v", err)
	}

	// Unmarshal the JSON data into a HostsConfig struct
	var config HostsConfig
	err = json.Unmarshal(bytes, &config)
	if err != nil {
		return nil, fmt.Errorf("could not parse JSON: %v", err)
	}

	// Ensure the HostPublicKeys array is initialized and has the same length as Hosts
	if len(config.HostPublicKeys) != len(config.Hosts) {
		config.HostPublicKeys = make([]string, len(config.Hosts))
	}

	// Initialize the slice for public keys
	pubKeys := make([]crypto.PubKey, len(config.Hosts))

	// Check if all public keys are missing or empty
	allKeysEmpty := true
	for _, pubKeyStr := range config.HostPublicKeys {
		if pubKeyStr != "" {
			allKeysEmpty = false
			break
		}
	}

	// If all keys are empty, read from key files and update config
	if allKeysEmpty {
		fmt.Println("HostPublicKeys are empty, generating keys from peer key files.")
		for i := range config.Hosts {
			// Read the private key for this host (peer) and extract the public key
			privKey, err := GetKey(i)
			if err != nil {
				return nil, fmt.Errorf("failed to get private key for host %d: %v", i, err)
			}
			pubKey := privKey.GetPublic()

			// Marshal the public key and store it in the config
			pubBytes, err := crypto.MarshalPublicKey(pubKey)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal public key for host %d: %v", i, err)
			}

			// Base64 encode the public key and store it in the config
			config.HostPublicKeys[i] = base64.StdEncoding.EncodeToString(pubBytes)

			pubKeys[i] = pubKey // Add to the pubKeys array
		}

		// Write the updated public keys to the config file
		err = writeUpdatedConfig(filePath, config)
		if err != nil {
			return nil, fmt.Errorf("failed to update config file: %v", err)
		}

	} else {
		// If the public keys exist, just unmarshal them and populate the pubKeys array
		fmt.Println("HostPublicKeys already exist, unmarshaling them.")
		for i, pubKeyStr := range config.HostPublicKeys {
			if pubKeyStr != "" {
				// Verify that the public key is valid Base64 data
				if _, err := base64.StdEncoding.DecodeString(pubKeyStr); err != nil {
					fmt.Printf("Invalid Base64 public key for host %d: %s\n", i, pubKeyStr)
					continue
				}

				pubKeyBytes, err := base64.StdEncoding.DecodeString(pubKeyStr)
				if err != nil {
					return nil, fmt.Errorf("failed to decode Base64 public key for host %d: %v", i, err)
				}

				// Unmarshal the public key from the decoded bytes
				pubKey, err := crypto.UnmarshalPublicKey(pubKeyBytes)
				if err != nil {
					return nil, fmt.Errorf("failed to unmarshal public key for host %d: %v", i, err)
				}

				pubKeys[i] = pubKey
			} else {

			}
		}
	}

	// Return the array of public keys
	return pubKeys, nil
}

// writeUpdatedConfig writes the updated HostsConfig back to the config file.
func writeUpdatedConfig(filePath string, config HostsConfig) error {
	// Marshal the updated config back to JSON
	configData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal updated config: %v", err)
	}

	// Write the updated config to the file
	err = os.WriteFile(filePath, configData, 0600)
	if err != nil {
		return fmt.Errorf("failed to write updated config to file: %v", err)
	}

	return nil
}

// GetKey checks if the key file exists for the given pid.
// If the file doesn't exist, it creates a new keypair, writes it to the file, and returns the private key.
// If the file exists, it reads the private key from the file and returns it.
func GetKey(pid int) (crypto.PrivKey, error) {
	// Create the key file name from the pid
	keyFilePath := GeneratePeerKeyFilePath(pid)

	// Check if the key file already exists
	if _, err := os.Stat(keyFilePath); err == nil {
		// File exists, read and return the key
		priv, err := ReadPrivateKeyFromFile(keyFilePath)
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

// GeneratePeerKeyFilePath creates a file path using the pid.
func GeneratePeerKeyFilePath(pid int) string {
	return fmt.Sprintf("auth_keys/peer%d.key", pid)
}

// ReadPrivateKeyFromFile reads a private key from a file and returns it as a crypto.PrivKey.
func ReadPrivateKeyFromFile(filePath string) (crypto.PrivKey, error) {
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

func CalculateByzantineQuoromSize(numNodes int) (size int) {
	// n = 3f + 1 : solve for f
	faultTolerance := (numNodes - 1) / 3
	// quorom size is 2f + 1
	size = (2 * faultTolerance) + 1
	return
}

func CalculateSingleHonestQuoromSize(numNodes int) (size int) {
	// n = 3f + 1 : solve for f
	faultTolerance := (numNodes - 1) / 3
	return faultTolerance + 1
}
