package logging

import (
	"encoding/json"
	"fmt"
)

// jsonrpcMessage is a minimal struct for parsing JSON-RPC messages.
type jsonrpcMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method,omitempty"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   json.RawMessage `json:"error,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// ParseJSONRPC returns a human-readable summary of a JSON-RPC message.
func ParseJSONRPC(raw []byte) string {
	var msg jsonrpcMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		return "invalid JSON"
	}

	if msg.Method != "" {
		if msg.ID != nil && len(msg.ID) > 0 {
			return fmt.Sprintf("request: %s", msg.Method)
		}
		return fmt.Sprintf("notification: %s", msg.Method)
	}

	if msg.Error != nil && len(msg.Error) > 0 {
		return "error response"
	}

	if msg.Result != nil && len(msg.Result) > 0 {
		return "result response"
	}

	return "unknown message"
}
