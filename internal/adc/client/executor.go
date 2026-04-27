// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/utils/ptr"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/internal/types"
)

const (
	defaultHTTPADCExecutorAddr = "http://127.0.0.1:3000"
)

type ADCExecutor interface {
	Execute(ctx context.Context, config adctypes.Config, args []string) error
	Validate(ctx context.Context, config adctypes.Config, args []string) error
}

func BuildADCExecuteArgs(filePath string, labels map[string]string, types []string) []string {
	args := []string{
		"sync",
		"-f", filePath,
	}
	for k, v := range labels {
		args = append(args, "--label-selector", k+"="+v)
	}
	for _, t := range types {
		args = append(args, "--include-resource-type", t)
	}
	return args
}

// ADCServerRequest represents the request body for ADC Server /sync endpoint
type ADCServerRequest struct {
	Task ADCServerTask `json:"task"`
}

// ADCServerTask represents the task configuration in ADC Server request
type ADCServerTask struct {
	Opts   ADCServerOpts      `json:"opts"`
	Config adctypes.Resources `json:"config"`
}

// ADCServerOpts represents the options in ADC Server task
type ADCServerOpts struct {
	Backend             string            `json:"backend"`
	Server              []string          `json:"server"`
	Token               string            `json:"token"`
	LabelSelector       map[string]string `json:"labelSelector,omitempty"`
	IncludeResourceType []string          `json:"includeResourceType,omitempty"`
	TlsSkipVerify       *bool             `json:"tlsSkipVerify,omitempty"`
	CacheKey            string            `json:"cacheKey"`
}

type ADCValidateResult struct {
	Success      *bool                       `json:"success,omitempty"`
	ErrorMessage string                      `json:"errorMessage,omitempty"`
	Errors       []types.ADCValidationDetail `json:"errors,omitempty"`
}

// HTTPADCExecutor implements ADCExecutor interface using HTTP calls to ADC Server
type HTTPADCExecutor struct {
	httpClient *http.Client
	serverURL  string
	log        logr.Logger
}

// NewHTTPADCExecutor creates a new HTTPADCExecutor with the specified ADC Server URL.
// serverURL can be "http(s)://host:port" or "unix:///path/to/socket" or "unix:/path/to/socket".
func NewHTTPADCExecutor(log logr.Logger, serverURL string, timeout time.Duration) *HTTPADCExecutor {
	httpClient := &http.Client{
		Timeout: timeout,
	}

	if strings.HasPrefix(serverURL, "unix:") {
		var socketPath string
		if strings.HasPrefix(serverURL, "unix:///") {
			socketPath = strings.TrimPrefix(serverURL, "unix://")
		} else {
			socketPath = strings.TrimPrefix(serverURL, "unix:")
		}
		transport := &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
			},
		}
		httpClient.Transport = transport
		serverURL = "http://unix"
	}

	return &HTTPADCExecutor{
		httpClient: httpClient,
		serverURL:  serverURL,
		log:        log.WithName("executor"),
	}
}

// Execute implements the ADCExecutor interface using HTTP calls
func (e *HTTPADCExecutor) Execute(ctx context.Context, config adctypes.Config, args []string) error {
	return e.runHTTPSync(ctx, config, args)
}

func (e *HTTPADCExecutor) Validate(ctx context.Context, config adctypes.Config, args []string) error {
	return e.runHTTPValidate(ctx, config, args)
}

// runHTTPSync performs HTTP sync to ADC Server for each server address
func (e *HTTPADCExecutor) runHTTPSync(ctx context.Context, config adctypes.Config, args []string) error {
	var execErrs = types.ADCExecutionError{
		Name: config.Name,
	}

	serverAddrs := func() []string {
		if config.BackendType == "apisix-standalone" {
			return []string{strings.Join(config.ServerAddrs, ",")}
		}
		return config.ServerAddrs
	}()
	e.log.V(1).Info("running http sync", "serverAddrs", serverAddrs)

	for _, addr := range serverAddrs {
		if err := e.runHTTPSyncForSingleServer(ctx, addr, config, args); err != nil {
			e.log.Error(err, "failed to run http sync for server", "server", addr)
			var execErr types.ADCExecutionServerAddrError
			if errors.As(err, &execErr) {
				execErrs.FailedErrors = append(execErrs.FailedErrors, execErr)
			} else {
				execErrs.FailedErrors = append(execErrs.FailedErrors, types.ADCExecutionServerAddrError{
					ServerAddr: addr,
					Err:        err.Error(),
				})
			}
		}
	}
	if len(execErrs.FailedErrors) > 0 {
		return execErrs
	}
	return nil
}

func (e *HTTPADCExecutor) runHTTPValidate(ctx context.Context, config adctypes.Config, args []string) error {
	var validationErr = types.ADCValidationError{
		Name: config.Name,
	}
	var infraErrs []error

	serverAddrs := func() []string {
		return config.ServerAddrs
	}()
	e.log.V(1).Info("running http validate", "serverAddrs", serverAddrs)

	for _, addr := range serverAddrs {
		if err := e.runHTTPValidateForSingleServer(ctx, addr, config, args); err != nil {
			e.log.Error(err, "failed to run http validate for server", "server", addr)
			var validationServerErr types.ADCValidationServerAddrError
			if errors.As(err, &validationServerErr) {
				validationErr.FailedErrors = append(validationErr.FailedErrors, validationServerErr)
				continue
			}
			infraErrs = append(infraErrs, err)
		}
	}

	if len(validationErr.FailedErrors) > 0 {
		return validationErr
	}
	if len(infraErrs) > 0 {
		return errors.Join(infraErrs...)
	}
	return nil
}

// runHTTPSyncForSingleServer performs HTTP sync to a single ADC Server
func (e *HTTPADCExecutor) runHTTPSyncForSingleServer(ctx context.Context, serverAddr string, config adctypes.Config, args []string) error {
	ctx, cancel := context.WithTimeout(ctx, e.httpClient.Timeout)
	defer cancel()

	// Parse args to extract labels, types, and file path
	labels, types, filePath, err := e.parseArgs(args)
	if err != nil {
		return fmt.Errorf("failed to parse args: %w", err)
	}

	// Load resources from file
	resources, err := e.loadResourcesFromFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to load resources from file %s: %w", filePath, err)
	}

	// Build HTTP request
	req, err := e.buildHTTPRequest(ctx, serverAddr, config, labels, types, resources, http.MethodPut, "/sync")
	if err != nil {
		return fmt.Errorf("failed to build HTTP request: %w", err)
	}

	// Send HTTP request
	resp, err := e.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			e.log.Error(closeErr, "failed to close response body")
		}
	}()

	// Handle HTTP response
	return e.handleHTTPResponse(resp, serverAddr)
}

func (e *HTTPADCExecutor) runHTTPValidateForSingleServer(ctx context.Context, serverAddr string, config adctypes.Config, args []string) error {
	ctx, cancel := context.WithTimeout(ctx, e.httpClient.Timeout)
	defer cancel()

	labels, types, filePath, err := e.parseArgs(args)
	if err != nil {
		return fmt.Errorf("failed to parse args: %w", err)
	}

	resources, err := e.loadResourcesFromFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to load resources from file %s: %w", filePath, err)
	}

	var (
		req        *http.Request
		httpClient = e.httpClient
	)
	if config.BackendType == "apisix-standalone" {
		req, err = e.buildAPISIXValidateRequest(ctx, serverAddr, config, resources)
		httpClient = e.newBackendHTTPClient(config)
	} else {
		req, err = e.buildHTTPRequest(ctx, serverAddr, config, labels, types, resources, http.MethodPost, "/validate")
	}
	if err != nil {
		return fmt.Errorf("failed to build validate request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			e.log.Error(closeErr, "failed to close response body")
		}
	}()

	return e.handleHTTPValidateResponse(resp, serverAddr)
}

type apisixValidateRequest struct {
	Routes         []map[string]any `json:"routes,omitempty"`
	Services       []map[string]any `json:"services,omitempty"`
	Consumers      []map[string]any `json:"consumers,omitempty"`
	SSLs           []map[string]any `json:"ssls,omitempty"`
	GlobalRules    []map[string]any `json:"global_rules,omitempty"`
	StreamRoutes   []map[string]any `json:"stream_routes,omitempty"`
	PluginMetadata []map[string]any `json:"plugin_metadata,omitempty"`
	Upstreams      []map[string]any `json:"upstreams,omitempty"`
}

func (e *HTTPADCExecutor) buildAPISIXValidateRequest(ctx context.Context, serverAddr string, config adctypes.Config, resources *adctypes.Resources) (*http.Request, error) {
	body, err := buildAPISIXValidatePayload(resources)
	if err != nil {
		return nil, err
	}

	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal APISIX validate request body: %w", err)
	}

	validateURL, err := url.JoinPath(serverAddr, "/apisix/admin/configs/validate")
	if err != nil {
		return nil, fmt.Errorf("failed to build APISIX validate URL: %w", err)
	}

	e.log.V(1).Info("sending APISIX validate request",
		"url", validateURL,
		"server", serverAddr,
		"cacheKey", config.Name,
		"body", body,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, validateURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create APISIX validate request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", config.Token)
	return req, nil
}

func (e *HTTPADCExecutor) newBackendHTTPClient(config adctypes.Config) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if !config.TlsVerify {
		if transport.TLSClientConfig == nil {
			transport.TLSClientConfig = &tls.Config{}
		}
		transport.TLSClientConfig.InsecureSkipVerify = true
	}

	return &http.Client{
		Timeout:   e.httpClient.Timeout,
		Transport: transport,
	}
}

func buildAPISIXValidatePayload(resources *adctypes.Resources) (*apisixValidateRequest, error) {
	body := &apisixValidateRequest{}

	for _, service := range resources.Services {
		if service == nil {
			continue
		}

		serviceMap, err := toMap(service)
		if err != nil {
			return nil, err
		}
		delete(serviceMap, "routes")
		delete(serviceMap, "stream_routes")
		delete(serviceMap, "upstreams")

		body.Services = append(body.Services, serviceMap)

		for _, upstream := range service.Upstreams {
			upstreamMap, err := toMap(upstream)
			if err != nil {
				return nil, err
			}
			body.Upstreams = append(body.Upstreams, upstreamMap)
		}

		for _, route := range service.Routes {
			routeMap, err := buildAPISIXRouteValidateObject(route)
			if err != nil {
				return nil, err
			}
			if service.ID != "" {
				routeMap["service_id"] = service.ID
			}
			body.Routes = append(body.Routes, routeMap)
		}

		for _, streamRoute := range service.StreamRoutes {
			streamRouteMap, err := toMap(streamRoute)
			if err != nil {
				return nil, err
			}
			body.StreamRoutes = append(body.StreamRoutes, streamRouteMap)
		}
	}

	for _, consumer := range resources.Consumers {
		consumerMap, err := buildAPISIXConsumerValidateObject(consumer)
		if err != nil {
			return nil, err
		}
		body.Consumers = append(body.Consumers, consumerMap)
	}

	for _, ssl := range resources.SSLs {
		sslMap, err := buildAPISIXSSLValidateObject(ssl)
		if err != nil {
			return nil, err
		}
		body.SSLs = append(body.SSLs, sslMap)
	}

	return body, nil
}

func buildAPISIXRouteValidateObject(route *adctypes.Route) (map[string]any, error) {
	routeMap, err := toMap(route)
	if err != nil {
		return nil, err
	}

	delete(routeMap, "description")
	return routeMap, nil
}

func buildAPISIXConsumerValidateObject(consumer *adctypes.Consumer) (map[string]any, error) {
	consumerMap, err := toMap(consumer)
	if err != nil {
		return nil, err
	}

	if len(consumer.Credentials) == 0 {
		return consumerMap, nil
	}

	plugins, ok := consumerMap["plugins"].(map[string]any)
	if !ok || plugins == nil {
		plugins = make(map[string]any, len(consumer.Credentials))
	}

	for _, credential := range consumer.Credentials {
		plugins[credential.Type] = credential.Config
	}

	consumerMap["plugins"] = plugins
	delete(consumerMap, "credentials")
	return consumerMap, nil
}

func buildAPISIXSSLValidateObject(ssl *adctypes.SSL) (map[string]any, error) {
	sslMap, err := toMap(ssl)
	if err != nil {
		return nil, err
	}

	delete(sslMap, "certificates")

	switch len(ssl.Certificates) {
	case 0:
		return sslMap, nil
	case 1:
		sslMap["cert"] = ssl.Certificates[0].Certificate
		sslMap["key"] = ssl.Certificates[0].Key
	default:
		sslMap["cert"] = ssl.Certificates[0].Certificate
		sslMap["key"] = ssl.Certificates[0].Key

		certs := make([]string, 0, len(ssl.Certificates)-1)
		keys := make([]string, 0, len(ssl.Certificates)-1)
		for _, certificate := range ssl.Certificates[1:] {
			certs = append(certs, certificate.Certificate)
			keys = append(keys, certificate.Key)
		}
		sslMap["certs"] = certs
		sslMap["keys"] = keys
	}

	return sslMap, nil
}

func toMap(obj any) (map[string]any, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal validation object: %w", err)
	}

	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("failed to unmarshal validation object: %w", err)
	}
	return out, nil
}

// parseArgs parses the command line arguments to extract labels, types, and file path
func (e *HTTPADCExecutor) parseArgs(args []string) (map[string]string, []string, string, error) {
	labels := make(map[string]string)
	var types []string
	var filePath string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-f":
			if i+1 < len(args) {
				filePath = args[i+1]
				i++
			}
		case "--label-selector":
			if i+1 < len(args) {
				labelPair := args[i+1]
				parts := strings.SplitN(labelPair, "=", 2)
				if len(parts) == 2 {
					labels[parts[0]] = parts[1]
				}
				i++
			}
		case "--include-resource-type":
			if i+1 < len(args) {
				types = append(types, args[i+1])
				i++
			}
		}
	}

	if filePath == "" {
		return nil, nil, "", errors.New("file path not found in args")
	}

	return labels, types, filePath, nil
}

// loadResourcesFromFile loads ADC resources from the specified file
func (e *HTTPADCExecutor) loadResourcesFromFile(filePath string) (*adctypes.Resources, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var resources adctypes.Resources
	if err := json.Unmarshal(data, &resources); err != nil {
		return nil, fmt.Errorf("failed to unmarshal resources: %w", err)
	}

	return &resources, nil
}

// buildHTTPRequest builds the HTTP request for ADC Server
func (e *HTTPADCExecutor) buildHTTPRequest(ctx context.Context, serverAddr string, config adctypes.Config, labels map[string]string, types []string, resources *adctypes.Resources, method string, path string) (*http.Request, error) {
	// Prepare request body
	tlsVerify := config.TlsVerify
	reqBody := ADCServerRequest{
		Task: ADCServerTask{
			Opts: ADCServerOpts{
				Backend:             config.BackendType,
				Server:              strings.Split(serverAddr, ","),
				Token:               config.Token,
				LabelSelector:       labels,
				IncludeResourceType: types,
				TlsSkipVerify:       ptr.To(!tlsVerify),
				CacheKey:            config.Name,
			},
			Config: *resources,
		},
	}

	e.log.V(1).Info("prepared request body", "body", reqBody)

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	e.log.V(1).Info("sending HTTP request to ADC Server",
		"url", e.serverURL+path,
		"server", serverAddr,
		"mode", config.BackendType,
		"cacheKey", config.Name,
		"labelSelector", labels,
		"includeResourceType", types,
		"tlsSkipVerify", !tlsVerify,
	)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, method, e.serverURL+path, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

// handleHTTPResponse handles the HTTP response from ADC Server
func (e *HTTPADCExecutor) handleHTTPResponse(resp *http.Response, serverAddr string) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	e.log.V(1).Info("received HTTP response from ADC Server",
		"server", serverAddr,
		"status", resp.StatusCode,
		"response", string(body),
	)

	// not only 200, HTTP 202 is also accepted
	if resp.StatusCode/100 != 2 {
		return types.ADCExecutionServerAddrError{
			ServerAddr: serverAddr,
			Err:        fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)),
		}
	}

	// Parse response body
	var result adctypes.SyncResult
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to unmarshal response body: %s, err: %w", string(body), err)
	}

	// Check for sync failures
	// For apisix-standalone mode: Failed is always empty, check EndpointStatus instead
	if result.FailedCount > 0 {
		if len(result.Failed) > 0 {
			reason := result.Failed[0].Reason
			e.log.Error(fmt.Errorf("ADC Server sync failed: %s", reason), "ADC Server sync failed", "result", result)
			return types.ADCExecutionServerAddrError{
				ServerAddr:     serverAddr,
				Err:            reason,
				FailedStatuses: result.Failed,
			}
		}
		if len(result.EndpointStatus) > 0 {
			// apisix-standalone mode: use EndpointStatus
			var failedEndpoints []string
			for _, ep := range result.EndpointStatus {
				if !ep.Success {
					failedEndpoints = append(failedEndpoints, fmt.Sprintf("%s: %s", ep.Server, ep.Reason))
				}
			}
			if len(failedEndpoints) > 0 {
				reason := strings.Join(failedEndpoints, "; ")
				e.log.Error(fmt.Errorf("ADC Server sync failed (standalone mode): %s", reason), "ADC Server sync failed", "result", result)
				return types.ADCExecutionServerAddrError{
					ServerAddr: serverAddr,
					Err:        reason,
					FailedStatuses: []adctypes.SyncStatus{
						{Reason: reason},
					},
				}
			}
		}
	}

	e.log.V(1).Info("ADC Server sync success", "result", result)
	return nil
}

func (e *HTTPADCExecutor) handleHTTPValidateResponse(resp *http.Response, serverAddr string) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	e.log.V(1).Info("received HTTP validate response from ADC Server",
		"server", serverAddr,
		"status", resp.StatusCode,
		"response", string(body),
	)

	parseValidationResult := func() *ADCValidateResult {
		if len(body) == 0 {
			return nil
		}
		var result ADCValidateResult
		if err := json.Unmarshal(body, &result); err != nil {
			return nil
		}
		return &result
	}

	if resp.StatusCode == http.StatusBadRequest {
		result := parseValidationResult()
		errMsg := string(body)
		if result != nil && result.ErrorMessage != "" {
			errMsg = result.ErrorMessage
		}
		return types.ADCValidationServerAddrError{
			ServerAddr: serverAddr,
			Err:        errMsg,
			ValidationErrors: func() []types.ADCValidationDetail {
				if result == nil {
					return nil
				}
				return result.Errors
			}(),
		}
	}

	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	if result := parseValidationResult(); result != nil && result.Success != nil && !*result.Success {
		errMsg := result.ErrorMessage
		if errMsg == "" {
			errMsg = "ADC validation failed"
		}
		return types.ADCValidationServerAddrError{
			ServerAddr:       serverAddr,
			Err:              errMsg,
			ValidationErrors: result.Errors,
		}
	}

	return nil
}
