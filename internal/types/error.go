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

package types

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/apache/apisix-ingress-controller/api/adc"
)

type ReasonError struct {
	Reason  string
	Message string
}

func (e ReasonError) Error() string {
	return e.Message
}

func IsSomeReasonError[Reason ~string](err error, reasons ...Reason) bool {
	if err == nil {
		return false
	}
	var re ReasonError
	if !errors.As(err, &re) {
		return false
	}
	if len(reasons) == 0 {
		return true
	}
	return slices.Contains(reasons, Reason(re.Reason))
}

func NewInvalidKindError[Kind ~string](kind Kind) ReasonError {
	return ReasonError{
		Reason:  string(gatewayv1.RouteReasonInvalidKind),
		Message: fmt.Sprintf("Invalid kind %s, only Service is supported", kind),
	}
}

type ADCExecutionErrors struct {
	Errors []ADCExecutionError
}

func (e ADCExecutionErrors) Error() string {
	messages := make([]string, 0, len(e.Errors))
	for _, err := range e.Errors {
		messages = append(messages, err.Error())
	}
	return fmt.Sprintf("ADC execution errors: [%s]", strings.Join(messages, "; "))
}

type ADCExecutionError struct {
	Name         string
	FailedErrors []ADCExecutionServerAddrError
}

func (e ADCExecutionError) Error() string {
	messages := make([]string, 0, len(e.FailedErrors))
	for _, failed := range e.FailedErrors {
		messages = append(messages, failed.Error())
	}
	return fmt.Sprintf("ADC execution error for %s: [%s]", e.Name, strings.Join(messages, "; "))
}

type ADCExecutionServerAddrError struct {
	Err            string
	ServerAddr     string
	FailedStatuses []adc.SyncStatus
}

func (e ADCExecutionServerAddrError) Error() string {
	return fmt.Sprintf("ServerAddr: %s, Err: %s", e.ServerAddr, e.Err)
}
