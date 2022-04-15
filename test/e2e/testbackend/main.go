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

package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	ecpb "google.golang.org/grpc/examples/features/proto/echo"
	hwpb "google.golang.org/grpc/examples/helloworld/helloworld"
	"google.golang.org/grpc/reflection"
)

// hwServer is used to implement helloworld.GreeterServer.
type hwServer struct {
	hwpb.UnimplementedGreeterServer
}

// SayHello implements helloworld.GreeterServer
func (s *hwServer) SayHello(ctx context.Context, in *hwpb.HelloRequest) (*hwpb.HelloReply, error) {
	return &hwpb.HelloReply{Message: "Hello " + in.Name}, nil
}

type ecServer struct {
	ecpb.UnimplementedEchoServer
}

func (s *ecServer) UnaryEcho(ctx context.Context, req *ecpb.EchoRequest) (*ecpb.EchoResponse, error) {
	return &ecpb.EchoResponse{Message: req.Message}, nil
}

func hello(w http.ResponseWriter, req *http.Request) {
	log.Printf("%s %s%s from %v", req.Method, req.Host, req.RequestURI, req.RemoteAddr)
	_, _ = fmt.Fprintf(w, "Hello World\n")
}

func newDefaultCACertPool() *x509.CertPool {
	caCert, err := ioutil.ReadFile("tls/ca.pem")
	if err != nil {
		log.Fatalf("failed to load ca cert: %v", err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)
	return caCertPool
}

func loadTLSCredentials(clientAuth bool) credentials.TransportCredentials {
	serverCert, err := tls.LoadX509KeyPair("tls/server.pem", "tls/server.key")
	if err != nil {
		log.Fatalf("failed to load server cert: %v", err)
	}
	var config *tls.Config
	if clientAuth {
		// Create the credentials and return it
		config = &tls.Config{
			Certificates: []tls.Certificate{serverCert},
			ClientCAs:    newDefaultCACertPool(),
			ClientAuth:   tls.RequireAndVerifyClientCert,
		}
	} else {
		// Create the credentials and return it
		config = &tls.Config{
			Certificates: []tls.Certificate{serverCert},
			ClientAuth:   tls.NoClientCert,
		}
	}
	return credentials.NewTLS(config)
}

func main() {
	http.HandleFunc("/hello", hello)

	go func() {
		// curl http://e2e.apisix.local:80/hello --resolve e2e.apisix.local:80:127.0.0.1
		log.Printf("starting http server in 80")
		log.Fatalln(http.ListenAndServe(":80", nil))
	}()

	go func() {
		// curl https://e2e.apisix.local:443/hello --resolve e2e.apisix.local:443:127.0.0.1 --cacert ca.pem
		log.Printf("starting https server in 443")
		log.Fatalln(http.ListenAndServeTLS(":443", "tls/server.pem", "tls/server.key", nil))
	}()

	go func() {
		// curl https://e2e.apisix.local:8443/hello --resolve e2e.apisix.local:8443:127.0.0.1 --cacert ca.pem --cert client.pem --key client.key
		log.Printf("starting mtls http server in 8443")
		tlsConfig := &tls.Config{
			ClientCAs:  newDefaultCACertPool(),
			ClientAuth: tls.RequireAndVerifyClientCert,
		}
		server := &http.Server{
			Addr:      ":8443",
			TLSConfig: tlsConfig,
		}
		log.Fatalln(server.ListenAndServeTLS("tls/server.pem", "tls/server.key"))
	}()

	go func() {
		// grpcurl -plaintext -d '{"name": "apisix-ingress"}' 127.0.0.1:50051 helloworld.Greeter/SayHello
		lis, err := net.Listen("tcp", ":50051")
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
		log.Printf("starting grpc server in %v\n", lis.Addr())
		s := grpc.NewServer()

		hwpb.RegisterGreeterServer(s, &hwServer{})
		ecpb.RegisterEchoServer(s, &ecServer{})
		reflection.Register(s)

		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	go func() {
		// grpcurl -cacert ca.pem -servername e2e.apisix.local  -d '{"name": "apisix-ingress"}' 127.0.0.1:50052 helloworld.Greeter/SayHello
		lis, err := net.Listen("tcp", ":50052")
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}

		log.Printf("starting grpcs server in %v\n", lis.Addr())
		s := grpc.NewServer(
			grpc.Creds(loadTLSCredentials(false)))

		hwpb.RegisterGreeterServer(s, &hwServer{})
		ecpb.RegisterEchoServer(s, &ecServer{})
		reflection.Register(s)

		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	go func() {
		// grpcurl -key client.key -cert client.pem -cacert ca.pem -servername e2e.apisix.local  -d '{"name": "apisix-ingress"}' 127.0.0.1:50053 helloworld.Greeter/SayHello
		lis, err := net.Listen("tcp", ":50053")
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
		log.Printf("starting mtls grpc server in %v\n", lis.Addr())
		s := grpc.NewServer(
			grpc.Creds(loadTLSCredentials(true)))

		hwpb.RegisterGreeterServer(s, &hwServer{})
		ecpb.RegisterEchoServer(s, &ecServer{})
		reflection.Register(s)

		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigCh
	log.Printf("signal %d (%s) received\n", sig, sig.String())
}
