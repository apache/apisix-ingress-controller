<!--
  Licensed to the Apache Software Foundation (ASF) under one
  or more contributor license agreements.  See the NOTICE file
  distributed with this work for additional information
  regarding copyright ownership.  The ASF licenses this file
  to you under the Apache License, Version 2.0 (the
  "License"); you may not use this file except in compliance
  with the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing,
  software distributed under the License is distributed on an
  "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
  KIND, either express or implied.  See the License for the
  specific language governing permissions and limitations
  under the License.
-->

# Apache APISIX Ingress Controller — Agent Instructions

This file is read by automated agents (security scanners, code
analyzers, AI assistants) operating on this repository. It
points them at the human-authored references they should
consult before producing output.

## Security Model

This repository inherits the Apache APISIX project threat
model. The authoritative document lives at:

<https://github.com/apache/apisix/blob/master/docs/en/latest/security-threat-model.md>

The §4.2 component-family table in that document covers this
repository under the `apisix-ingress-controller` family.

Of particular relevance to this controller specifically: §4.8
includes a CRD-to-Admin-API fidelity invariant — silent drop,
injection, or rename between the Kubernetes `apisix.apache.org`
CRD spec and the Admin API target is a controller bug, not
operator misconfiguration. The §4.8 entry recommends an e2e
contract test enforcing this invariant.
