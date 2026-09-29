package mist_go

// PushStart is a one-off push of Stream to Target. Stream may be "name",
// "name+" (every wildcard stream of name), "name+wild" or "#tag".
type PushStart struct {
	Stream string `json:"stream"`
	Target string `json:"target"`
}

// PushStartRequest starts a push right now ("push_start").
type PushStartRequest struct {
	authorizeRequest
	PushStart PushStart `json:"push_start"`
}

type pushAutoListRequest struct {
	authorizeRequest
	PushAutoList bool `json:"push_auto_list"`
}

// AutoPush is one automatic push rule (MistServer 3.6+). StartRule and
// EndRule are [variable, operator, value] triples; see the push_auto_list docs
// for the operator codes.
type AutoPush struct {
	Stream       string `json:"stream"`
	Target       string `json:"target"`
	Notes        string `json:"x-LSP-notes,omitempty"`
	ScheduleTime int64  `json:"scheduletime,omitempty"`
	CompleteTime int64  `json:"completetime,omitempty"`
	StartRule    []any  `json:"start_rule,omitempty"`
	EndRule      []any  `json:"end_rule,omitempty"`
}

// PushAutoListResponse maps auto-push IDs (usable with PostAutoPushRemove) to
// rules.
type PushAutoListResponse struct {
	BaseResponse
	AutoPush map[string]AutoPush `json:"auto_push"`
}

// PushSettingsUpdate changes auto-push restart behaviour; nil fields are left
// unchanged, so the zero value only reads the settings.
type PushSettingsUpdate struct {
	Wait     *int `json:"wait,omitempty"`     // seconds before restarting a failed push; 0 = never
	MaxSpeed *int `json:"maxspeed,omitempty"` // max auto-push restarts per second; 0 = unlimited
}

// PushSettingsRequest reads and optionally changes push settings
// ("push_settings").
type PushSettingsRequest struct {
	authorizeRequest
	PushSettings PushSettingsUpdate `json:"push_settings"`
}

// PushSettings are the current auto-push settings.
type PushSettings struct {
	Wait     int `json:"wait"`
	MaxSpeed int `json:"maxspeed"`
}

// PushSettingsResponse is the reply to PostPushSettings, after any change.
type PushSettingsResponse struct {
	BaseResponse
	PushSettings PushSettings `json:"push_settings"`
}

// StopSessionsRequest disconnects sessions ("stop_sessions"): stream name →
// protocol. An empty protocol matches every protocol and an empty stream name
// every stream, so {"live": ""} stops all sessions of "live" and
// {"": "RTMP"} all RTMP sessions. "INPUT" and "OUTPUT" are special protocols.
//
// This is the object form on purpose: MistServer reads the documented array
// form as protocol names, not stream names.
type StopSessionsRequest struct {
	authorizeRequest
	StopSessions map[string]string `json:"stop_sessions"`
}

// StopSessIDRequest disconnects sessions by session ID ("stop_sessid"), as
// seen in the USER_NEW trigger payload.
type StopSessIDRequest struct {
	authorizeRequest
	StopSessID []string `json:"stop_sessid"`
}

// StopTagRequest disconnects every session carrying one of the session tags
// ("stop_tag").
type StopTagRequest struct {
	authorizeRequest
	StopTag []string `json:"stop_tag"`
}

// TagSessIDRequest tags sessions ("tag_sessid"): session ID → tag.
type TagSessIDRequest struct {
	authorizeRequest
	TagSessID map[string]string `json:"tag_sessid"`
}

// InvalidateSessionsRequest re-runs the USER_NEW trigger for the sessions of
// the given streams ("invalidate_sessions").
type InvalidateSessionsRequest struct {
	authorizeRequest
	InvalidateSessions []string `json:"invalidate_sessions"`
}

func (s *service) PostPushStart(request PushStartRequest) error {
	return doNoReply(s, request)
}

func (s *service) PostPushAutoList() (*PushAutoListResponse, error) {
	return doAuthorized[PushAutoListResponse](s, pushAutoListRequest{PushAutoList: true})
}

func (s *service) PostPushSettings(request PushSettingsRequest) (*PushSettingsResponse, error) {
	return doAuthorized[PushSettingsResponse](s, request)
}

func (s *service) PostStopSessions(request StopSessionsRequest) error {
	return doNoReply(s, request)
}

func (s *service) PostStopSessID(request StopSessIDRequest) error {
	return doNoReply(s, request)
}

func (s *service) PostStopTag(request StopTagRequest) error {
	return doNoReply(s, request)
}

func (s *service) PostTagSessID(request TagSessIDRequest) error {
	return doNoReply(s, request)
}

func (s *service) PostInvalidateSessions(request InvalidateSessionsRequest) error {
	return doNoReply(s, request)
}
