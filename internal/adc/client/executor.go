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
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/api7/gopkg/pkg/log"
	"go.uber.org/zap"
	"k8s.io/utils/ptr"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/internal/types"
)

const (
	defaultHTTPADCExecutorAddr = "http://127.0.0.1:3000"
)

type ADCExecutor interface {
	Execute(ctx context.Context, mode string, config adctypes.Config, args []string) error
}

type DefaultADCExecutor struct {
	sync.Mutex
}

func (e *DefaultADCExecutor) Execute(ctx context.Context, mode string, config adctypes.Config, args []string) error {
	e.Lock()
	defer e.Unlock()

	return e.runADC(ctx, mode, config, args)
}

func (e *DefaultADCExecutor) runADC(ctx context.Context, mode string, config adctypes.Config, args []string) error {
	var execErrs = types.ADCExecutionError{
		Name: config.Name,
	}

	for _, addr := range config.ServerAddrs {
		if err := e.runForSingleServerWithTimeout(ctx, addr, mode, config, args); err != nil {
			log.Errorw("failed to run adc for server", zap.String("server", addr), zap.Error(err))
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

func (e *DefaultADCExecutor) runForSingleServerWithTimeout(ctx context.Context, serverAddr, mode string, config adctypes.Config, args []string) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	return e.runForSingleServer(ctx, serverAddr, mode, config, args)
}

func (e *DefaultADCExecutor) runForSingleServer(ctx context.Context, serverAddr, mode string, config adctypes.Config, args []string) error {
	cmdArgs := append([]string{}, args...)
	if !config.TlsVerify {
		cmdArgs = append(cmdArgs, "--tls-skip-verify")
	}

	cmdArgs = append(cmdArgs, "--timeout", "15s")

	env := e.prepareEnv(serverAddr, mode, config.Token)

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "adc", cmdArgs...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = append(os.Environ(), env...)

	log.Debugw("running adc command",
		zap.String("command", strings.Join(cmd.Args, " ")),
		zap.Strings("env", filterSensitiveEnv(env)),
	)

	if err := cmd.Run(); err != nil {
		return e.buildCmdError(err, stdout.Bytes(), stderr.Bytes())
	}

	result, err := e.handleOutput(stdout.Bytes())
	if err != nil {
		log.Errorw("failed to handle adc output",
			zap.Error(err),
			zap.String("stdout", stdout.String()),
			zap.String("stderr", stderr.String()),
		)
		return fmt.Errorf("failed to handle adc output: %w", err)
	}
	if result.FailedCount > 0 && len(result.Failed) > 0 {
		log.Errorw("adc sync failed", zap.Any("result", result))
		return types.ADCExecutionServerAddrError{
			ServerAddr:     serverAddr,
			Err:            result.Failed[0].Reason,
			FailedStatuses: result.Failed,
		}
	}
	log.Debugw("adc sync success", zap.Any("result", result))
	return nil
}

func (e *DefaultADCExecutor) prepareEnv(serverAddr, mode, token string) []string {
	return []string{
		"ADC_EXPERIMENTAL_FEATURE_FLAGS=remote-state-file,parallel-backend-request",
		"ADC_RUNNING_MODE=ingress",
		"ADC_BACKEND=" + mode,
		"ADC_SERVER=" + serverAddr,
		"ADC_TOKEN=" + token,
	}
}

// filterSensitiveEnv filters out sensitive information from environment variables for logging
func filterSensitiveEnv(env []string) []string {
	filtered := make([]string, 0, len(env))
	for _, envVar := range env {
		if strings.Contains(envVar, "ADC_TOKEN=") {
			filtered = append(filtered, "ADC_TOKEN=***")
		} else {
			filtered = append(filtered, envVar)
		}
	}
	return filtered
}

func (e *DefaultADCExecutor) buildCmdError(runErr error, stdout, stderr []byte) error {
	errMsg := string(stderr)
	if errMsg == "" {
		errMsg = string(stdout)
	}
	log.Errorw("failed to run adc",
		zap.Error(runErr),
		zap.String("output", string(stdout)),
		zap.String("stderr", string(stderr)),
	)
	return errors.New("failed to sync resources: " + errMsg + ", exit err: " + runErr.Error())
}

func (e *DefaultADCExecutor) handleOutput(output []byte) (*adctypes.SyncResult, error) {
	var result adctypes.SyncResult
	log.Debugw("adc output", zap.String("output", string(output)))
	if lines := bytes.Split(output, []byte{'\n'}); len(lines) > 0 {
		output = lines[len(lines)-1]
	}
	if err := json.Unmarshal(output, &result); err != nil {
		log.Errorw("failed to unmarshal adc output",
			zap.Error(err),
			zap.String("stdout", string(output)),
		)
		return nil, errors.New("failed to parse adc result: " + err.Error())
	}

	return &result, nil
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
	Server              string            `json:"server"`
	Token               string            `json:"token"`
	LabelSelector       map[string]string `json:"labelSelector,omitempty"`
	IncludeResourceType []string          `json:"includeResourceType,omitempty"`
	TlsSkipVerify       *bool             `json:"tlsSkipVerify,omitempty"`
}

// HTTPADCExecutor implements ADCExecutor interface using HTTP calls to ADC Server
type HTTPADCExecutor struct {
	sync.Mutex
	httpClient *http.Client
	serverURL  string
}

// NewHTTPADCExecutor creates a new HTTPADCExecutor with the specified ADC Server URL
func NewHTTPADCExecutor(serverURL string, timeout time.Duration) *HTTPADCExecutor {
	return &HTTPADCExecutor{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		serverURL: serverURL,
	}
}

// Execute implements the ADCExecutor interface using HTTP calls
func (e *HTTPADCExecutor) Execute(ctx context.Context, mode string, config adctypes.Config, args []string) error {
	e.Lock()
	defer e.Unlock()

	return e.runHTTPSync(ctx, mode, config, args)
}

// runHTTPSync performs HTTP sync to ADC Server for each server address
func (e *HTTPADCExecutor) runHTTPSync(ctx context.Context, mode string, config adctypes.Config, args []string) error {
	var execErrs = types.ADCExecutionError{
		Name: config.Name,
	}

	for _, addr := range config.ServerAddrs {
		if err := e.runHTTPSyncForSingleServer(ctx, addr, mode, config, args); err != nil {
			log.Errorw("failed to run http sync for server", zap.String("server", addr), zap.Error(err))
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
func (e *HTTPADCExecutor) runHTTPSyncForSingleServer(ctx context.Context, serverAddr, mode string, config adctypes.Config, args []string) error {
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
	req, err := e.buildHTTPRequest(ctx, serverAddr, mode, config, labels, types, resources)
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
			log.Warnw("failed to close response body", zap.Error(closeErr))
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
func (e *HTTPADCExecutor) buildHTTPRequest(ctx context.Context, serverAddr, mode string, config adctypes.Config, labels map[string]string, types []string, resources *adctypes.Resources) (*http.Request, error) {
	// Prepare request body
	tlsVerify := config.TlsVerify
	reqBody := ADCServerRequest{
		Task: ADCServerTask{
			Opts: ADCServerOpts{
				Backend:             mode,
				Server:              serverAddr,
				Token:               config.Token,
				LabelSelector:       labels,
				IncludeResourceType: types,
				TlsSkipVerify:       ptr.To(!tlsVerify),
			},
			Config: *resources,
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	log.Debugw("request body", zap.String("body", string(jsonData)))

	log.Debugw("sending HTTP request to ADC Server",
		zap.String("url", e.serverURL+"/sync"),
		zap.String("server", serverAddr),
		zap.String("mode", mode),
		zap.Any("labelSelector", labels),
		zap.Strings("includeResourceType", types),
		zap.Bool("tlsSkipVerify", !tlsVerify),
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

	log.Debugw("received HTTP response from ADC Server",
		zap.String("server", serverAddr),
		zap.Int("status", resp.StatusCode),
		zap.String("response", string(body)),
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
		log.Errorw("failed to unmarshal ADC Server response",
			zap.Error(err),
			zap.String("response", string(body)),
		)
		return fmt.Errorf("failed to parse ADC Server response: %w", err)
	}

	// Check for sync failures
	if result.FailedCount > 0 && len(result.Failed) > 0 {
		log.Errorw("ADC Server sync failed", zap.Any("result", result))
		return types.ADCExecutionServerAddrError{
			ServerAddr:     serverAddr,
			Err:            result.Failed[0].Reason,
			FailedStatuses: result.Failed,
		}
	}

	log.Debugw("ADC Server sync success", zap.Any("result", result))
	return nil
}
