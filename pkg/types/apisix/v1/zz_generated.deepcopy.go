// +build !ignore_autogenerated

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

// Code generated by deepcopy-gen. DO NOT EDIT.

package v1

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BasicAuthConsumerConfig) DeepCopyInto(out *BasicAuthConsumerConfig) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BasicAuthConsumerConfig.
func (in *BasicAuthConsumerConfig) DeepCopy() *BasicAuthConsumerConfig {
	if in == nil {
		return nil
	}
	out := new(BasicAuthConsumerConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BasicAuthRouteConfig) DeepCopyInto(out *BasicAuthRouteConfig) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BasicAuthRouteConfig.
func (in *BasicAuthRouteConfig) DeepCopy() *BasicAuthRouteConfig {
	if in == nil {
		return nil
	}
	out := new(BasicAuthRouteConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Consumer) DeepCopyInto(out *Consumer) {
	*out = *in
	if in.Labels != nil {
		in, out := &in.Labels, &out.Labels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	in.Plugins.DeepCopyInto(&out.Plugins)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Consumer.
func (in *Consumer) DeepCopy() *Consumer {
	if in == nil {
		return nil
	}
	out := new(Consumer)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CorsConfig) DeepCopyInto(out *CorsConfig) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CorsConfig.
func (in *CorsConfig) DeepCopy() *CorsConfig {
	if in == nil {
		return nil
	}
	out := new(CorsConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GlobalRule) DeepCopyInto(out *GlobalRule) {
	*out = *in
	in.Plugins.DeepCopyInto(&out.Plugins)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GlobalRule.
func (in *GlobalRule) DeepCopy() *GlobalRule {
	if in == nil {
		return nil
	}
	out := new(GlobalRule)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IPRestrictConfig) DeepCopyInto(out *IPRestrictConfig) {
	*out = *in
	if in.Allowlist != nil {
		in, out := &in.Allowlist, &out.Allowlist
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Blocklist != nil {
		in, out := &in.Blocklist, &out.Blocklist
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IPRestrictConfig.
func (in *IPRestrictConfig) DeepCopy() *IPRestrictConfig {
	if in == nil {
		return nil
	}
	out := new(IPRestrictConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KeyAuthConsumerConfig) DeepCopyInto(out *KeyAuthConsumerConfig) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KeyAuthConsumerConfig.
func (in *KeyAuthConsumerConfig) DeepCopy() *KeyAuthConsumerConfig {
	if in == nil {
		return nil
	}
	out := new(KeyAuthConsumerConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Metadata) DeepCopyInto(out *Metadata) {
	*out = *in
	if in.Labels != nil {
		in, out := &in.Labels, &out.Labels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Metadata.
func (in *Metadata) DeepCopy() *Metadata {
	if in == nil {
		return nil
	}
	out := new(Metadata)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MutualTLSClientConfig) DeepCopyInto(out *MutualTLSClientConfig) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MutualTLSClientConfig.
func (in *MutualTLSClientConfig) DeepCopy() *MutualTLSClientConfig {
	if in == nil {
		return nil
	}
	out := new(MutualTLSClientConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PluginConfig) DeepCopyInto(out *PluginConfig) {
	*out = *in
	in.Metadata.DeepCopyInto(&out.Metadata)
	in.Plugins.DeepCopyInto(&out.Plugins)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PluginConfig.
func (in *PluginConfig) DeepCopy() *PluginConfig {
	if in == nil {
		return nil
	}
	out := new(PluginConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RedirectConfig) DeepCopyInto(out *RedirectConfig) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RedirectConfig.
func (in *RedirectConfig) DeepCopy() *RedirectConfig {
	if in == nil {
		return nil
	}
	out := new(RedirectConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RewriteConfig) DeepCopyInto(out *RewriteConfig) {
	*out = *in
	if in.RewriteTargetRegex != nil {
		in, out := &in.RewriteTargetRegex, &out.RewriteTargetRegex
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RewriteConfig.
func (in *RewriteConfig) DeepCopy() *RewriteConfig {
	if in == nil {
		return nil
	}
	out := new(RewriteConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Route) DeepCopyInto(out *Route) {
	*out = *in
	in.Metadata.DeepCopyInto(&out.Metadata)
	if in.Hosts != nil {
		in, out := &in.Hosts, &out.Hosts
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Timeout != nil {
		in, out := &in.Timeout, &out.Timeout
		*out = new(UpstreamTimeout)
		**out = **in
	}
	if in.Vars != nil {
		in, out := &in.Vars, &out.Vars
		*out = make(Vars, len(*in))
		for i := range *in {
			if (*in)[i] != nil {
				in, out := &(*in)[i], &(*out)[i]
				*out = make([]StringOrSlice, len(*in))
				for i := range *in {
					(*in)[i].DeepCopyInto(&(*out)[i])
				}
			}
		}
	}
	if in.Uris != nil {
		in, out := &in.Uris, &out.Uris
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Methods != nil {
		in, out := &in.Methods, &out.Methods
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.RemoteAddrs != nil {
		in, out := &in.RemoteAddrs, &out.RemoteAddrs
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	in.Plugins.DeepCopyInto(&out.Plugins)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Route.
func (in *Route) DeepCopy() *Route {
	if in == nil {
		return nil
	}
	out := new(Route)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Ssl) DeepCopyInto(out *Ssl) {
	*out = *in
	if in.Snis != nil {
		in, out := &in.Snis, &out.Snis
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Labels != nil {
		in, out := &in.Labels, &out.Labels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Client != nil {
		in, out := &in.Client, &out.Client
		*out = new(MutualTLSClientConfig)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Ssl.
func (in *Ssl) DeepCopy() *Ssl {
	if in == nil {
		return nil
	}
	out := new(Ssl)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *StreamRoute) DeepCopyInto(out *StreamRoute) {
	*out = *in
	if in.Labels != nil {
		in, out := &in.Labels, &out.Labels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Upstream != nil {
		in, out := &in.Upstream, &out.Upstream
		*out = new(Upstream)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StreamRoute.
func (in *StreamRoute) DeepCopy() *StreamRoute {
	if in == nil {
		return nil
	}
	out := new(StreamRoute)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *StringOrSlice) DeepCopyInto(out *StringOrSlice) {
	*out = *in
	if in.SliceVal != nil {
		in, out := &in.SliceVal, &out.SliceVal
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StringOrSlice.
func (in *StringOrSlice) DeepCopy() *StringOrSlice {
	if in == nil {
		return nil
	}
	out := new(StringOrSlice)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TrafficSplitConfig) DeepCopyInto(out *TrafficSplitConfig) {
	*out = *in
	if in.Rules != nil {
		in, out := &in.Rules, &out.Rules
		*out = make([]TrafficSplitConfigRule, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TrafficSplitConfig.
func (in *TrafficSplitConfig) DeepCopy() *TrafficSplitConfig {
	if in == nil {
		return nil
	}
	out := new(TrafficSplitConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TrafficSplitConfigRule) DeepCopyInto(out *TrafficSplitConfigRule) {
	*out = *in
	if in.WeightedUpstreams != nil {
		in, out := &in.WeightedUpstreams, &out.WeightedUpstreams
		*out = make([]TrafficSplitConfigRuleWeightedUpstream, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TrafficSplitConfigRule.
func (in *TrafficSplitConfigRule) DeepCopy() *TrafficSplitConfigRule {
	if in == nil {
		return nil
	}
	out := new(TrafficSplitConfigRule)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TrafficSplitConfigRuleWeightedUpstream) DeepCopyInto(out *TrafficSplitConfigRuleWeightedUpstream) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TrafficSplitConfigRuleWeightedUpstream.
func (in *TrafficSplitConfigRuleWeightedUpstream) DeepCopy() *TrafficSplitConfigRuleWeightedUpstream {
	if in == nil {
		return nil
	}
	out := new(TrafficSplitConfigRuleWeightedUpstream)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Upstream) DeepCopyInto(out *Upstream) {
	*out = *in
	in.Metadata.DeepCopyInto(&out.Metadata)
	if in.Checks != nil {
		in, out := &in.Checks, &out.Checks
		*out = new(UpstreamHealthCheck)
		(*in).DeepCopyInto(*out)
	}
	if in.Nodes != nil {
		in, out := &in.Nodes, &out.Nodes
		*out = make(UpstreamNodes, len(*in))
		copy(*out, *in)
	}
	if in.Retries != nil {
		in, out := &in.Retries, &out.Retries
		*out = new(int)
		**out = **in
	}
	if in.Timeout != nil {
		in, out := &in.Timeout, &out.Timeout
		*out = new(UpstreamTimeout)
		**out = **in
	}
	if in.TLS != nil {
		in, out := &in.TLS, &out.TLS
		*out = new(ClientTLS)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Upstream.
func (in *Upstream) DeepCopy() *Upstream {
	if in == nil {
		return nil
	}
	out := new(Upstream)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *UpstreamActiveHealthCheck) DeepCopyInto(out *UpstreamActiveHealthCheck) {
	*out = *in
	if in.HTTPRequestHeaders != nil {
		in, out := &in.HTTPRequestHeaders, &out.HTTPRequestHeaders
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	in.Healthy.DeepCopyInto(&out.Healthy)
	in.Unhealthy.DeepCopyInto(&out.Unhealthy)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new UpstreamActiveHealthCheck.
func (in *UpstreamActiveHealthCheck) DeepCopy() *UpstreamActiveHealthCheck {
	if in == nil {
		return nil
	}
	out := new(UpstreamActiveHealthCheck)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *UpstreamActiveHealthCheckHealthy) DeepCopyInto(out *UpstreamActiveHealthCheckHealthy) {
	*out = *in
	in.UpstreamPassiveHealthCheckHealthy.DeepCopyInto(&out.UpstreamPassiveHealthCheckHealthy)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new UpstreamActiveHealthCheckHealthy.
func (in *UpstreamActiveHealthCheckHealthy) DeepCopy() *UpstreamActiveHealthCheckHealthy {
	if in == nil {
		return nil
	}
	out := new(UpstreamActiveHealthCheckHealthy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *UpstreamActiveHealthCheckUnhealthy) DeepCopyInto(out *UpstreamActiveHealthCheckUnhealthy) {
	*out = *in
	in.UpstreamPassiveHealthCheckUnhealthy.DeepCopyInto(&out.UpstreamPassiveHealthCheckUnhealthy)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new UpstreamActiveHealthCheckUnhealthy.
func (in *UpstreamActiveHealthCheckUnhealthy) DeepCopy() *UpstreamActiveHealthCheckUnhealthy {
	if in == nil {
		return nil
	}
	out := new(UpstreamActiveHealthCheckUnhealthy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *UpstreamHealthCheck) DeepCopyInto(out *UpstreamHealthCheck) {
	*out = *in
	if in.Active != nil {
		in, out := &in.Active, &out.Active
		*out = new(UpstreamActiveHealthCheck)
		(*in).DeepCopyInto(*out)
	}
	if in.Passive != nil {
		in, out := &in.Passive, &out.Passive
		*out = new(UpstreamPassiveHealthCheck)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new UpstreamHealthCheck.
func (in *UpstreamHealthCheck) DeepCopy() *UpstreamHealthCheck {
	if in == nil {
		return nil
	}
	out := new(UpstreamHealthCheck)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *UpstreamNode) DeepCopyInto(out *UpstreamNode) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new UpstreamNode.
func (in *UpstreamNode) DeepCopy() *UpstreamNode {
	if in == nil {
		return nil
	}
	out := new(UpstreamNode)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *UpstreamPassiveHealthCheck) DeepCopyInto(out *UpstreamPassiveHealthCheck) {
	*out = *in
	in.Healthy.DeepCopyInto(&out.Healthy)
	in.Unhealthy.DeepCopyInto(&out.Unhealthy)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new UpstreamPassiveHealthCheck.
func (in *UpstreamPassiveHealthCheck) DeepCopy() *UpstreamPassiveHealthCheck {
	if in == nil {
		return nil
	}
	out := new(UpstreamPassiveHealthCheck)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *UpstreamPassiveHealthCheckHealthy) DeepCopyInto(out *UpstreamPassiveHealthCheckHealthy) {
	*out = *in
	if in.HTTPStatuses != nil {
		in, out := &in.HTTPStatuses, &out.HTTPStatuses
		*out = make([]int, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new UpstreamPassiveHealthCheckHealthy.
func (in *UpstreamPassiveHealthCheckHealthy) DeepCopy() *UpstreamPassiveHealthCheckHealthy {
	if in == nil {
		return nil
	}
	out := new(UpstreamPassiveHealthCheckHealthy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *UpstreamPassiveHealthCheckUnhealthy) DeepCopyInto(out *UpstreamPassiveHealthCheckUnhealthy) {
	*out = *in
	if in.HTTPStatuses != nil {
		in, out := &in.HTTPStatuses, &out.HTTPStatuses
		*out = make([]int, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new UpstreamPassiveHealthCheckUnhealthy.
func (in *UpstreamPassiveHealthCheckUnhealthy) DeepCopy() *UpstreamPassiveHealthCheckUnhealthy {
	if in == nil {
		return nil
	}
	out := new(UpstreamPassiveHealthCheckUnhealthy)
	in.DeepCopyInto(out)
	return out
}
