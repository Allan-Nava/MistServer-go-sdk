package mist_go

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-resty/resty/v2"
	"gopkg.in/validator.v2"
)

type mistGo struct {
	Url        string
	restClient *resty.Client
	debug      bool
}

type IMistGoClient interface {
	//
	HealthCheck() error
	IsDebug() bool
	//debugPrint(data interface{})
	// Auth
	Authenticate(auth AuthorizeCommand) (*ResponseAuth, error)
	// Stream
}

/*func NewMistGoClient(url string, debug bool) *mistGo {
	return &mistGo{
		Url:        url,
		restClient: resty.New(),
		debug:      debug,
	}
}*/

func (o *mistGo) HealthCheck() error {
	_, err := o.restyGet(o.Url, nil)
	if err != nil {
		return err
	}
	return nil
}

func (o *mistGo) IsDebug() bool {
	return o.debug
}

func (o *mistGo) Authenticate(auth AuthorizeCommand) (*ResponseAuth, error) {
	//
	if errs := validator.Validate(auth); errs != nil {
		// values not valid, deal with errors here
		return nil, errs
	}
	//
	rBody := &AuthCommand{
		Authorize: auth,
	}
	b, err := json.Marshal(rBody)
	if err != nil {
		return nil, err
	}
	request := map[string]string{
		"command": string(b),
	}
	resp, err := o.restyGet(COMMAND_URL, request)
	if err != nil {
		return nil, err
	}
	o.debugPrint(resp)
	//
	var obj ResponseAuth
	if err := json.Unmarshal(resp.Body(), &obj); err != nil {
		return nil, err
	}
	return &obj, nil
}

// Resty Methods

func (o *mistGo) restyPost(url string, body interface{}) (*resty.Response, error) {
	resp, err := o.restClient.R().
		SetHeader("Accept", "application/json").
		SetBody(body).
		Post(url)

	if err != nil {
		return nil, err
	}
	if !strings.Contains(resp.Status(), "200") {
		o.debugPrint(resp)
		err = fmt.Errorf("%v", resp)
		return nil, err
	}
	return resp, nil
}

func (o *mistGo) restyGet(url string, queryParams map[string]string) (*resty.Response, error) {
	resp, err := o.restClient.R().
		SetQueryParams(queryParams).
		Get(url)
	//
	if err != nil {
		return nil, err
	}
	if !strings.Contains(resp.Status(), "200") {
		o.debugPrint(resp)
		err = fmt.Errorf("%v", resp)
		return nil, err
	}
	return resp, nil
}
