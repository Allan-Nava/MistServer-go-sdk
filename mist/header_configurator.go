package mist_go

import "encoding/base64"

type HeaderConfigurator struct {
	Headers map[string]string
}

func InitHeaderConfigurator() *HeaderConfigurator {
	return &HeaderConfigurator{
		Headers: map[string]string{},
	}
}

func (h *HeaderConfigurator) SetHeader(key string, value string) {
	h.Headers[key] = value
}

func (h *HeaderConfigurator) GetHeader(key string) *string {
	v, ok := h.Headers[key]
	if !ok {
		return nil
	}
	return &v
}

func (h *HeaderConfigurator) GetHeaders() map[string]string {
	return h.Headers
}

func (h *HeaderConfigurator) SetHeaders(headers map[string]string) {
	h.Headers = headers
}

func (h *HeaderConfigurator) DeleteHeader(key string) {
	delete(h.Headers, key)
}

func (h *HeaderConfigurator) DeleteHeaders() {
	h.Headers = make(map[string]string)
}

func (h *HeaderConfigurator) HasHeader(key string) bool {
	_, ok := h.Headers[key]
	return ok
}

func (h *HeaderConfigurator) HasHeaders() bool {
	return len(h.Headers) > 0
}

func (h *HeaderConfigurator) GetHeaderKeys() []string {
	keys := make([]string, len(h.Headers))
	i := 0
	for k := range h.Headers {
		keys[i] = k
		i++
	}
	return keys
}

func (h *HeaderConfigurator) GetHeaderValues() []string {
	values := make([]string, len(h.Headers))
	i := 0
	for _, v := range h.Headers {
		values[i] = v
		i++
	}
	return values
}

func (h *HeaderConfigurator) GetHeaderKeyValuePairs() map[string]string {
	return h.Headers
}

// Authorization Stuff
func (h *HeaderConfigurator) CreateBasicAuthHeader(username string, password string) {
	h.Headers["Authorization"] = "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+password))
}

func (h *HeaderConfigurator) CreateBasicAuthHeaderEncoded(base64EncodedToken string) {
	h.Headers["Authorization"] = "Basic " + base64EncodedToken
}

//
