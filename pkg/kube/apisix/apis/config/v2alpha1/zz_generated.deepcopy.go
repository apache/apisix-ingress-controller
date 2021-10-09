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

package v2alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApisixClusterAdminConfig) DeepCopyInto(out *ApisixClusterAdminConfig) {
	*out = *in
	out.ClientTimeout = in.ClientTimeout
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApisixClusterAdminConfig.
func (in *ApisixClusterAdminConfig) DeepCopy() *ApisixClusterAdminConfig {
	if in == nil {
		return nil
	}
	out := new(ApisixClusterAdminConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApisixClusterConfig) DeepCopyInto(out *ApisixClusterConfig) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApisixClusterConfig.
func (in *ApisixClusterConfig) DeepCopy() *ApisixClusterConfig {
	if in == nil {
		return nil
	}
	out := new(ApisixClusterConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ApisixClusterConfig) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApisixClusterConfigList) DeepCopyInto(out *ApisixClusterConfigList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ApisixClusterConfig, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApisixClusterConfigList.
func (in *ApisixClusterConfigList) DeepCopy() *ApisixClusterConfigList {
	if in == nil {
		return nil
	}
	out := new(ApisixClusterConfigList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ApisixClusterConfigList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApisixClusterConfigSpec) DeepCopyInto(out *ApisixClusterConfigSpec) {
	*out = *in
	if in.Monitoring != nil {
		in, out := &in.Monitoring, &out.Monitoring
		*out = new(ApisixClusterMonitoringConfig)
		**out = **in
	}
	if in.Admin != nil {
		in, out := &in.Admin, &out.Admin
		*out = new(ApisixClusterAdminConfig)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApisixClusterConfigSpec.
func (in *ApisixClusterConfigSpec) DeepCopy() *ApisixClusterConfigSpec {
	if in == nil {
		return nil
	}
	out := new(ApisixClusterConfigSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApisixClusterMonitoringConfig) DeepCopyInto(out *ApisixClusterMonitoringConfig) {
	*out = *in
	out.Prometheus = in.Prometheus
	out.Skywalking = in.Skywalking
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApisixClusterMonitoringConfig.
func (in *ApisixClusterMonitoringConfig) DeepCopy() *ApisixClusterMonitoringConfig {
	if in == nil {
		return nil
	}
	out := new(ApisixClusterMonitoringConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApisixClusterPrometheusConfig) DeepCopyInto(out *ApisixClusterPrometheusConfig) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApisixClusterPrometheusConfig.
func (in *ApisixClusterPrometheusConfig) DeepCopy() *ApisixClusterPrometheusConfig {
	if in == nil {
		return nil
	}
	out := new(ApisixClusterPrometheusConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApisixClusterSkywalkingConfig) DeepCopyInto(out *ApisixClusterSkywalkingConfig) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApisixClusterSkywalkingConfig.
func (in *ApisixClusterSkywalkingConfig) DeepCopy() *ApisixClusterSkywalkingConfig {
	if in == nil {
		return nil
	}
	out := new(ApisixClusterSkywalkingConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApisixConsumer) DeepCopyInto(out *ApisixConsumer) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApisixConsumer.
func (in *ApisixConsumer) DeepCopy() *ApisixConsumer {
	if in == nil {
		return nil
	}
	out := new(ApisixConsumer)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ApisixConsumer) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApisixConsumerAuthParameter) DeepCopyInto(out *ApisixConsumerAuthParameter) {
	*out = *in
	if in.BasicAuth != nil {
		in, out := &in.BasicAuth, &out.BasicAuth
		*out = new(ApisixConsumerBasicAuth)
		(*in).DeepCopyInto(*out)
	}
	if in.KeyAuth != nil {
		in, out := &in.KeyAuth, &out.KeyAuth
		*out = new(ApisixConsumerKeyAuth)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApisixConsumerAuthParameter.
func (in *ApisixConsumerAuthParameter) DeepCopy() *ApisixConsumerAuthParameter {
	if in == nil {
		return nil
	}
	out := new(ApisixConsumerAuthParameter)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApisixConsumerBasicAuth) DeepCopyInto(out *ApisixConsumerBasicAuth) {
	*out = *in
	if in.SecretRef != nil {
		in, out := &in.SecretRef, &out.SecretRef
		*out = new(v1.LocalObjectReference)
		**out = **in
	}
	if in.Value != nil {
		in, out := &in.Value, &out.Value
		*out = new(ApisixConsumerBasicAuthValue)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApisixConsumerBasicAuth.
func (in *ApisixConsumerBasicAuth) DeepCopy() *ApisixConsumerBasicAuth {
	if in == nil {
		return nil
	}
	out := new(ApisixConsumerBasicAuth)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApisixConsumerBasicAuthValue) DeepCopyInto(out *ApisixConsumerBasicAuthValue) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApisixConsumerBasicAuthValue.
func (in *ApisixConsumerBasicAuthValue) DeepCopy() *ApisixConsumerBasicAuthValue {
	if in == nil {
		return nil
	}
	out := new(ApisixConsumerBasicAuthValue)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApisixConsumerKeyAuth) DeepCopyInto(out *ApisixConsumerKeyAuth) {
	*out = *in
	if in.SecretRef != nil {
		in, out := &in.SecretRef, &out.SecretRef
		*out = new(v1.LocalObjectReference)
		**out = **in
	}
	if in.Value != nil {
		in, out := &in.Value, &out.Value
		*out = new(ApisixConsumerKeyAuthValue)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApisixConsumerKeyAuth.
func (in *ApisixConsumerKeyAuth) DeepCopy() *ApisixConsumerKeyAuth {
	if in == nil {
		return nil
	}
	out := new(ApisixConsumerKeyAuth)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApisixConsumerKeyAuthValue) DeepCopyInto(out *ApisixConsumerKeyAuthValue) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApisixConsumerKeyAuthValue.
func (in *ApisixConsumerKeyAuthValue) DeepCopy() *ApisixConsumerKeyAuthValue {
	if in == nil {
		return nil
	}
	out := new(ApisixConsumerKeyAuthValue)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApisixConsumerList) DeepCopyInto(out *ApisixConsumerList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ApisixConsumer, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApisixConsumerList.
func (in *ApisixConsumerList) DeepCopy() *ApisixConsumerList {
	if in == nil {
		return nil
	}
	out := new(ApisixConsumerList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ApisixConsumerList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApisixConsumerSpec) DeepCopyInto(out *ApisixConsumerSpec) {
	*out = *in
	in.AuthParameter.DeepCopyInto(&out.AuthParameter)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApisixConsumerSpec.
func (in *ApisixConsumerSpec) DeepCopy() *ApisixConsumerSpec {
	if in == nil {
		return nil
	}
	out := new(ApisixConsumerSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApisixRoute) DeepCopyInto(out *ApisixRoute) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	if in.Spec != nil {
		in, out := &in.Spec, &out.Spec
		*out = new(ApisixRouteSpec)
		(*in).DeepCopyInto(*out)
	}
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApisixRoute.
func (in *ApisixRoute) DeepCopy() *ApisixRoute {
	if in == nil {
		return nil
	}
	out := new(ApisixRoute)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ApisixRoute) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApisixRouteAuthentication) DeepCopyInto(out *ApisixRouteAuthentication) {
	*out = *in
	out.KeyAuth = in.KeyAuth
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApisixRouteAuthentication.
func (in *ApisixRouteAuthentication) DeepCopy() *ApisixRouteAuthentication {
	if in == nil {
		return nil
	}
	out := new(ApisixRouteAuthentication)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApisixRouteAuthenticationKeyAuth) DeepCopyInto(out *ApisixRouteAuthenticationKeyAuth) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApisixRouteAuthenticationKeyAuth.
func (in *ApisixRouteAuthenticationKeyAuth) DeepCopy() *ApisixRouteAuthenticationKeyAuth {
	if in == nil {
		return nil
	}
	out := new(ApisixRouteAuthenticationKeyAuth)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApisixRouteHTTP) DeepCopyInto(out *ApisixRouteHTTP) {
	*out = *in
	if in.Timeout != nil {
		in, out := &in.Timeout, &out.Timeout
		*out = new(UpstreamTimeout)
		**out = **in
	}
	if in.Match != nil {
		in, out := &in.Match, &out.Match
		*out = new(ApisixRouteHTTPMatch)
		(*in).DeepCopyInto(*out)
	}
	if in.Backend != nil {
		in, out := &in.Backend, &out.Backend
		*out = new(ApisixRouteHTTPBackend)
		(*in).DeepCopyInto(*out)
	}
	if in.Backends != nil {
		in, out := &in.Backends, &out.Backends
		*out = make([]*ApisixRouteHTTPBackend, len(*in))
		for i := range *in {
			if (*in)[i] != nil {
				in, out := &(*in)[i], &(*out)[i]
				*out = new(ApisixRouteHTTPBackend)
				(*in).DeepCopyInto(*out)
			}
		}
	}
	if in.Plugins != nil {
		in, out := &in.Plugins, &out.Plugins
		*out = make([]*ApisixRouteHTTPPlugin, len(*in))
		for i := range *in {
			if (*in)[i] != nil {
				in, out := &(*in)[i], &(*out)[i]
				*out = new(ApisixRouteHTTPPlugin)
				(*in).DeepCopyInto(*out)
			}
		}
	}
	if in.Authentication != nil {
		in, out := &in.Authentication, &out.Authentication
		*out = new(ApisixRouteAuthentication)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApisixRouteHTTP.
func (in *ApisixRouteHTTP) DeepCopy() *ApisixRouteHTTP {
	if in == nil {
		return nil
	}
	out := new(ApisixRouteHTTP)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApisixRouteHTTPBackend) DeepCopyInto(out *ApisixRouteHTTPBackend) {
	*out = *in
	out.ServicePort = in.ServicePort
	if in.Weight != nil {
		in, out := &in.Weight, &out.Weight
		*out = new(int)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApisixRouteHTTPBackend.
func (in *ApisixRouteHTTPBackend) DeepCopy() *ApisixRouteHTTPBackend {
	if in == nil {
		return nil
	}
	out := new(ApisixRouteHTTPBackend)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApisixRouteHTTPMatch) DeepCopyInto(out *ApisixRouteHTTPMatch) {
	*out = *in
	if in.Paths != nil {
		in, out := &in.Paths, &out.Paths
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Methods != nil {
		in, out := &in.Methods, &out.Methods
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Hosts != nil {
		in, out := &in.Hosts, &out.Hosts
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.RemoteAddrs != nil {
		in, out := &in.RemoteAddrs, &out.RemoteAddrs
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.NginxVars != nil {
		in, out := &in.NginxVars, &out.NginxVars
		*out = make([]ApisixRouteHTTPMatchExpr, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApisixRouteHTTPMatch.
func (in *ApisixRouteHTTPMatch) DeepCopy() *ApisixRouteHTTPMatch {
	if in == nil {
		return nil
	}
	out := new(ApisixRouteHTTPMatch)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApisixRouteHTTPMatchExpr) DeepCopyInto(out *ApisixRouteHTTPMatchExpr) {
	*out = *in
	out.Subject = in.Subject
	if in.Set != nil {
		in, out := &in.Set, &out.Set
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Value != nil {
		in, out := &in.Value, &out.Value
		*out = new(string)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApisixRouteHTTPMatchExpr.
func (in *ApisixRouteHTTPMatchExpr) DeepCopy() *ApisixRouteHTTPMatchExpr {
	if in == nil {
		return nil
	}
	out := new(ApisixRouteHTTPMatchExpr)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApisixRouteHTTPMatchExprSubject) DeepCopyInto(out *ApisixRouteHTTPMatchExprSubject) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApisixRouteHTTPMatchExprSubject.
func (in *ApisixRouteHTTPMatchExprSubject) DeepCopy() *ApisixRouteHTTPMatchExprSubject {
	if in == nil {
		return nil
	}
	out := new(ApisixRouteHTTPMatchExprSubject)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApisixRouteHTTPPlugin) DeepCopyInto(out *ApisixRouteHTTPPlugin) {
	*out = *in
	in.Config.DeepCopyInto(&out.Config)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApisixRouteHTTPPlugin.
func (in *ApisixRouteHTTPPlugin) DeepCopy() *ApisixRouteHTTPPlugin {
	if in == nil {
		return nil
	}
	out := new(ApisixRouteHTTPPlugin)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApisixRouteList) DeepCopyInto(out *ApisixRouteList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ApisixRoute, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApisixRouteList.
func (in *ApisixRouteList) DeepCopy() *ApisixRouteList {
	if in == nil {
		return nil
	}
	out := new(ApisixRouteList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ApisixRouteList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApisixRouteSpec) DeepCopyInto(out *ApisixRouteSpec) {
	*out = *in
	if in.HTTP != nil {
		in, out := &in.HTTP, &out.HTTP
		*out = make([]*ApisixRouteHTTP, len(*in))
		for i := range *in {
			if (*in)[i] != nil {
				in, out := &(*in)[i], &(*out)[i]
				*out = new(ApisixRouteHTTP)
				(*in).DeepCopyInto(*out)
			}
		}
	}
	if in.TCP != nil {
		in, out := &in.TCP, &out.TCP
		*out = make([]*ApisixRouteTCP, len(*in))
		for i := range *in {
			if (*in)[i] != nil {
				in, out := &(*in)[i], &(*out)[i]
				*out = new(ApisixRouteTCP)
				**out = **in
			}
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApisixRouteSpec.
func (in *ApisixRouteSpec) DeepCopy() *ApisixRouteSpec {
	if in == nil {
		return nil
	}
	out := new(ApisixRouteSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApisixRouteTCP) DeepCopyInto(out *ApisixRouteTCP) {
	*out = *in
	out.Match = in.Match
	out.Backend = in.Backend
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApisixRouteTCP.
func (in *ApisixRouteTCP) DeepCopy() *ApisixRouteTCP {
	if in == nil {
		return nil
	}
	out := new(ApisixRouteTCP)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApisixRouteTCPBackend) DeepCopyInto(out *ApisixRouteTCPBackend) {
	*out = *in
	out.ServicePort = in.ServicePort
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApisixRouteTCPBackend.
func (in *ApisixRouteTCPBackend) DeepCopy() *ApisixRouteTCPBackend {
	if in == nil {
		return nil
	}
	out := new(ApisixRouteTCPBackend)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApisixRouteTCPMatch) DeepCopyInto(out *ApisixRouteTCPMatch) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApisixRouteTCPMatch.
func (in *ApisixRouteTCPMatch) DeepCopy() *ApisixRouteTCPMatch {
	if in == nil {
		return nil
	}
	out := new(ApisixRouteTCPMatch)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ApisixStatus) DeepCopyInto(out *ApisixStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = new([]metav1.Condition)
		if **in != nil {
			in, out := *in, *out
			*out = make([]metav1.Condition, len(*in))
			for i := range *in {
				(*in)[i].DeepCopyInto(&(*out)[i])
			}
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApisixStatus.
func (in *ApisixStatus) DeepCopy() *ApisixStatus {
	if in == nil {
		return nil
	}
	out := new(ApisixStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *UpstreamTimeout) DeepCopyInto(out *UpstreamTimeout) {
	*out = *in
	out.Connect = in.Connect
	out.Send = in.Send
	out.Read = in.Read
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new UpstreamTimeout.
func (in *UpstreamTimeout) DeepCopy() *UpstreamTimeout {
	if in == nil {
		return nil
	}
	out := new(UpstreamTimeout)
	in.DeepCopyInto(out)
	return out
}
