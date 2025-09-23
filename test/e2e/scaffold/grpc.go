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
package scaffold

import (
	"context"
	_ "embed"
	"fmt"
	"strings"
	"time"

	"github.com/apache/apisix-ingress-controller/test/e2e/framework"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	pb "sigs.k8s.io/gateway-api/conformance/echo-basic/grpcechoserver"
)

type RequestMetadata struct {
	// The :authority pseudoheader to set on the outgoing request.
	Authority string

	// Outgoing metadata pairs to add to the request.
	Metadata map[string]string
}

type ExpectedResponse struct {
	EchoRequest      *pb.EchoRequest
	EchoTwoRequest   *pb.EchoRequest
	EchoThreeRequest *pb.EchoRequest

	RequestMetadata *RequestMetadata

	Headers map[string]string

	EchoResponse EchoResponse
}

type EchoResponse struct {
	Code     codes.Code
	Headers  *metadata.MD
	Trailers *metadata.MD
	Response *pb.EchoResponse
}

func (s *Scaffold) DeployGRPCBackend() {
	s.Framework.DeployGRPCBackend(framework.GRPCBackendOpts{
		KubectlOptions: s.kubectlOptions,
	})
}

func (s *Scaffold) RequestEchoBackend(exp ExpectedResponse) error {
	endpoint := s.apisixTunnels.HTTP.Endpoint()

	endpoint = strings.Replace(endpoint, "localhost", "127.0.0.1", 1)

	dialOpts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	if exp.RequestMetadata != nil && exp.RequestMetadata.Authority != "" {
		dialOpts = append(dialOpts, grpc.WithAuthority(exp.RequestMetadata.Authority))
	}
	conn, err := grpc.NewClient(endpoint, dialOpts...)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if exp.RequestMetadata != nil && len(exp.RequestMetadata.Metadata) > 0 {
		ctx = metadata.NewOutgoingContext(ctx, metadata.New(exp.RequestMetadata.Metadata))
	}

	var (
		resp = &EchoResponse{
			Headers:  &metadata.MD{},
			Trailers: &metadata.MD{},
		}
	)

	client := pb.NewGrpcEchoClient(conn)
	switch {
	case exp.EchoRequest != nil:
		resp.Response, err = client.Echo(ctx, exp.EchoRequest, grpc.Header(resp.Headers), grpc.Trailer(resp.Trailers))
	case exp.EchoTwoRequest != nil:
		resp.Response, err = client.EchoTwo(ctx, exp.EchoTwoRequest, grpc.Header(resp.Headers), grpc.Trailer(resp.Trailers))
	case exp.EchoThreeRequest != nil:
		resp.Response, err = client.EchoThree(ctx, exp.EchoThreeRequest, grpc.Header(resp.Headers), grpc.Trailer(resp.Trailers))
	}
	if err != nil {
		resp.Code = status.Code(err)
		fmt.Printf("RPC finished with error: %v\n", err)
	} else {
		resp.Code = codes.OK
	}
	if err := expectEchoResponses(&exp, resp); err != nil {
		return err
	}
	return nil
}

func expectEchoResponses(expected *ExpectedResponse, actual *EchoResponse) error {
	if expected.EchoResponse.Code != actual.Code {
		return fmt.Errorf("expected status code to be %s (%d), but got %s (%d)",
			expected.EchoResponse.Code.String(),
			expected.EchoResponse.Code,
			actual.Code.String(),
			actual.Code,
		)
	}
	if expected.EchoResponse.Headers != nil {
		for key, values := range *expected.EchoResponse.Headers {
			actualValues := actual.Headers.Get(key)
			if len(values) != len(actualValues) {
				return fmt.Errorf("expected header %q to have %d values, but got %d", key, len(values), len(actualValues))
			}
			for i, v := range values {
				if actualValues[i] != v {
					return fmt.Errorf("expected header %q to have value %q, but got %q", key, v, actualValues[i])
				}
			}
		}
	}
	if len(expected.Headers) > 0 {
		msgHeaders := actual.Response.GetAssertions().GetHeaders()

		kv := make(map[string]string)
		for _, header := range msgHeaders {
			kv[header.GetKey()] = header.GetValue()
		}
		for key, value := range expected.Headers {
			actualValue, ok := kv[strings.ToLower(key)]
			if !ok {
				if value != "" {
					return fmt.Errorf("expected header %q to be present, but not found", key)
				}
				continue
			}
			if actualValue != value {
				return fmt.Errorf("expected header %q to be %q, but got %q", key, value, actualValue)
			}
		}
	}
	return nil
}
