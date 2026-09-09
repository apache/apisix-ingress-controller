---
title: Configuration File
slug: /reference/apisix-ingress-controller/configuration-file
description: Configure the APISIX Ingress Controller using the config.yaml file, including configurations such as log settings, leader election, metrics, and sync behavior.
---

<!--
#
# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
-->

The APISIX Ingress Controller uses a configuration file `config.yaml` to define core settings such as log level, leader election behavior, metrics endpoints, and sync intervals.

Configurations are defined in a Kubernetes ConfigMap and mounted into the controller pod as a file at runtime. To apply changes, you can update the ConfigMap and restart the controller Deployment to reload the configurations.

Below are all available configuration options, including their default values and usage:

```yaml
log_level: "info"                               # The log level of the APISIX Ingress Controller.
                                                # The default value is "info".

controller_name: apisix.apache.org/apisix-ingress-controller  # The controller name of the APISIX Ingress Controller,
                                                              # which is used to identify the controller in the GatewayClass.
                                                              # The default value is "apisix.apache.org/apisix-ingress-controller".
leader_election_id: "apisix-ingress-controller-leader"        # The leader election ID for the APISIX Ingress Controller.
                                                              # The default value is "apisix-ingress-controller-leader".
leader_election:
  lease_duration: 30s                   # lease_duration is the duration that non-leader candidates will wait
                                        # after observing a leadership renewal until attempting to acquire leadership of a
                                        # leader election.
  renew_deadline: 20s                   # renew_deadline is the time in seconds that the acting controller
                                        # will retry refreshing leadership before giving up.
  retry_period: 2s                      # retry_period is the time in seconds that the acting controller
                                        # will wait between tries of actions with the controller.
  disable: false                        # Whether to disable leader election.

metrics_addr: ":8080"                   # The address the metrics endpoint binds to.
                                        # The default value is ":8080".

enable_http2: false                     # Whether to enable HTTP/2 for the server.
                                        # The default value is false.

probe_addr: ":8081"                     # The address the probe endpoint binds to.
                                        # The default value is ":8081".

secure_metrics: false                   # The secure metrics configuration.
                                        # The default value is "" (empty).

exec_adc_timeout: 15s                   # The timeout for the ADC to execute.
                                        # The default value is 15 seconds.

listener_port_match_mode: "off"         # Mode for injecting server_port route vars from Gateway listener ports.
                                        # - "off": never inject server_port vars.
                                        # - "auto": inject when parentRefs explicitly target listeners (sectionName/port) or when multiple listener ports are matched.
                                        # - "explicit": inject only when parentRefs explicitly target listeners.
                                        # The default value is "off". APISIX matches server_port against the port it
                                        # accepted the connection on, which is not the port the Gateway listener
                                        # declares, so only enable this when APISIX listens on the declared ports.

provider:
  type: "apisix"                        # Provider type.
                                        # Value can be "apisix" or "apisix-standalone".

  sync_period: 1h                       # The period between two consecutive syncs.
                                        # The default value is 1 hour, which means the controller will not sync.
                                        # If you want to enable the sync, set it to a positive value.
  init_sync_delay: 20m                  # The initial delay before the first sync, only used when the controller is started.
                                        # The default value is 20 minutes.
```

## Listener port matching

`listener_port_match_mode` turns a Gateway listener port into a `server_port` route variable. It is the only mechanism that binds a route to a specific listener port, so it is also what decides whether a route attached to an HTTPS listener can be reached over plaintext HTTP on the same host and path. Two preconditions decide whether it takes effect.

**The Gateway listener port must equal the port APISIX listens on.** APISIX evaluates `server_port` against the port it accepted the connection on, not the port the Gateway declares. If the Gateway declares `443` while APISIX listens on `9443` behind a Service that maps `443` to `9443`, the injected predicate is `server_port == 443` and it matches nothing, so the route stops serving on both protocols. The controller cannot open data plane ports, so declaring the port APISIX actually listens on is the only working configuration. Verify with a real request after changing the mode; a successful Helm upgrade and a ready pod do not prove the route still matches.

**Listeners that set a hostname are never pinned to a port.** A listener with `spec.listeners[].hostname` is treated as isolated by host, so it contributes no port to the predicate. If every listener a route attaches to sets a hostname, no `server_port` variable is emitted at all, whatever the mode. A route attached to such an HTTPS listener therefore still matches requests arriving on the plaintext port, because host matching alone does not distinguish the two. To pin the route to a port, leave `hostname` unset on the listener and select the host on the route instead.

With the default `off`, no `server_port` variable is emitted at all, and a route attached only to an HTTPS listener is reachable over HTTP on the same host and path whenever the data plane also serves plaintext. Use an HTTPS redirect filter, or separate the two protocols by hostname, if that is not acceptable.
