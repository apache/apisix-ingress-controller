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
default: help

VERSION ?= 0.0.0
GITSHA ?= $(shell git rev-parse --short=7 HEAD)
OSNAME ?= $(shell uname -s | tr A-Z a-z)
OSARCH ?= $(shell uname -m | tr A-Z a-z)
ifeq ($(OSARCH), x86_64)
	OSARCH = amd64
endif

VERSYM="github.com/api7/ingress-controller/pkg/version._buildVersion"
GITSHASYM="github.com/api7/ingress-controller/pkg/version._buildGitRevision"
BUILDOSSYM="github.com/api7/ingress-controller/pkg/version._buildOS"
GO_LDFLAGS ?= "-X=$(VERSYM)=$(VERSION) -X=$(GITSHASYM)=$(GITSHA) -X=$(BUILDOSSYM)=$(OSNAME)/$(OSARCH)"

### build:            Build apisix-ingress-controller
build:
	go build \
		-o apisix-ingress-controller \
		-ldflags $(GO_LDFLAGS) \
		main.go

### lint:             Do static lint check
lint:
	golangci-lint run

### license-check:    Do Apache License Header check
license-check:
ifeq ("$(wildcard .actions/openwhisk-utilities/scancode/scanCode.py)", "")
	git clone https://github.com/apache/openwhisk-utilities.git .actions/openwhisk-utilities
	cp .actions/ASF* .actions/openwhisk-utilities/scancode/
endif
	.actions/openwhisk-utilities/scancode/scanCode.py --config .actions/ASF-Release.cfg ./

### help:             Show Makefile rules
help:
	@echo Makefile rules:
	@echo
	@grep -E '^### [-A-Za-z0-9_]+:' Makefile | sed 's/###/   /'

.PHONY: build lint help
