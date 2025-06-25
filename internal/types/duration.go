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
	"encoding/json"
	"fmt"
	"time"
)

// TimeDuration is yet another time.Duration but implements json.Unmarshaler
// and json.Marshaler, yaml.Unmarshaler and yaml.Marshaler interfaces so one
// can use "1h", "5s" and etc in their json/yaml configurations.
//
// Note the format to represent time is same as time.Duration.
// See the comments about time.ParseDuration for more details.
type TimeDuration struct {
	time.Duration `json:",inline"`
}

func (d *TimeDuration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *TimeDuration) UnmarshalJSON(data []byte) error {
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	switch v := value.(type) {
	case float64:
		d.Duration = time.Duration(v)
	case string:
		dur, err := time.ParseDuration(v)
		if err != nil {
			return err
		}
		d.Duration = dur
	default:
		return fmt.Errorf("unknown type: %T", v)
	}
	return nil
}

func (d *TimeDuration) MarshalYAML() (any, error) {
	return d.String(), nil
}

func (d *TimeDuration) UnmarshalYAML(unmarshal func(any) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	d.Duration = dur
	return nil
}
