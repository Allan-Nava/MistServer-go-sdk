package mist_go

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Allan-Nava/MistServer-go-sdk/lib"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

type service struct {
	mistConfiguration    mistConfiguration
	logger               *zap.SugaredLogger
	restyClient          *resty.Client
	lock                 sync.Mutex
	lastAuthorized       time.Time
	lastAuthorizeRequest *authorizeRequest
}

// ErrUnauthorized is returned when MistServer rejects the login, i.e. the
// reply's authorize.status is not "OK". A "CHALL" reply is retried once with a
// fresh challenge first; "NOACC" (no accounts configured) is not retried.
// The wrapped message includes the status.
var ErrUnauthorized = errors.New("mistserver: unauthorized")

// ErrInvalidRequest is returned, without contacting the server, for a request
// that MistServer would accept but that is almost certainly a mistake, such as
// a nil stream list that would delete every stream.
var ErrInvalidRequest = errors.New("mistserver: invalid request")

const (
	authorizeStatusOK        = "OK"
	authorizeStatusChallenge = "CHALL"
)

// IMistGoClient is a client for the MistServer controller API. Every method
// logs in on its own (challenge-response, cached for a minute) and returns
// ErrUnauthorized if MistServer rejects the credentials.
type IMistGoClient interface {
	// Health logs in and returns the server config, the configured streams
	// and the recent log. Use it as a liveness and credentials check.
	Health() (*Response, error)
	// PostStream creates or updates streams ("addstream"). Zero-valued
	// AddStream fields are not sent, so existing settings are kept.
	PostStream(request PostStreamRequest) (*PostStreamResponse, error)
	// PostStreamRemove deletes a stream by name ("deletestream").
	PostStreamRemove(request PostStreamRemoveRequest) (*PostStreamResponse, error)
	// PostAutoPush adds a rule that pushes a stream to a target every time
	// it comes online ("push_auto_add").
	PostAutoPush(request PostAutoPushRequest) (*Response, error)
	// PostAutoPushRemove removes auto-push rules ("push_auto_remove"); a
	// stream name removes every rule for that stream.
	PostAutoPushRemove(request PostAutoPushRemoveRequest) (*Response, error)
	// PostPushStop stops running pushes by ID ("push_stop"); IDs are the
	// first element of each PostPushList entry.
	PostPushStop(request PostPushStopRequest) (*Response, error)
	// PostPushList lists the pushes running now ("push_list"). Each entry is
	// [id, stream, original target, resolved target, ...].
	PostPushList(request PostPushListRequest) (*PostPushListResponse, error)

	// --- streams and stream tags

	// PostStreams replaces the whole stream list ("streams"); streams not in
	// the map are deleted. A nil map returns ErrInvalidRequest.
	PostStreams(request StreamsRequest) (*Response, error)
	// PostDeleteStreamSource deletes streams and their source files where
	// unambiguous ("deletestreamsource").
	PostDeleteStreamSource(request DeleteStreamSourceRequest) (*DeleteStreamSourceResponse, error)
	// PostNukeStream force-stops a stream and cleans up its memory
	// ("nuke_stream").
	PostNukeStream(request NukeStreamRequest) error
	// PostNoUnconfiguredStreams nukes active streams that are not configured,
	// or run from a different source ("no_unconfigured_streams").
	PostNoUnconfiguredStreams() error
	// PostTagStream adds stream tags ("tag_stream").
	PostTagStream(request TagStreamRequest) error
	// PostUntagStream removes stream tags ("untag_stream").
	PostUntagStream(request UntagStreamRequest) error
	// PostStreamTags returns stream tags ("stream_tags"); no names means all
	// active streams.
	PostStreamTags(request StreamTagsRequest) (*StreamTagsResponse, error)

	// --- statistics and server information

	// PostActiveStreams lists active streams with the requested fields
	// ("active_streams").
	PostActiveStreams(request ActiveStreamsRequest) (*ActiveStreamsResponse, error)
	// PostStatsStreams lists streams with recent statistics ("stats_streams").
	PostStatsStreams(request StatsStreamsRequest) (*StatsStreamsResponse, error)
	// PostClients reports connected clients at a point in time ("clients").
	PostClients(request ClientsRequest) (*ClientsResponse, error)
	// PostTotals reports aggregated statistics over a period ("totals").
	PostTotals(request TotalsRequest) (*TotalsResponse, error)
	// PostProcList lists stream processes ("proc_list").
	PostProcList(request ProcListRequest) (*ProcListResponse, error)
	// PostCapabilities describes installed inputs/outputs and the machine
	// ("capabilities").
	PostCapabilities() (*CapabilitiesResponse, error)

	// --- pushes

	// PostPushStart starts a push now ("push_start").
	PostPushStart(request PushStartRequest) error
	// PostPushAutoList lists auto-push rules by ID ("push_auto_list").
	PostPushAutoList() (*PushAutoListResponse, error)
	// PostPushSettings reads and optionally changes auto-push restart
	// settings ("push_settings").
	PostPushSettings(request PushSettingsRequest) (*PushSettingsResponse, error)

	// --- sessions

	// PostStopSessions disconnects sessions by stream and protocol
	// ("stop_sessions").
	PostStopSessions(request StopSessionsRequest) error
	// PostStopSessID disconnects sessions by session ID ("stop_sessid").
	PostStopSessID(request StopSessIDRequest) error
	// PostStopTag disconnects sessions by session tag ("stop_tag").
	PostStopTag(request StopTagRequest) error
	// PostTagSessID tags sessions ("tag_sessid").
	PostTagSessID(request TagSessIDRequest) error
	// PostInvalidateSessions re-runs USER_NEW for a stream's sessions
	// ("invalidate_sessions").
	PostInvalidateSessions(request InvalidateSessionsRequest) error

	// --- configuration and server control

	// PostConfig changes core server settings ("config").
	PostConfig(request ConfigRequest) (*Response, error)
	// PostAddProtocol enables outputs ("addprotocol").
	PostAddProtocol(request AddProtocolRequest) (*Response, error)
	// PostDeleteProtocol removes outputs that match exactly ("deleteprotocol").
	PostDeleteProtocol(request DeleteProtocolRequest) (*Response, error)
	// PostUpdateProtocol replaces an output configuration ("updateprotocol").
	PostUpdateProtocol(request UpdateProtocolRequest) (*Response, error)
	// PostConfigBackup returns the full configuration ("config_backup").
	PostConfigBackup() (*ConfigBackupResponse, error)
	// PostConfigRestore replaces the full configuration ("config_restore").
	PostConfigRestore(request ConfigRestoreRequest) error
	// PostSave writes the configuration to disk now ("save").
	PostSave() error
	// PostUISettings stores or reads interface settings ("ui_settings").
	PostUISettings(request UISettingsRequest) (*UISettingsResponse, error)
	// PostAPIEndpoint returns the local TCP API URL ("api_endpoint").
	PostAPIEndpoint() (*APIEndpointResponse, error)
	// PostBrowse lists a directory on the server ("browse").
	PostBrowse(request BrowseRequest) (*BrowseResponse, error)
	// PostShutdown shuts the controller down; honoured only over a local
	// connection ("shutdown").
	PostShutdown(request ShutdownRequest) (*ShutdownResponse, error)
	// PostLogout drops the login on the current connection ("logout",
	// MistServer 3.9.1+). The SDK logs in again on the next call.
	PostLogout() error
	// PostClearStatLogs truncates the server log ("clearstatlogs").
	PostClearStatLogs() error
	// PostUpdate returns cached update information ("update").
	PostUpdate() (*UpdateResponse, error)
	// PostAutoUpdate starts a rolling update if one is available
	// ("autoupdate").
	PostAutoUpdate() (*UpdateResponse, error)

	// --- stream keys and JSON Web Keys

	// PostStreamKeys reads or replaces all stream keys ("streamkeys").
	PostStreamKeys(request StreamKeysRequest) (*StreamKeysResponse, error)
	// PostStreamKeyAdd adds or re-points stream keys ("streamkey_add").
	PostStreamKeyAdd(request StreamKeyAddRequest) (*StreamKeyAddResponse, error)
	// PostStreamKeyDel deletes stream keys ("streamkey_del").
	PostStreamKeyDel(request StreamKeyDelRequest) (*StreamKeyDelResponse, error)
	// PostJWKS reads or replaces all JSON Web Keys ("jwks").
	PostJWKS(request JWKSRequest) (*JWKSResponse, error)
	// PostAddJWKS adds JSON Web Keys ("addjwks").
	PostAddJWKS(request AddJWKSRequest) (*AddJWKSResponse, error)
	// PostDeleteJWKS removes JSON Web Keys ("deletejwks").
	PostDeleteJWKS(request DeleteJWKSRequest) (*DeleteJWKSResponse, error)

	// --- custom variables and external writers

	// PostVariableList lists custom variables ("variable_list").
	PostVariableList() (*VariablesResponse, error)
	// PostVariableAdd adds or updates a custom variable ("variable_add").
	PostVariableAdd(request VariableAddRequest) (*VariablesResponse, error)
	// PostVariableRemove removes custom variables ("variable_remove").
	PostVariableRemove(request VariableRemoveRequest) (*VariablesResponse, error)
	// PostExternalWriterList lists external writers ("external_writer_list").
	PostExternalWriterList() (*ExternalWritersResponse, error)
	// PostExternalWriterAdd adds an external writer ("external_writer_add").
	PostExternalWriterAdd(request ExternalWriterAddRequest) (*ExternalWritersResponse, error)
	// PostExternalWriterRemove removes an external writer
	// ("external_writer_remove").
	PostExternalWriterRemove(request ExternalWriterRemoveRequest) (*ExternalWritersResponse, error)

	// --- anything else

	// PostRaw sends arbitrary API members and returns the reply undecoded.
	PostRaw(command map[string]any) (map[string]json.RawMessage, error)
}

// NewService returns a client for the MistServer API at the configured base URL.
// A nil restyClient defaults to resty.New() and a nil logger to a no-op logger.
func NewService(restyClient *resty.Client, logger *zap.SugaredLogger, functions ...func(sc *mistConfiguration)) IMistGoClient {
	if restyClient == nil {
		restyClient = resty.New()
	}
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}

	s := &service{
		restyClient:       restyClient,
		logger:            logger,
		mistConfiguration: defaultMistConfiguration(),
		lastAuthorized:    time.UnixMicro(0),
	}

	for _, fn := range functions {
		fn(&s.mistConfiguration)
	}

	return s
}

func (s *service) resetAuthorization() {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.lastAuthorizeRequest = nil
}

func (s *service) getAuthorization() (*authorizeRequest, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if time.Since(s.lastAuthorized) < time.Minute && s.lastAuthorizeRequest != nil {
		return s.lastAuthorizeRequest, nil
	}

	response, err := postRequest[postAuthorizeRequest, AuthorizationResponse](s, postAuthorizeRequest{})

	if err != nil {
		s.logger.Errorw("challenge request failed", "error", err)
		return nil, err
	}

	password := lib.GenerateMD5(
		lib.GenerateMD5(s.mistConfiguration.Password) + response.Authorize.Challenge,
	)

	s.lastAuthorizeRequest = &authorizeRequest{
		authorizeRequestInner{
			Username: s.mistConfiguration.Username,
			Password: password,
		},
	}
	s.lastAuthorized = time.Now()

	return s.lastAuthorizeRequest, nil
}

func (s *service) Health() (*Response, error) {
	return doAuthorized[Response](s, healthRequest{})
}

func (s *service) PostStream(request PostStreamRequest) (*PostStreamResponse, error) {
	return doAuthorized[PostStreamResponse](s, request)
}

func (s *service) PostStreamRemove(request PostStreamRemoveRequest) (*PostStreamResponse, error) {
	return doAuthorized[PostStreamResponse](s, request)
}

func (s *service) PostAutoPush(request PostAutoPushRequest) (*Response, error) {
	return doAuthorized[Response](s, request)
}

func (s *service) PostPushList(request PostPushListRequest) (*PostPushListResponse, error) {
	return doAuthorized[PostPushListResponse](s, request)
}

func (s *service) PostPushStop(request PostPushStopRequest) (*Response, error) {
	return doAuthorized[Response](s, request)
}

func (s *service) PostAutoPushRemove(request PostAutoPushRemoveRequest) (*Response, error) {
	return doAuthorized[Response](s, request)
}

// doAuthorized attaches the cached authorization to request, sends it, and
// checks authorize.status in the reply: MistServer answers HTTP 200 even when
// the login is rejected. A rejected login is retried once with a fresh
// challenge, since the cached one may have expired server-side.
func doAuthorized[R any, T any, PT interface {
	*T
	setAuthorization(authorizeRequest)
}](s *service, request T) (*R, error) {
	for attempt := 0; ; attempt++ {
		auth, err := s.getAuthorization()
		if err != nil {
			s.logger.Errorw("get authorization failed", "error", err)
			return nil, err
		}
		PT(&request).setAuthorization(*auth)

		response, err := postRequest[T, R](s, request)
		if err != nil {
			return response, err
		}

		status := ""
		if a, ok := any(response).(interface{ authorizeStatus() string }); ok {
			status = a.authorizeStatus()
		}
		if status == "" || status == authorizeStatusOK {
			return response, nil
		}

		s.resetAuthorization()
		// Only CHALL comes with a fresh challenge worth retrying; NOACC (no
		// accounts configured) would fail the same way again.
		if attempt > 0 || status != authorizeStatusChallenge {
			return nil, fmt.Errorf("%w: status %q", ErrUnauthorized, status)
		}
	}
}

// doNoReply is doAuthorized for commands MistServer sends no reply member
// for: only the login status is checked.
func doNoReply[T any, PT interface {
	*T
	setAuthorization(authorizeRequest)
}](s *service, request T) error {
	_, err := doAuthorized[BaseResponse, T, PT](s, request)
	return err
}

func postRequest[T any, R any](s *service, request T) (*R, error) {
	var response R

	r, err := s.restyClient.
		R().
		// MistServer only reads the body as the command when the Content-Type
		// is exactly this; any parameter (e.g. a charset) makes it ignore it.
		SetHeader("Content-Type", "application/json").
		SetBody(request).
		Post(s.mistConfiguration.BaseUrl)

	if err != nil {
		return &response, err
	}

	if r.IsError() {
		return &response, errors.New(r.String())
	}

	err = json.Unmarshal(r.Body(), &response)
	if err != nil {
		s.logger.Warnw("unmarshal response failed", "error", err)
		return &response, err
	}

	return &response, nil
}
