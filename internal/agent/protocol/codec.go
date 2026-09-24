package protocol

import (
	"encoding/json"
	"fmt"
)

// Wire is the JSON frame on the WebSocket.
type Wire struct {
	Type      Type            `json:"type"`
	RequestID string          `json:"request_id,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

func Encode(msgType Type, requestID string, payload any) ([]byte, error) {
	var raw json.RawMessage
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		raw = b
	}
	return json.Marshal(Wire{Type: msgType, RequestID: requestID, Payload: raw})
}

func Decode(data []byte) (Wire, error) {
	var w Wire
	if err := json.Unmarshal(data, &w); err != nil {
		return Wire{}, fmt.Errorf("decode wire: %w", err)
	}
	if w.Type == "" {
		return Wire{}, fmt.Errorf("missing type")
	}
	return w, nil
}

func DecodePayload[T any](w Wire) (T, error) {
	var v T
	if len(w.Payload) == 0 {
		return v, nil
	}
	if err := json.Unmarshal(w.Payload, &v); err != nil {
		return v, fmt.Errorf("decode payload %s: %w", w.Type, err)
	}
	return v, nil
}
