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

provider:
  type: "apisix"                        # Provider type.
                                        # Value can be "apisix" or "apisix-standalone".

  sync_period: 1h                       # The period between two consecutive syncs.
                                        # The default value is 1 hour, which means the controller will not sync.
                                        # If you want to enable the sync, set it to a positive value.
  init_sync_delay: 20m                  # The initial delay before the first sync, only used when the controller is started.
                                        # The default value is 20 minutes.
```
