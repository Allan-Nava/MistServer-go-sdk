package mist_go

import (
	"encoding/json"
	"fmt"
)

// Variable is a custom variable usable as $name in targets and triggers. A
// static variable only has Value; a dynamic one has Target (a command or URL)
// re-evaluated every Interval seconds, waiting at most WaitTime seconds.
// Names longer than 31 characters are truncated by MistServer.
type Variable struct {
	Name     string  `json:"name,omitempty"`
	Target   string  `json:"target,omitempty"`
	Interval float64 `json:"interval,omitempty"`
	WaitTime float64 `json:"waitTime,omitempty"`
	Value    string  `json:"value,omitempty"`
}

type variableListRequest struct {
	authorizeRequest
	VariableList bool `json:"variable_list"`
}

// VariableAddRequest adds or updates a custom variable ("variable_add").
type VariableAddRequest struct {
	authorizeRequest
	VariableAdd Variable `json:"variable_add"`
}

// VariableRemoveRequest removes custom variables by name ("variable_remove").
type VariableRemoveRequest struct {
	authorizeRequest
	VariableRemove []string `json:"variable_remove"`
}

// VariablesResponse holds every custom variable, by name.
type VariablesResponse struct {
	BaseResponse
	VariableList map[string]Variable `json:"variable_list"`
}

// ExternalWriter hands pushes to URLs with one of Protocols (e.g. "s3", no
// "://") to CmdLine, a command or URL.
type ExternalWriter struct {
	Name      string
	CmdLine   string
	Protocols []string
}

// MarshalJSON writes the object form that external_writer_add accepts.
func (w ExternalWriter) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Name      string   `json:"name"`
		CmdLine   string   `json:"cmdline"`
		Protocols []string `json:"protocols"`
	}{w.Name, w.CmdLine, w.Protocols})
}

// UnmarshalJSON reads the [name, cmdline, [protocols]] rows that
// external_writer_list returns.
func (w *ExternalWriter) UnmarshalJSON(b []byte) error {
	var row []json.RawMessage
	if err := json.Unmarshal(b, &row); err != nil {
		return err
	}
	if len(row) < 3 {
		return fmt.Errorf("mistserver: external writer row has %d elements, want 3", len(row))
	}
	if err := json.Unmarshal(row[0], &w.Name); err != nil {
		return err
	}
	if err := json.Unmarshal(row[1], &w.CmdLine); err != nil {
		return err
	}
	return json.Unmarshal(row[2], &w.Protocols)
}

type externalWriterListRequest struct {
	authorizeRequest
	ExternalWriterList bool `json:"external_writer_list"`
}

// ExternalWriterAddRequest adds an external writer ("external_writer_add").
type ExternalWriterAddRequest struct {
	authorizeRequest
	ExternalWriterAdd ExternalWriter `json:"external_writer_add"`
}

// ExternalWriterRemoveRequest removes one external writer by name
// ("external_writer_remove").
type ExternalWriterRemoveRequest struct {
	authorizeRequest
	ExternalWriterRemove string `json:"external_writer_remove"`
}

// ExternalWritersResponse holds every external writer.
type ExternalWritersResponse struct {
	BaseResponse
	ExternalWriterList []ExternalWriter `json:"external_writer_list"`
}

func (s *service) PostVariableList() (*VariablesResponse, error) {
	return doAuthorized[VariablesResponse](s, variableListRequest{VariableList: true})
}

func (s *service) PostVariableAdd(request VariableAddRequest) (*VariablesResponse, error) {
	return doAuthorized[VariablesResponse](s, request)
}

func (s *service) PostVariableRemove(request VariableRemoveRequest) (*VariablesResponse, error) {
	return doAuthorized[VariablesResponse](s, request)
}

func (s *service) PostExternalWriterList() (*ExternalWritersResponse, error) {
	return doAuthorized[ExternalWritersResponse](s, externalWriterListRequest{ExternalWriterList: true})
}

func (s *service) PostExternalWriterAdd(request ExternalWriterAddRequest) (*ExternalWritersResponse, error) {
	return doAuthorized[ExternalWritersResponse](s, request)
}

func (s *service) PostExternalWriterRemove(request ExternalWriterRemoveRequest) (*ExternalWritersResponse, error) {
	return doAuthorized[ExternalWritersResponse](s, request)
}
