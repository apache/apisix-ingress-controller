// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package upstream_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/pkg/providers/ingress/translation/annotations"
	"github.com/apache/apisix-ingress-controller/pkg/providers/ingress/translation/annotations/upstream"
)

func TestIPRestrictionHandler(t *testing.T) {
	anno := map[string]string{
		annotations.AnnotationsUpstreamScheme: "grpcs",
	}
	u := upstream.NewParser()

	out, err := u.Parse(annotations.NewExtractor(anno))
	ups, ok := out.(upstream.Upstream)
	if !ok {
		t.Fatalf("could not parse upstream")
	}
	assert.Nil(t, err, "checking given error")
	assert.Equal(t, "grpcs", ups.Scheme)

	anno[annotations.AnnotationsUpstreamScheme] = "gRPC"
	out, err = u.Parse(annotations.NewExtractor(anno))
	ups, ok = out.(upstream.Upstream)
	if !ok {
		t.Fatalf("could not parse upstream")
	}
	assert.Nil(t, err, "checking given error")
	assert.Equal(t, "grpc", ups.Scheme)

	anno[annotations.AnnotationsUpstreamScheme] = "nothing"
	out, err = u.Parse(annotations.NewExtractor(anno))
	assert.NotNil(t, err, "checking given error")
	assert.Nil(t, out, "checking given output")
}

func TestRetryParsing(t *testing.T) {
	anno := map[string]string{
		annotations.AnnotationsUpstreamRetry: "2",
	}
	u := upstream.NewParser()
	out, err := u.Parse(annotations.NewExtractor(anno))
	if err != nil {
		t.Fatalf(err.Error())
	}
	ups, ok := out.(upstream.Upstream)
	if !ok {
		t.Fatalf("could not parse upstream")
	}
	assert.Nil(t, err, "checking given error")
	assert.Equal(t, 2, ups.Retry)

	anno[annotations.AnnotationsUpstreamRetry] = "asdf"
	out, err = u.Parse(annotations.NewExtractor(anno))
	assert.NotNil(t, err, "checking given error")
}

func TestTimeoutParsing(t *testing.T) {
	anno := map[string]string{
		annotations.AnnotationsUpstreamTimeoutConnect: "2s",
		annotations.AnnotationsUpstreamTimeoutRead:    "3s",
		annotations.AnnotationsUpstreamTimeoutSend:    "4s",
	}
	u := upstream.NewParser()
	out, err := u.Parse(annotations.NewExtractor(anno))
	if err != nil {
		t.Fatalf(err.Error())
	}
	ups, ok := out.(upstream.Upstream)
	if !ok {
		t.Fatalf("could not parse upstream")
	}
	assert.Nil(t, err, "checking given error")
	assert.Equal(t, 2, ups.TimeoutConnect)
	assert.Equal(t, 3, ups.TimeoutRead)
	assert.Equal(t, 4, ups.TimeoutSend)
	anno[annotations.AnnotationsUpstreamRetry] = "asdf"
	out, err = u.Parse(annotations.NewExtractor(anno))
	assert.NotNil(t, err, "checking given error")
}
