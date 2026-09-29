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
