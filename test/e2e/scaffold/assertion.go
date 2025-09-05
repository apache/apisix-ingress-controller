// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package scaffold

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gavv/httpexpect/v2"
	. "github.com/onsi/gomega" //nolint:staticcheck
	"github.com/onsi/gomega/types"
)

const (
	DefaultTimeout  = 30 * time.Second
	DefaultInterval = 2 * time.Second
)

type ResponseCheckFunc func(*HTTPResponse) error

type HTTPResponse struct {
	*http.Response

	Body string
}

type BasicAuth struct {
	Username string
	Password string
}

type RequestAssert struct {
	Client    *httpexpect.Expect
	Method    string
	Path      string
	Host      string
	Query     map[string]any
	Headers   map[string]string
	Body      []byte
	BasicAuth *BasicAuth

	Timeout  time.Duration
	Interval time.Duration

	Check  ResponseCheckFunc
	Checks []ResponseCheckFunc
}

func (c *RequestAssert) request(method, path string, body []byte) *httpexpect.Request {
	switch strings.ToUpper(method) {
	case "GET":
		return c.Client.GET(path)
	case "POST":
		return c.Client.POST(path).WithBytes(body)
	case "PUT":
		return c.Client.PUT(path).WithBytes(body)
	case "DELETE":
		return c.Client.DELETE(path)
	case "PATCH":
		return c.Client.PATCH(path).WithBytes(body)
	default:
		panic("unsupported method: " + method)
	}
}

func (c *RequestAssert) WithCheck(check ResponseCheckFunc) *RequestAssert {
	c.Checks = append(c.Checks, check)
	return c
}

func (c *RequestAssert) WithChecks(checks ...ResponseCheckFunc) *RequestAssert {
	c.Checks = append(c.Checks, checks...)
	return c
}

func (c *RequestAssert) SetChecks(checks ...ResponseCheckFunc) *RequestAssert {
	c.Checks = checks
	return c
}

func WithExpectedStatus(status int) ResponseCheckFunc {
	return func(resp *HTTPResponse) error {
		if resp.StatusCode != status {
			return fmt.Errorf("expected %d, but got %d", status, resp.StatusCode)
		}
		return nil
	}
}

func WithExpectedBodyContains(expectedBodyList ...string) ResponseCheckFunc {
	return func(resp *HTTPResponse) error {
		for _, body := range expectedBodyList {
			if !strings.Contains(resp.Body, body) {
				return fmt.Errorf("expected body to contain %q, but got %q", body, resp.Body)
			}
		}
		return nil
	}
}

func WithExpectedBodyNotContains(unexpectedBodyList ...string) ResponseCheckFunc {
	return func(resp *HTTPResponse) error {
		for _, unexpectedBody := range unexpectedBodyList {
			if strings.Contains(resp.Body, unexpectedBody) {
				return fmt.Errorf("expected body not to contain %q, but got %q", unexpectedBody, resp.Body)
			}
		}
		return nil
	}
}

func WithExpectedHeader(key, value string) ResponseCheckFunc {
	return func(resp *HTTPResponse) error {
		if resp.Header.Get(key) != value {
			return fmt.Errorf("expected header %q to be %q, but got %q",
				key, value, resp.Header.Get(key))
		}
		return nil
	}
}

func WithExpectedHeaders(expectedHeaders map[string]string) ResponseCheckFunc {
	return func(resp *HTTPResponse) error {
		for key, expectedValue := range expectedHeaders {
			actualValue := resp.Header.Get(key)
			if actualValue != expectedValue {
				return fmt.Errorf("expected header %q to be %q, but got %q",
					key, expectedValue, actualValue)
			}
		}
		return nil
	}
}

func WithExpectedNotHeader(key string) ResponseCheckFunc {
	return func(resp *HTTPResponse) error {
		if resp.Header.Get(key) != "" {
			return fmt.Errorf("expected header %q to be empty, but got %q",
				key, resp.Header.Get(key))
		}
		return nil
	}
}

func WithExpectedNotHeaders(unexpectedHeaders []string) ResponseCheckFunc {
	return func(resp *HTTPResponse) error {
		for _, key := range unexpectedHeaders {
			if resp.Header.Get(key) != "" {
				return fmt.Errorf("expected header %q to be empty, but got %q",
					key, resp.Header.Get(key))
			}
		}
		return nil
	}
}

func (s *Scaffold) RequestAssert(r *RequestAssert) bool {
	if r.Client == nil {
		r.Client = s.NewAPISIXClient()
	}
	if r.Method == "" {
		if len(r.Body) > 0 {
			r.Method = "POST"
		} else {
			r.Method = "GET"
		}
	}
	if r.Timeout == 0 {
		r.Timeout = DefaultTimeout
	}
	if r.Interval == 0 {
		r.Interval = DefaultInterval
	}
	if r.Check == nil && len(r.Checks) == 0 {
		r.Check = WithExpectedStatus(http.StatusOK)
	} else if r.Check != nil {
		r.Checks = append(r.Checks, r.Check)
	}

	return EventuallyWithOffset(1, func() error {
		req := r.request(r.Method, r.Path, r.Body)
		if len(r.Headers) > 0 {
			req = req.WithHeaders(r.Headers)
		}
		if r.Host != "" {
			req = req.WithHost(r.Host)
		}
		if len(r.Query) > 0 {
			for key, value := range r.Query {
				req = req.WithQuery(key, value)
			}
		}
		if r.BasicAuth != nil {
			req = req.WithBasicAuth(r.BasicAuth.Username, r.BasicAuth.Password)
		}
		expResp := req.Expect()

		resp := &HTTPResponse{
			Response: expResp.Raw(),
			Body:     expResp.Body().Raw(),
		}

		for _, check := range r.Checks {
			if err := check(resp); err != nil {
				return fmt.Errorf("response check failed: %w", err)
			}
		}
		return nil
	}).WithTimeout(r.Timeout).ProbeEvery(r.Interval).Should(Succeed())
}

// RetryAssertion provides a reusable Eventually-based assertion
type RetryAssertion struct {
	timeout  time.Duration
	interval time.Duration

	args        []any
	actualOrCtx any
}

// NewRetryAssertion creates a RetryAssertion with defaults
func (s *Scaffold) RetryAssertion(actualOrCtx any, args ...any) *RetryAssertion {
	return &RetryAssertion{
		timeout:     DefaultTimeout,
		interval:    DefaultInterval,
		args:        args,
		actualOrCtx: actualOrCtx,
	}
}

// WithTimeout sets the timeout
func (r *RetryAssertion) WithTimeout(timeout time.Duration) *RetryAssertion {
	r.timeout = timeout
	return r
}

// WithInterval sets the polling interval
func (r *RetryAssertion) WithInterval(interval time.Duration) *RetryAssertion {
	r.interval = interval
	return r
}

// Should runs the Eventually assertion with the given matcher
func (r *RetryAssertion) Should(matcher types.GomegaMatcher, optionalDescription ...any) bool {
	return EventuallyWithOffset(1, r.actualOrCtx, r.args...).
		WithTimeout(r.timeout).
		ProbeEvery(r.interval).
		Should(matcher, optionalDescription...)
}

// ShouldNot runs the Eventually assertion with the given matcher
func (r *RetryAssertion) ShouldNot(matcher types.GomegaMatcher, optionalDescription ...any) bool {
	return EventuallyWithOffset(1, r.actualOrCtx, r.args...).
		WithTimeout(r.timeout).
		ProbeEvery(r.interval).
		ShouldNot(matcher, optionalDescription...)
}
