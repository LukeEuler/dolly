package common

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

// copy from: https://github.com/moul/http2curl

// CurlCommand contains exec.Command compatible slice + helpers
type CurlCommand []string

// append appends a string to the CurlCommand
func (c *CurlCommand) append(newSlice ...string) {
	*c = append(*c, newSlice...)
}

// String returns a ready to copy/paste command
func (c *CurlCommand) String() string {
	return strings.Join(*c, " ")
}

func bashEscape(str string) string {
	return `'` + strings.ReplaceAll(str, `'`, `'\''`) + `'`
}

const redactedValue = "REDACTED"

var sensitiveHeaders = map[string]struct{}{
	"authorization":       {},
	"proxy-authorization": {},
	"cookie":              {},
	"set-cookie":          {},
	"x-api-key":           {},
	"api-key":             {},
	"api_key":             {},
	"x-auth-token":        {},
	"x-access-token":      {},
}

var sensitiveQueryKeys = map[string]struct{}{
	"apikey":       {},
	"api_key":      {},
	"key":          {},
	"token":        {},
	"access_token": {},
	"secret":       {},
	"password":     {},
	"auth":         {},
}

type curlCommandOptions struct {
	redactSensitive bool
}

// CurlCommandOption configures GetCurlCommand.
type CurlCommandOption func(*curlCommandOptions)

// WithCurlSensitiveData makes GetCurlCommand include sensitive values.
//
// The default output is redacted and is safe to place in logs. Use this option
// only when the command is handled in a trusted environment.
func WithCurlSensitiveData() CurlCommandOption {
	return func(options *curlCommandOptions) {
		options.redactSensitive = false
	}
}

func redactURL(u *url.URL) string {
	redacted := *u
	changed := false

	if password, ok := u.User.Password(); ok && password != "" {
		redacted.User = url.UserPassword(u.User.Username(), redactedValue)
		changed = true
	}

	if u.RawQuery != "" {
		query := u.Query()
		for key := range query {
			if _, ok := sensitiveQueryKeys[strings.ToLower(key)]; ok {
				query.Set(key, redactedValue)
				changed = true
			}
		}
		if changed {
			redacted.RawQuery = query.Encode()
		}
	}

	if !changed {
		return u.String()
	}
	return redacted.String()
}

// GetCurlCommand returns a CurlCommand corresponding to an http.Request.
// Sensitive request headers and URL values are redacted by default.
func GetCurlCommand(req *http.Request, opts ...CurlCommandOption) (*CurlCommand, error) {
	if req.URL == nil {
		return nil, fmt.Errorf("getCurlCommand: invalid request, req.URL is nil")
	}
	options := curlCommandOptions{redactSensitive: true}
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}

	command := CurlCommand{}

	command.append("curl")

	schema := req.URL.Scheme
	requestURL := req.URL.String()
	if options.redactSensitive {
		requestURL = redactURL(req.URL)
	}
	if schema == "" {
		schema = "http"
		if req.TLS != nil {
			schema = "https"
		}
		requestURL = schema + "://" + req.Host + req.URL.Path
	}

	if schema == "https" {
		command.append("-k")
	}

	command.append("-X", bashEscape(req.Method))

	if req.Body != nil {
		var buff bytes.Buffer
		_, err := buff.ReadFrom(req.Body)
		if err != nil {
			return nil, fmt.Errorf("getCurlCommand: buffer read from body error: %w", err)
		}
		// reset body for potential re-reads
		req.Body = io.NopCloser(bytes.NewBuffer(buff.Bytes()))
		if len(buff.String()) > 0 {
			bodyEscaped := bashEscape(buff.String())
			command.append("-d", bodyEscaped)
		}
	}

	keys := make([]string, 0, len(req.Header))

	for k := range req.Header {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		value := strings.Join(req.Header[k], " ")
		if options.redactSensitive {
			if _, ok := sensitiveHeaders[strings.ToLower(k)]; ok {
				value = redactedValue
			}
		}
		command.append("-H", bashEscape(fmt.Sprintf("%s: %s", k, value)))
	}

	command.append(bashEscape(requestURL))

	command.append("--compressed")

	return &command, nil
}
