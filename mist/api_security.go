package mist_go

import (
	"encoding/json"
	"errors"
)

// StreamKeysRequest reads or replaces all stream keys ("streamkeys",
// MistServer 3.7+). A nil map only reads; any other map, even empty, replaces
// the whole list.
type StreamKeysRequest struct {
	StreamKeys map[string]string
}

type streamKeysWire struct {
	authorizeRequest
	StreamKeys any `json:"streamkeys"`
}

// StreamKeysResponse maps stream keys to the stream they give push access to.
type StreamKeysResponse struct {
	BaseResponse
	StreamKeys map[string]string `json:"streamkeys"`
}

// StreamKeyAddRequest adds or re-points stream keys ("streamkey_add"): key →
// stream name.
type StreamKeyAddRequest struct {
	authorizeRequest
	StreamKeyAdd map[string]string `json:"streamkey_add"`
}

// StreamKeyAddResult lists keys whose configuration actually changed.
type StreamKeyAddResult struct {
	Added []string `json:"added"`
	Error string   `json:"error,omitempty"`
}

// StreamKeyAddResponse is the reply to PostStreamKeyAdd.
type StreamKeyAddResponse struct {
	BaseResponse
	StreamKeyAdd StreamKeyAddResult `json:"streamkey_add"`
}

// StreamKeyDelRequest deletes stream keys ("streamkey_del").
type StreamKeyDelRequest struct {
	authorizeRequest
	StreamKeyDel []string `json:"streamkey_del"`
}

// StreamKeyDelResult lists keys that existed and were deleted.
type StreamKeyDelResult struct {
	Deleted []string `json:"deleted"`
	Error   string   `json:"error,omitempty"`
}

// StreamKeyDelResponse is the reply to PostStreamKeyDel.
type StreamKeyDelResponse struct {
	BaseResponse
	StreamKeyDel StreamKeyDelResult `json:"streamkey_del"`
}

// JWK is one JSON Web Key entry: Key is either a JWK object (RFC 7517; give it
// a "kid" so it can be matched later) or the URL of a JWKS endpoint.
// Permissions nil means MistServer's defaults.
type JWK struct {
	Key         any
	Permissions *JWKPermissions
}

// JWKPermissions says what a key may be used for. Stream is "*", a stream
// name, or a list of stream names. All three booleans are always sent, so set
// each one you want true.
type JWKPermissions struct {
	Input  bool `json:"input"`
	Output bool `json:"output"`
	Admin  bool `json:"admin"`
	Stream any  `json:"stream,omitempty"`
}

// MarshalJSON writes the [key, permissions] pair MistServer expects, or the
// bare key when there are no permissions.
func (j JWK) MarshalJSON() ([]byte, error) {
	if j.Key == nil {
		return nil, errors.New("mistserver: JWK.Key is required")
	}
	if j.Permissions == nil {
		return json.Marshal(j.Key)
	}
	return json.Marshal([]any{j.Key, j.Permissions})
}

// JWKSRequest reads or replaces all JWKs ("jwks"). A nil slice only reads;
// anything else replaces the whole set.
type JWKSRequest struct {
	JWKS []JWK
}

type jwksWire struct {
	authorizeRequest
	JWKS any `json:"jwks"`
}

// JWKSResponse holds the configured keys. Reads return each one as a
// [key, permissions] pair, with MistServer's default permissions filled in
// (URLs included).
type JWKSResponse struct {
	BaseResponse
	JWKS []json.RawMessage `json:"jwks"`
}

// AddJWKSRequest adds keys without removing others ("addjwks"); a key with
// the same kid replaces the old one.
type AddJWKSRequest struct {
	authorizeRequest
	AddJWKS []JWK `json:"addjwks"`
}

// AddJWKSResponse holds the keys that were added.
type AddJWKSResponse struct {
	BaseResponse
	AddJWKS []json.RawMessage `json:"addjwks"`
}

// DeleteJWKSRequest removes keys ("deletejwks"), matched by "kid" or by exact
// key, e.g. JWK{Key: map[string]any{"kid": "k1"}}.
type DeleteJWKSRequest struct {
	authorizeRequest
	DeleteJWKS []JWK `json:"deletejwks"`
}

// DeleteJWKSResponse holds the removed keys, or null if none matched.
type DeleteJWKSResponse struct {
	BaseResponse
	DeleteJWKS json.RawMessage `json:"deletejwks"`
}

func (s *service) PostStreamKeys(request StreamKeysRequest) (*StreamKeysResponse, error) {
	wire := streamKeysWire{StreamKeys: true}
	if request.StreamKeys != nil {
		wire.StreamKeys = request.StreamKeys
	}
	return doAuthorized[StreamKeysResponse](s, wire)
}

func (s *service) PostStreamKeyAdd(request StreamKeyAddRequest) (*StreamKeyAddResponse, error) {
	return doAuthorized[StreamKeyAddResponse](s, request)
}

func (s *service) PostStreamKeyDel(request StreamKeyDelRequest) (*StreamKeyDelResponse, error) {
	return doAuthorized[StreamKeyDelResponse](s, request)
}

func (s *service) PostJWKS(request JWKSRequest) (*JWKSResponse, error) {
	wire := jwksWire{JWKS: true}
	if request.JWKS != nil {
		wire.JWKS = request.JWKS
	}
	return doAuthorized[JWKSResponse](s, wire)
}

func (s *service) PostAddJWKS(request AddJWKSRequest) (*AddJWKSResponse, error) {
	return doAuthorized[AddJWKSResponse](s, request)
}

func (s *service) PostDeleteJWKS(request DeleteJWKSRequest) (*DeleteJWKSResponse, error) {
	return doAuthorized[DeleteJWKSResponse](s, request)
}
