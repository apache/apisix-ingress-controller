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

package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math/big"
	"time"
)

func main() {
	generateCertificateAndKey("../test/e2e/testbackend/tls/cert1.pem", "../test/e2e/testbackend/tls/key1.pem")
	generateCertificateAndKey("../test/e2e/testbackend/tls/cert2.pem", "../test/e2e/testbackend/tls/key2.pem")
	generateCertificateAndKey("../test/e2e/testbackend/tls/cert3.pem", "../test/e2e/testbackend/tls/key3.pem")
	generateCertificateAndKey("../test/e2e/testbackend/tls/certUpdate1.pem", "../test/e2e/testbackend/tls/keyUpdate1.pem")
	generateCertificateAndKey("../test/e2e/testbackend/tls/certUpdate2.pem", "../test/e2e/testbackend/tls/keyUpdate2.pem")
	generateCertificateAndKey("../test/e2e/testbackend/tls/certUpdate3.pem", "../test/e2e/testbackend/tls/keyUpdate3.pem")
}

func generateCertificateAndKey(certFilename, keyFilename string) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		panic(err)
	}

	template := x509.Certificate{
		SerialNumber:          big.NewInt(time.Now().Unix()),
		Subject:               pkix.Name{CommonName: "api6.com"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(30, 0, 0),
		BasicConstraintsValid: true,
		DNSNames:              []string{"api6.com", "*.api6.com"},
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	certificateBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		panic(err)
	}

	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	certificatePEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certificateBytes})
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: privateKeyBytes})

	err = ioutil.WriteFile(certFilename, certificatePEM, 0644)
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile(keyFilename, privateKeyPEM, 0600)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Certificate and private key generated successfully for file located at: %s and %s!\n", certFilename, keyFilename)

}
