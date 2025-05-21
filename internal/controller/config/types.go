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
	"github.com/apache/apisix-ingress-controller/internal/types"
)

const (
	// IngressAPISIXLeader is the default election id for the controller
	// leader election.
	DefaultLeaderElectionID = "apisix-ingress-gateway-leader"

	// IngressClass is the default ingress class name, used for Ingress
	// object's IngressClassName field in Kubernetes clusters version v1.18.0
	// or higher, or the annotation "kubernetes.io/ingress.class" (deprecated).
	DefaultControllerName = "apisix.apache.org/apisix-ingress-controller"

	// DefaultLogLevel is the default log level for apisix-ingress-controller.
	DefaultLogLevel = "info"

	DefaultMetricsAddr = ":8080"
	DefaultProbeAddr   = ":8081"
)

// Config contains all config items which are necessary for
// apisix-ingress-controller's running.
type Config struct {
	LogLevel         string             `json:"log_level" yaml:"log_level"`
	ControllerName   string             `json:"controller_name" yaml:"controller_name"`
	LeaderElectionID string             `json:"leader_election_id" yaml:"leader_election_id"`
	MetricsAddr      string             `json:"metrics_addr" yaml:"metrics_addr"`
	EnableHTTP2      bool               `json:"enable_http2" yaml:"enable_http2"`
	ProbeAddr        string             `json:"probe_addr" yaml:"probe_addr"`
	SecureMetrics    bool               `json:"secure_metrics" yaml:"secure_metrics"`
	LeaderElection   *LeaderElection    `json:"leader_election" yaml:"leader_election"`
	ExecADCTimeout   types.TimeDuration `json:"exec_adc_timeout" yaml:"exec_adc_timeout"`
	ProviderConfig   ProviderConfig     `json:"provider" yaml:"provider"`
}

type GatewayConfig struct {
	Name         string              `json:"name" yaml:"name"`
	ControlPlane *ControlPlaneConfig `json:"control_plane" yaml:"control_plane"`
	Addresses    []string            `json:"addresses" yaml:"addresses"`
}

type ControlPlaneConfig struct {
	AdminKey  string   `json:"admin_key" yaml:"admin_key"`
	Endpoints []string `json:"endpoints" yaml:"endpoints"`
	TLSVerify *bool    `json:"tls_verify" yaml:"tls_verify"`
}

type LeaderElection struct {
	LeaseDuration types.TimeDuration `json:"lease_duration,omitempty" yaml:"lease_duration,omitempty"`
	RenewDeadline types.TimeDuration `json:"renew_deadline,omitempty" yaml:"renew_deadline,omitempty"`
	RetryPeriod   types.TimeDuration `json:"retry_period,omitempty" yaml:"retry_period,omitempty"`
	Disable       bool               `json:"disable,omitempty" yaml:"disable,omitempty"`
}

type ProviderConfig struct {
	SyncPeriod    types.TimeDuration `json:"sync_period" yaml:"sync_period"`
	InitSyncDelay types.TimeDuration `json:"init_sync_delay" yaml:"init_sync_delay"`
}
