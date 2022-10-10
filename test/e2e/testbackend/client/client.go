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
package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	hwpb "google.golang.org/grpc/examples/helloworld/helloworld"
)

// RequestHello request grpc method in addr use specific ca
func RequestHello(addr string, ca []byte) error {
	var (
		creds    credentials.TransportCredentials
		tlsConf  tls.Config
		certPool = x509.NewCertPool()
	)
	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		return errors.New("failed to append ca certs")
	}
	tlsConf.RootCAs = certPool
	creds = credentials.NewTLS(&tlsConf)
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(creds), grpc.WithAuthority("e2e.apisix.local"))
	if err != nil {
		return err
	}
	defer conn.Close()
	c := hwpb.NewGreeterClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := c.SayHello(ctx, &hwpb.HelloRequest{Name: "apisix-ingress"})
	if err != nil {
		return err
	}
	log.Printf("Get response %s", r.GetMessage())
	return nil
}
