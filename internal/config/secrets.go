package config

import (
	"fmt"

	"github.com/zalando/go-keyring"
)

const serviceName = "mcp-overwatch"

// SecretKey returns the keyring user key for a given server and env var.
func SecretKey(serverID, envVarName string) string {
	return fmt.Sprintf("%s:%s:%s", serviceName, serverID, envVarName)
}

// SetSecret stores a secret value in the OS keychain.
func SetSecret(serverID, envVarName, value string) error {
	return keyring.Set(serviceName, SecretKey(serverID, envVarName), value)
}

// GetSecret retrieves a secret value from the OS keychain.
func GetSecret(serverID, envVarName string) (string, error) {
	return keyring.Get(serviceName, SecretKey(serverID, envVarName))
}

// DeleteSecret removes a secret from the OS keychain.
func DeleteSecret(serverID, envVarName string) error {
	return keyring.Delete(serviceName, SecretKey(serverID, envVarName))
}
