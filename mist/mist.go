package mist_go

import (
	"encoding/json"
	"errors"
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

func NewService(restyClient *resty.Client, logger *zap.SugaredLogger, functions ...func(sc *mistConfiguration)) IMistGoClient {
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

func (s *service) getAuthorization() (*authorizeRequest, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if time.Since(s.lastAuthorized) < time.Minute && s.lastAuthorizeRequest != nil {
		return s.lastAuthorizeRequest, nil
	}

	response, err := postRequest[postAuthorizeRequest, AuthorizationResponse](s, postAuthorizeRequest{})

	if err != nil {
		s.logger.Error("post request failed", "error", err)
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
	auth, err := s.getAuthorization()
	if err != nil {
		s.logger.Error("get authorization failed", "error", err)
		return nil, err
	}

	request := healthRequest{}
	request.authorizeRequest = *auth

	return postRequest[healthRequest, Response](s, request)
}

func (s *service) PostStream(request PostStreamRequest) (*PostStreamResponse, error) {
	auth, err := s.getAuthorization()
	if err != nil {
		s.logger.Error("get authorization failed", "error", err)
		return nil, err
	}

	request.authorizeRequest = *auth

	return postRequest[PostStreamRequest, PostStreamResponse](s, request)
}

func (s *service) PostStreamRemove(request PostStreamRemoveRequest) (*PostStreamResponse, error) {
	auth, err := s.getAuthorization()
	if err != nil {
		s.logger.Error("get authorization failed", "error", err)
		return nil, err
	}

	request.authorizeRequest = *auth
	return postRequest[PostStreamRemoveRequest, PostStreamResponse](s, request)
}

func (s *service) PostAutoPush(request PostAutoPushRequest) (*Response, error) {
	auth, err := s.getAuthorization()
	if err != nil {
		s.logger.Error("get authorization failed", "error", err)
		return nil, err
	}

	request.authorizeRequest = *auth
	return postRequest[PostAutoPushRequest, Response](s, request)
}

func (s *service) PostPushList(request PostPushListRequest) (*PostPushListResponse, error) {
	auth, err := s.getAuthorization()
	if err != nil {
		s.logger.Error("get authorization failed", "error", err)
		return nil, err
	}

	request.authorizeRequest = *auth
	return postRequest[PostPushListRequest, PostPushListResponse](s, request)
}

func (s *service) PostPushStop(request PostPushStopRequest) (*Response, error) {
	auth, err := s.getAuthorization()
	if err != nil {
		s.logger.Error("get authorization failed", "error", err)
		return nil, err
	}

	request.authorizeRequest = *auth
	return postRequest[PostPushStopRequest, Response](s, request)
}

func (s *service) PostAutoPushRemove(request PostAutoPushRemoveRequest) (*Response, error) {
	auth, err := s.getAuthorization()
	if err != nil {
		s.logger.Error("get authorization failed", "error", err)
		return nil, err
	}

	request.authorizeRequest = *auth
	return postRequest[PostAutoPushRemoveRequest, Response](s, request)
}

func postRequest[T any, R any](s *service, request T) (*R, error) {
	var response R

	r, err := s.restyClient.
		R().
		SetResult(&response).
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
		s.logger.Warn("unmarshal response failed", "error", err)
		return &response, err
	}

	return &response, nil
}
