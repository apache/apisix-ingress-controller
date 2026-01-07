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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
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
	req, err := e.buildHTTPRequest(ctx, serverAddr, config, labels, types, resources)
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
func (e *HTTPADCExecutor) buildHTTPRequest(ctx context.Context, serverAddr string, config adctypes.Config, labels map[string]string, types []string, resources *adctypes.Resources) (*http.Request, error) {
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
		"url", e.serverURL+"/sync",
		"server", serverAddr,
		"mode", config.BackendType,
		"cacheKey", config.Name,
		"labelSelector", labels,
		"includeResourceType", types,
		"tlsSkipVerify", !tlsVerify,
	)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "PUT", e.serverURL+"/sync", bytes.NewBuffer(jsonData))
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
