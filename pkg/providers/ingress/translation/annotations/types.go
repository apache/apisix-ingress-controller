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
package annotations

import (
	"strings"
)

const (
	// AnnotationsPrefix is the apisix annotation prefix
	AnnotationsPrefix = "k8s.apisix.apache.org/"
)

// Extractor encapsulates some auxiliary methods to extract annotations.
type Extractor interface {
	// GetStringAnnotation returns the string value of the target annotation.
	// When the target annoatation is missing, empty string will be given.
	GetStringAnnotation(string) string
	// GetStringsAnnotation returns a string slice which splits the value of target
	// annotation by the comma symbol. When the target annotation is missing, a nil
	// slice will be given.
	GetStringsAnnotation(string) []string
	// GetBoolAnnotation returns a boolean value from the given annotation.
	// When value is "true", true will be given, other values will be treated as
	// false.
	GetBoolAnnotation(string) bool
}

// Handler abstracts the behavior so that the apisix-ingress-controller knows
// how to parse some annotations and convert them to APISIX plugins.
type Handler interface {
	// Handle parses the target annotation and converts it to the type-agnostic structure.
	// The return value might be nil since some features have an explicit switch, users should
	// judge whether Handle is failed by the second error value.
	Handle(Extractor) (interface{}, error)
	// PluginName returns a string which indicates the target plugin name in APISIX.
	PluginName() string
}

type extractor struct {
	annotations map[string]string
}

func (e *extractor) GetStringAnnotation(name string) string {
	return e.annotations[name]
}

func (e *extractor) GetStringsAnnotation(name string) []string {
	value := e.GetStringAnnotation(name)
	if value == "" {
		return nil
	}
	return strings.Split(e.annotations[name], ",")
}

func (e *extractor) GetBoolAnnotation(name string) bool {
	return e.annotations[name] == "true"
}

// NewExtractor creates an annotations extractor.
func NewExtractor(annotations map[string]string) Extractor {
	return &extractor{
		annotations: annotations,
	}
}
