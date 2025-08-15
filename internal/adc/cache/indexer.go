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

package cache

import (
	"fmt"
	"strings"

	"github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/internal/controller/label"
)

const (
	KindLabelIndex = "label"
)

/*
var KindLabelIndexer = LabelIndexer{
	LabelKeys: []string{label.LabelKind, label.LabelName, label.LabelNamespace},
	GetLabels: func(obj adc.Object) map[string]string {
		return obj.GetLabels()
	},
}
*/

var (
	KindLabelIndexer = LabelIndexer{
		LabelKeys: []string{label.LabelKind, label.LabelNamespace, label.LabelName},
		GetLabels: func(obj any) map[string]string {
			o, ok := obj.(adc.Object)
			if !ok {
				return nil
			}
			return o.GetLabels()
		},
	}
)

type LabelIndexer struct {
	LabelKeys []string
	GetLabels func(obj any) map[string]string
}

// ref: https://pkg.go.dev/github.com/hashicorp/go-memdb#Txn.Get
// by adding suffixes to avoid prefix matching
func (emi *LabelIndexer) genKey(labelValues []string) []byte {
	return []byte(strings.Join(labelValues, "/") + "\x00")
}

func (emi *LabelIndexer) FromObject(obj any) (bool, []byte, error) {
	labels := emi.GetLabels(obj)
	var labelValues []string
	for _, key := range emi.LabelKeys {
		if value, exists := labels[key]; exists {
			labelValues = append(labelValues, value)
		}
	}

	if len(labelValues) == 0 {
		return false, nil, nil
	}

	return true, emi.genKey(labelValues), nil
}

func (emi *LabelIndexer) FromArgs(args ...any) ([]byte, error) {
	if len(args) != len(emi.LabelKeys) {
		return nil, fmt.Errorf("expected %d arguments, got %d", len(emi.LabelKeys), len(args))
	}

	labelValues := make([]string, 0, len(args))
	for _, arg := range args {
		value, ok := arg.(string)
		if !ok {
			return nil, fmt.Errorf("argument is not a string")
		}
		labelValues = append(labelValues, value)
	}

	return emi.genKey(labelValues), nil
}
