package mist_go

import "encoding/json"

type postAuthorizeRequest struct{}
type healthRequest struct {
	authorizeRequest
}

// PostStreamRequest creates or updates streams ("addstream"). An update
// replaces the stream's whole configuration, so send every setting it should
// keep (see AddStream.Options).
type PostStreamRequest struct {
	authorizeRequest
	AddStream map[string]AddStream `json:"addstream"`
}

// AddStream is a stream configuration as sent to addstream / streams.
//
// MistServer replaces a stream's configuration with exactly what is sent, so
// any setting left out — including zero-valued fields, which are omitted — is
// removed from the stream. Put every other input or stream setting
// (always_on, processes, tags, input options, ...) in Options; the typed
// fields take precedence over Options keys of the same name.
type AddStream struct {
	Name   string `json:"name"`
	Source string `json:"source"`
	DVR    int    `json:"DVR,omitempty"`
	Debug  int    `json:"debug,omitempty"`
	// StopSessions disconnects the stream's current sessions after saving,
	// so viewers reconnect with the new settings. MistServer only reads
	// stop_sessions as a separate command, so the SDK sends it as one.
	StopSessions bool `json:"-"`
	// Options holds any other stream settings, sent as-is.
	Options map[string]any `json:"-"`
}

// MarshalJSON flattens Options into the stream object.
func (a AddStream) MarshalJSON() ([]byte, error) {
	type typed AddStream // no methods: avoids recursing into MarshalJSON
	b, err := json.Marshal(typed(a))
	if err != nil || len(a.Options) == 0 {
		return b, err
	}
	merged := make(map[string]any, len(a.Options)+4)
	for k, v := range a.Options {
		merged[k] = v
	}
	var fields map[string]any
	if err := json.Unmarshal(b, &fields); err != nil {
		return nil, err
	}
	for k, v := range fields {
		merged[k] = v
	}
	return json.Marshal(merged)
}

// stopSessionsFor returns the top-level stop_sessions member for streams
// with StopSessions set, or nil.
func stopSessionsFor(streams map[string]AddStream) map[string]string {
	var stop map[string]string
	for name, st := range streams {
		if st.StopSessions {
			if stop == nil {
				stop = map[string]string{}
			}
			stop[name] = "" // every protocol
		}
	}
	return stop
}

type PostAutoPushRequest struct {
	authorizeRequest
	PushAutoAdd PushAutoAdd `json:"push_auto_add"`
}

type PushAutoAdd struct {
	Stream string `json:"stream"`
	Target string `json:"target"`
}

// Deprecated: identical to PostAutoPushRemoveRequest and not accepted by any
// method; use PostAutoPushRemoveRequest.
type PostAutoPushStopRequest struct {
	authorizeRequest
	PushAutoRemove string `json:"push_auto_remove"`
}

type PostPushListRequest struct {
	authorizeRequest
	PushList bool `json:"push_list"`
}

type PostPushStopRequest struct {
	authorizeRequest
	PushStop []int `json:"push_stop"`
}

type PostAutoPushRemoveRequest struct {
	authorizeRequest
	PushAutoRemove string `json:"push_auto_remove"`
}

type PostStreamRemoveRequest struct {
	authorizeRequest
	DeleteStream string `json:"deletestream"`
}

type authorizeRequest struct {
	Authorize authorizeRequestInner `json:"authorize"`
}

func (a *authorizeRequest) setAuthorization(auth authorizeRequest) {
	*a = auth
}

type authorizeRequestInner struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
