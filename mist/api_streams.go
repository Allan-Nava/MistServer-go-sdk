package mist_go

import "fmt"

// StreamsRequest replaces the whole list of configured streams ("streams").
// Streams missing from the map are deleted, so an empty map deletes every
// stream. A nil map is rejected with ErrInvalidRequest: MistServer would
// ignore it, which is never what the caller meant.
type StreamsRequest struct {
	authorizeRequest
	Streams map[string]AddStream `json:"streams"`
}

type streamsWire struct {
	authorizeRequest
	Streams      map[string]AddStream `json:"streams"`
	StopSessions map[string]string    `json:"stop_sessions,omitempty"`
}

// DeleteStreamSourceRequest deletes streams and, where unambiguous, their
// source files ("deletestreamsource").
type DeleteStreamSourceRequest struct {
	authorizeRequest
	DeleteStreamSource []string `json:"deletestreamsource"`
}

// DeleteStreamSourceResponse holds one status string per requested stream, in
// request order, e.g. "-2: Stream and source file deleted". A negative number
// means the stream was removed from the configuration.
type DeleteStreamSourceResponse struct {
	BaseResponse
	DeleteStreamSource []string `json:"deletestreamsource"`
}

// NukeStreamRequest force-stops a running stream and cleans up its memory
// ("nuke_stream"). Consider PostStopSessions first.
type NukeStreamRequest struct {
	authorizeRequest
	NukeStream string `json:"nuke_stream"`
}

type noUnconfiguredStreamsRequest struct {
	authorizeRequest
	NoUnconfiguredStreams bool `json:"no_unconfigured_streams"`
}

// TagStreamRequest adds stream tags ("tag_stream"): stream name → tags.
// Stream tags are unrelated to session tags.
type TagStreamRequest struct {
	authorizeRequest
	TagStream map[string][]string `json:"tag_stream"`
}

// UntagStreamRequest removes stream tags ("untag_stream"): stream name → tags.
type UntagStreamRequest struct {
	authorizeRequest
	UntagStream map[string][]string `json:"untag_stream"`
}

// StreamTagsRequest looks up stream tags ("stream_tags"). Leave StreamTags
// empty to get the tags of every active stream.
type StreamTagsRequest struct {
	StreamTags []string
}

type streamTagsWire struct {
	authorizeRequest
	StreamTags any `json:"stream_tags"`
}

// StreamTagsResponse maps stream names to their tags; a stream without tags
// maps to nil.
type StreamTagsResponse struct {
	BaseResponse
	StreamTags map[string][]string `json:"stream_tags"`
}

func (s *service) PostStreams(request StreamsRequest) (*Response, error) {
	if request.Streams == nil {
		return nil, fmt.Errorf("%w: nil Streams (MistServer ignores it); pass an empty map to delete every stream", ErrInvalidRequest)
	}
	return doAuthorized[Response](s, streamsWire{
		Streams:      request.Streams,
		StopSessions: stopSessionsFor(request.Streams),
	})
}

func (s *service) PostDeleteStreamSource(request DeleteStreamSourceRequest) (*DeleteStreamSourceResponse, error) {
	return doAuthorized[DeleteStreamSourceResponse](s, request)
}

func (s *service) PostNukeStream(request NukeStreamRequest) error {
	return doNoReply(s, request)
}

func (s *service) PostNoUnconfiguredStreams() error {
	return doNoReply(s, noUnconfiguredStreamsRequest{NoUnconfiguredStreams: true})
}

func (s *service) PostTagStream(request TagStreamRequest) error {
	return doNoReply(s, request)
}

func (s *service) PostUntagStream(request UntagStreamRequest) error {
	return doNoReply(s, request)
}

func (s *service) PostStreamTags(request StreamTagsRequest) (*StreamTagsResponse, error) {
	wire := streamTagsWire{StreamTags: true}
	if len(request.StreamTags) > 0 {
		wire.StreamTags = request.StreamTags
	}
	return doAuthorized[StreamTagsResponse](s, wire)
}
