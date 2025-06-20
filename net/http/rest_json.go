package http

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/LukeEuler/dolly/common"
	"github.com/LukeEuler/dolly/log"
)

type RestJSON struct {
	client      *http.Client
	url         string
	showRequest bool // for debug
}

func NewRestJSON(url string) *RestJSON {
	ts := DefaultTS
	return &RestJSON{
		client: &http.Client{
			Timeout:   5 * time.Second,
			Transport: ts,
		},
		url: url,
	}
}

func (s *RestJSON) SetTimeout(timeout time.Duration) *RestJSON {
	s.client.Timeout = timeout
	return s
}

func (s *RestJSON) SetTransport(ts http.RoundTripper) {
	s.client.Transport = ts
}

func (s *RestJSON) ShowRequest(value bool) *RestJSON {
	s.showRequest = value
	return s
}

type RestJSONParam struct {
	Name  string
	Value string
}

func (s *RestJSON) Get(tail string, params []RestJSONParam, object any) error {
	//new request
	req, err := http.NewRequest("GET", s.url+tail, nil)
	if err != nil {
		return errors.New("new request is fail ")
	}
	//add params
	q := req.URL.Query()
	if params != nil {
		for _, item := range params {
			q.Add(item.Name, item.Value)
		}
		req.URL.RawQuery = q.Encode()
	}
	if s.showRequest {
		fmt.Println(s.url + tail + "?" + req.URL.RawQuery)
	}

	command, _ := common.GetCurlCommand(req)
	log.Entry.WithField("tags", "request").Debug(command)

	resp, err := s.client.Do(req)
	if err != nil {
		return errors.WithStack(err)
	}
	// nolint
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.WithStack(err)
	}
	if resp.StatusCode != 200 {
		bodyStr := string(bodyBytes)
		if len(bodyBytes) > 500 {
			bodyStr = string(bodyBytes[:150])
			bodyStr = strings.ToValidUTF8(bodyStr, "") + "   凸(゜皿゜メ)"
		}
		return errors.Errorf("http status %d, body: %s", resp.StatusCode, bodyStr)
	}
	return errors.WithStack(json.Unmarshal(bodyBytes, object))
}
