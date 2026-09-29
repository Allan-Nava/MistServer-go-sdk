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

// ErrUnauthorized is returned when MistServer rejects the credentials, i.e.
// authorize.status is not "OK" even after a fresh challenge.
var ErrUnauthorized = errors.New("mistserver: unauthorized")

const (
	authorizeStatusOK        = "OK"
	authorizeStatusChallenge = "CHALL"
)

type IMistGoClient interface {
	//
	Health() (*Response, error)
	PostStream(request PostStreamRequest) (*PostStreamResponse, error)
	PostStreamRemove(request PostStreamRemoveRequest) (*PostStreamResponse, error)
	PostAutoPush(request PostAutoPushRequest) (*Response, error)
	PostAutoPushRemove(request PostAutoPushRemoveRequest) (*Response, error)
	PostPushStop(request PostPushStopRequest) (*Response, error)
	PostPushList(request PostPushListRequest) (*PostPushListResponse, error)
	//
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
