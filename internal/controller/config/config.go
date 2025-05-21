// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/template"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/apache/apisix-ingress-controller/internal/types"
)

var (
	ControllerConfig = NewDefaultConfig()
)

func SetControllerConfig(cfg *Config) {
	ControllerConfig = cfg
}

// NewDefaultConfig creates a Config object which fills all config items with
// default value.
func NewDefaultConfig() *Config {
	return &Config{
		LogLevel:         DefaultLogLevel,
		ControllerName:   DefaultControllerName,
		LeaderElectionID: DefaultLeaderElectionID,
		ProbeAddr:        DefaultProbeAddr,
		MetricsAddr:      DefaultMetricsAddr,
		LeaderElection:   NewLeaderElection(),
		ExecADCTimeout:   types.TimeDuration{Duration: 15 * time.Second},
		ProviderConfig: ProviderConfig{
			SyncPeriod:    types.TimeDuration{Duration: 0},
			InitSyncDelay: types.TimeDuration{Duration: 20 * time.Minute},
		},
	}
}

func NewLeaderElection() *LeaderElection {
	return &LeaderElection{
		LeaseDuration: types.TimeDuration{Duration: 30 * time.Second},
		RenewDeadline: types.TimeDuration{Duration: 20 * time.Second},
		RetryPeriod:   types.TimeDuration{Duration: 2 * time.Second},
		Disable:       false,
	}
}

// NewConfigFromFile creates a Config object and fills all config items according
// to the configuration file. The file can be in JSON/YAML format, which will be
// distinguished according to the file suffix.
func NewConfigFromFile(filename string) (*Config, error) {
	cfg := NewDefaultConfig()
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	envVarMap := map[string]string{}
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		envVarMap[pair[0]] = pair[1]
	}

	tpl := template.New("text").Option("missingkey=error")
	tpl, err = tpl.Parse(string(data))
	if err != nil {
		return nil, fmt.Errorf("error parsing configuration template %v", err)
	}
	buf := bytes.NewBufferString("")
	err = tpl.Execute(buf, envVarMap)
	if err != nil {
		return nil, fmt.Errorf("error execute configuration template %v", err)
	}

	if strings.HasSuffix(filename, ".yaml") || strings.HasSuffix(filename, ".yml") {
		err = yaml.Unmarshal(buf.Bytes(), cfg)
	} else {
		err = json.Unmarshal(buf.Bytes(), cfg)
	}

	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) Validate() error {
	if c.ControllerName == "" {
		return fmt.Errorf("controller_name is required")
	}
	return nil
}

func GetControllerName() string {
	return ControllerConfig.ControllerName
}
