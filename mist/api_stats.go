package mist_go

import (
	"encoding/json"
	"sort"
)

// ActiveStreamsRequest lists currently active streams with the requested
// fields ("active_streams"). Empty Fields means MistServer's default set;
// empty Streams means all active streams. Names ending in "+" match every
// wildcard stream with that base name.
type ActiveStreamsRequest struct {
	Fields  []string
	Streams []string
}

type activeStreamsQuery struct {
	Fields   []string `json:"fields,omitempty"`
	Streams  []string `json:"streams,omitempty"`
	LongForm bool     `json:"longform"`
}

type activeStreamsWire struct {
	authorizeRequest
	ActiveStreams activeStreamsQuery `json:"active_streams"`
}

// ActiveStreamsResponse maps each active stream to field → value (the
// "longform" reply). It is nil when no stream matches.
type ActiveStreamsResponse struct {
	BaseResponse
	ActiveStreams map[string]map[string]any `json:"active_streams"`
}

// StatsStreamsRequest lists streams that have statistics in the last ~10
// minutes ("stats_streams"). Fields may be "clients" and/or "lastms"; leave it
// empty to get only the names.
type StatsStreamsRequest struct {
	Fields []string
}

type statsStreamsWire struct {
	authorizeRequest
	StatsStreams any `json:"stats_streams"`
}

// StatsStreams is the stats_streams reply. Names is always filled (sorted when
// it comes from a fields reply); Values holds the requested fields per stream,
// in request order, and is nil when no fields were requested.
type StatsStreams struct {
	Names  []string
	Values map[string][]any
}

// UnmarshalJSON accepts both reply forms: an array of names, or an object of
// stream → field values.
func (s *StatsStreams) UnmarshalJSON(b []byte) error {
	var names []string
	if err := json.Unmarshal(b, &names); err == nil {
		s.Names, s.Values = names, nil
		return nil
	}
	var values map[string][]any
	if err := json.Unmarshal(b, &values); err != nil {
		return err
	}
	s.Values = values
	s.Names = make([]string, 0, len(values))
	for name := range values {
		s.Names = append(s.Names, name)
	}
	sort.Strings(s.Names)
	return nil
}

// StatsStreamsResponse is the reply to PostStatsStreams.
type StatsStreamsResponse struct {
	BaseResponse
	StatsStreams StatsStreams `json:"stats_streams"`
}

// ClientsQuery selects the connected clients to report ("clients"). Empty
// slices mean "all"; Time is a unix timestamp, negative for seconds ago, zero
// for now. Asking 20–30 seconds in the past gives the most complete data.
type ClientsQuery struct {
	Streams   []string `json:"streams,omitempty"`
	Protocols []string `json:"protocols,omitempty"`
	Fields    []string `json:"fields,omitempty"`
	Time      int64    `json:"time,omitempty"`
}

// ClientsRequest asks for connected clients.
type ClientsRequest struct {
	authorizeRequest
	Clients ClientsQuery `json:"clients"`
}

// ClientsData holds one row per client, values in Fields order.
type ClientsData struct {
	Time   int64    `json:"time"`
	Fields []string `json:"fields"`
	Data   [][]any  `json:"data"`
}

// ClientsResponse is the reply to PostClients.
type ClientsResponse struct {
	BaseResponse
	Clients ClientsData `json:"clients"`
}

// TotalsQuery selects aggregated statistics over a period ("totals"). Start
// and End are unix timestamps, negative for seconds ago, zero for the widest
// range available.
type TotalsQuery struct {
	Streams   []string `json:"streams,omitempty"`
	Protocols []string `json:"protocols,omitempty"`
	Fields    []string `json:"fields,omitempty"`
	Start     int64    `json:"start,omitempty"`
	End       int64    `json:"end,omitempty"`
}

// TotalsRequest asks for aggregated statistics.
type TotalsRequest struct {
	authorizeRequest
	Totals TotalsQuery `json:"totals"`
}

// TotalsData holds data points in Fields order. Interval describes their
// spacing as [count, seconds] pairs, e.g. [[10,5],[10,1]].
type TotalsData struct {
	Start    int64     `json:"start"`
	End      int64     `json:"end"`
	Fields   []string  `json:"fields"`
	Interval [][]int64 `json:"interval"`
	Data     [][]any   `json:"data"`
}

// TotalsResponse is the reply to PostTotals.
type TotalsResponse struct {
	BaseResponse
	Totals TotalsData `json:"totals"`
}

// ProcListRequest lists stream processes ("proc_list"). Leave ProcList empty
// for all streams.
type ProcListRequest struct {
	authorizeRequest
	ProcList string `json:"proc_list"`
}

// Process is one entry of a proc_list reply. Terminated processes are listed
// once and then forgotten.
type Process struct {
	Source     string `json:"source"`
	Sink       string `json:"sink"`
	Process    string `json:"process"`
	Logs       []any  `json:"logs"`
	Terminated bool   `json:"terminated"`
}

// ProcListResponse maps process IDs to processes.
type ProcListResponse struct {
	BaseResponse
	ProcList map[string]Process `json:"proc_list"`
}

type capabilitiesRequest struct {
	authorizeRequest
	Capabilities bool `json:"capabilities"`
}

// Capabilities describes the server: installed outputs ("connectors") and
// inputs with their parameters, plus CPU, load and memory. Connector and input
// definitions are left raw; see the MistServer capabilities docs for their
// shape.
type Capabilities struct {
	Connectors map[string]json.RawMessage `json:"connectors"`
	Inputs     map[string]json.RawMessage `json:"inputs"`
	CPUUse     int                        `json:"cpu_use"` // tenths of a percent: 500 = 50%
	CPU        []CPUInfo                  `json:"cpu"`
	Load       LoadInfo                   `json:"load"`
	Mem        MemInfo                    `json:"mem"`
	Speed      int                        `json:"speed"`   // MHz summed over all cores
	Threads    int                        `json:"threads"` // threads summed over all CPUs
}

// CPUInfo describes one installed CPU.
type CPUInfo struct {
	Cores   int    `json:"cores"`
	MHz     int    `json:"mhz"`
	Model   string `json:"model"`
	Threads int    `json:"threads"`
}

// LoadInfo is the system load as reported by MistServer.
type LoadInfo struct {
	One     int `json:"one"`
	Five    int `json:"five"`
	Fifteen int `json:"fifteen"`
	Memory  int `json:"memory"`
}

// MemInfo is memory usage in MiB.
type MemInfo struct {
	Total     int `json:"total"`
	Used      int `json:"used"`
	Free      int `json:"free"`
	Cached    int `json:"cached"`
	SwapTotal int `json:"swaptotal"`
	SwapFree  int `json:"swapfree"`
}

// CapabilitiesResponse is the reply to PostCapabilities.
type CapabilitiesResponse struct {
	BaseResponse
	Capabilities Capabilities `json:"capabilities"`
}

func (s *service) PostActiveStreams(request ActiveStreamsRequest) (*ActiveStreamsResponse, error) {
	return doAuthorized[ActiveStreamsResponse](s, activeStreamsWire{ActiveStreams: activeStreamsQuery{
		Fields:   request.Fields,
		Streams:  request.Streams,
		LongForm: true,
	}})
}

func (s *service) PostStatsStreams(request StatsStreamsRequest) (*StatsStreamsResponse, error) {
	wire := statsStreamsWire{StatsStreams: true}
	if len(request.Fields) > 0 {
		wire.StatsStreams = request.Fields
	}
	return doAuthorized[StatsStreamsResponse](s, wire)
}

func (s *service) PostClients(request ClientsRequest) (*ClientsResponse, error) {
	return doAuthorized[ClientsResponse](s, request)
}

func (s *service) PostTotals(request TotalsRequest) (*TotalsResponse, error) {
	return doAuthorized[TotalsResponse](s, request)
}

func (s *service) PostProcList(request ProcListRequest) (*ProcListResponse, error) {
	return doAuthorized[ProcListResponse](s, request)
}

func (s *service) PostCapabilities() (*CapabilitiesResponse, error) {
	return doAuthorized[CapabilitiesResponse](s, capabilitiesRequest{Capabilities: true})
}
