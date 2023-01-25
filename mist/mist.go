package mist_go

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-resty/resty/v2"
)

type MistGo struct {
	Url        string
	restClient *resty.Client
	debug      bool
}

type IMistGoClient interface {
	HealthCheck() error
}

func (o *MistGo) HealthCheck() error {
	_, err := o.restyGet(o.Url, nil)
	if err != nil {
		return err
	}
	return nil
}


// 
func (o *MistGo) IsDebug() bool {
	return o.debug
}

// Resty Methods

func (o *MistGo) restyPost(url string, body interface{}) (*resty.Response, error) {
	resp, err := o.restClient.R().
		SetHeader("Accept", "application/json").
		SetBody(body).
		Post(url)

	if err != nil {
		return nil, err
	}
	if !strings.Contains(resp.Status(), "200") {
		o.debugPrint(fmt.Sprintf("resp -> %v", resp))
		return nil, errors.New(resp.Status())
	}
	return resp, nil
}

func (o *MistGo) restyGet(url string, queryParams map[string]string) (*resty.Response, error) {
	resp, err := o.restClient.R().
		SetQueryParams(queryParams).
		Get(url)
	//
	if err != nil {
		return nil, err
	}
	if !strings.Contains(resp.Status(), "200") {
		o.debugPrint(fmt.Sprintf("resp -> %v", resp))
		return nil, errors.New(resp.Status())
	}
	return resp, nil
}
