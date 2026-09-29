package mist_go

import (
	"encoding/json"
	"fmt"
)

// rawWire sends arbitrary command members next to the authorize block.
type rawWire struct {
	authorizeRequest
	command map[string]any
}

func (r rawWire) MarshalJSON() ([]byte, error) {
	merged := make(map[string]any, len(r.command)+1)
	for k, v := range r.command {
		merged[k] = v
	}
	merged["authorize"] = r.Authorize
	return json.Marshal(merged)
}

// rawResponse keeps every reply member undecoded.
type rawResponse map[string]json.RawMessage

func (r rawResponse) authorizeStatus() string {
	var a Authorize
	_ = json.Unmarshal(r["authorize"], &a)
	return a.Status
}

// PostRaw sends any combination of API members in one request, e.g. commands
// this SDK has no method for yet, and returns every member of the reply
// undecoded. Login and the status check are handled as for other methods, so
// command must not contain "authorize".
func (s *service) PostRaw(command map[string]any) (map[string]json.RawMessage, error) {
	if _, ok := command["authorize"]; ok {
		return nil, fmt.Errorf("%w: PostRaw sets \"authorize\" itself", ErrInvalidRequest)
	}
	resp, err := doAuthorized[rawResponse](s, rawWire{command: command})
	if err != nil {
		return nil, err
	}
	return *resp, nil
}
