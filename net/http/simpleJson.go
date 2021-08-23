package http

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

// SimpleJSON ...
type SimpleJSON struct {
	client *http.Client
	url    string
}

// NewSimpleJSON ...
func NewSimpleJSON(url string) *SimpleJSON {
	ts := DefaultTs
	return &SimpleJSON{
		client: &http.Client{
			Timeout:   5 * time.Second,
			Transport: ts,
		},
		url: url,
	}
}

// SetTimeout ste http timeout
func (s *SimpleJSON) SetTimeout(timeout time.Duration) *SimpleJSON {
	s.client.Timeout = timeout
	return s
}

// SetTransport set the Transport
func (s *SimpleJSON) SetTransport(ts http.RoundTripper) {
	s.client.Transport = ts
}

// Get ...
func (s *SimpleJSON) Get(tail string, object interface{}) error {
	resp, err := s.client.Get(s.url + tail)
	if err != nil {
		return errors.WithStack(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return errors.Errorf("http status %d != 200", resp.StatusCode)
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.WithStack(err)
	}
	return errors.WithStack(json.Unmarshal(bodyBytes, object))
}

// Post ...
func (s *SimpleJSON) Post(tail string, in, out interface{}) error {
	marshal, err := json.Marshal(in)
	if err != nil {
		return errors.WithStack(err)
	}
	resp, err := s.client.Post(s.url+tail, "application/json", bytes.NewReader(marshal))
	if err != nil {
		return errors.WithStack(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return errors.Errorf("http status %d != 200", resp.StatusCode)
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.WithStack(err)
	}
	return errors.WithStack(json.Unmarshal(bodyBytes, out))
}
