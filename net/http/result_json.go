package http

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/LukeEuler/dolly/common"
	"github.com/LukeEuler/dolly/log"
)

type ResultJSON struct {
	client *http.Client
	url    string
}

func NewResultJSON(url string) *ResultJSON {
	ts := DefaultTS
	return &ResultJSON{
		client: &http.Client{
			Timeout:   5 * time.Second,
			Transport: ts,
		},
		url: url,
	}
}

type result struct {
	Result json.RawMessage `json:"result"`
	Error  string          `json:"error"`
}

func (r *result) hasError() bool {
	return r.Error != ""
}

func (r *result) GetError() error {
	return errors.New(r.Error)
}

func (r *ResultJSON) SetTimeout(timeout time.Duration) *ResultJSON {
	r.client.Timeout = timeout
	return r
}

func (r ResultJSON) SetTransport(ts http.RoundTripper) {
	r.client.Transport = ts
}

func (r *ResultJSON) Get(tail string, object any) error {
	req, err := http.NewRequest("GET", r.url+tail, nil)
	if err != nil {
		return errors.WithStack(err)
	}

	command, _ := common.GetCurlCommand(req)
	log.Entry.WithField("tags", "request").Debug(command)

	resp, err := r.client.Do(req)
	if err != nil {
		return errors.WithStack(err)
	}
	return handleResponse(resp, object)
}

func (r *ResultJSON) Post(tail string, in, out any) error {
	marshal, err := json.Marshal(in)
	if err != nil {
		return errors.WithStack(err)
	}

	req, err := http.NewRequest("POST", r.url+tail, bytes.NewReader(marshal))
	if err != nil {
		return errors.WithStack(err)
	}
	req.Header.Set("Content-Type", "application/json")

	command, _ := common.GetCurlCommand(req)
	log.Entry.WithField("tags", "request").Debug(command)

	resp, err := r.client.Do(req)
	if err != nil {
		return errors.WithStack(err)
	}
	return handleResponse(resp, out)
}

func handleResponse(resp *http.Response, out any) error {
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
		return errors.Errorf("http status %d != 200\n%s", resp.StatusCode, bodyStr)
	}
	return unmarshalBody(bodyBytes, out)
}

func unmarshalBody(body []byte, object any) error {
	content, err := unmarshalResult(body)
	if err != nil {
		return err
	}
	return errors.WithStack(json.Unmarshal(content, object))
}

func unmarshalResult(raw []byte) (content []byte, err error) {
	resMsg := new(result)
	err = json.Unmarshal(raw, resMsg)
	if err = errors.WithStack(err); err != nil {
		return
	}
	if resMsg.hasError() {
		err = errors.WithStack(resMsg.GetError())
		return
	}
	content = resMsg.Result
	return
}
