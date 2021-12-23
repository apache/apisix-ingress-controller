// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package types

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

type structure1 struct {
	Interval TimeDuration `json:"interval" yaml:"interval"`
	Name     string       `json:"name" yaml:"name"`
}

func TestTimeDurationMarshalJSON(t *testing.T) {
	value := &structure1{
		Interval: TimeDuration{15 * time.Second},
		Name:     "alex",
	}
	data, err := json.Marshal(value)
	assert.Nil(t, err, "failed to marshal value: %s", err)
	assert.Contains(t, string(data), "15s", "bad marshalled json: %s", string(data))
}

func TestTimeDurationUnmarshalJSON(t *testing.T) {
	data := `
		{
			"interval": "3m",
			"name": "alex"
		}`

	var value structure1
	err := json.Unmarshal([]byte(data), &value)
	assert.Nil(t, err, "failed to unmarshal data to structure1: %v", err)
	assert.Equal(t, "alex", value.Name, "bad name: %s", value.Name)
	assert.Equal(t, TimeDuration{3 * time.Minute}, value.Interval, "bad interval: %v", value.Interval)
}

func TestTimeDurationMarshalYAML(t *testing.T) {
	value := &structure1{
		Interval: TimeDuration{15 * time.Second},
		Name:     "alex",
	}
	data, err := yaml.Marshal(value)
	assert.Nil(t, err, "failed to marshal value: %s", err)
	assert.Contains(t, string(data), "15s", "bad marshalled json: %s", string(data))
}

func TestTimeDurationUnmarshalYAML(t *testing.T) {
	data := `
interval: 3m
name: alex
`
	var value structure1
	err := yaml.Unmarshal([]byte(data), &value)
	assert.Nil(t, err, "failed to unmarshal data to structure1: %v", err)
	assert.Equal(t, "alex", value.Name, "bad name: %s", value.Name)
	assert.Equal(t, TimeDuration{3 * time.Minute}, value.Interval, "bad interval: %v", value.Interval)
}
