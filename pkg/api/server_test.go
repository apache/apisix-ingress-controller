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
package api

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/pkg/config"
)

const (
	certFileName = "cert.pem"
	_tlsCert     = `-----BEGIN CERTIFICATE-----
MIIDBTCCAe0CFHoW964zOGe29tXJwA4WWsrUxyggMA0GCSqGSIb3DQEBCwUAMD8x
PTA7BgNVBAMMNGFwaXNpeC1pbmdyZXNzLWNvbnRyb2xsZXItd2ViaG9vay5pbmdy
ZXNzLWFwaXNpeC5zdmMwHhcNMjEwODIxMDgxNDQzWhcNMjIwODIxMDgxNDQzWjA/
MT0wOwYDVQQDDDRhcGlzaXgtaW5ncmVzcy1jb250cm9sbGVyLXdlYmhvb2suaW5n
cmVzcy1hcGlzaXguc3ZjMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA
2mMDnAkHbpmMPMgZHTh5VKnUXRrHKMY3OEzTyDs4MxBSrxBsIrRYXjXBi6A75IRU
XD9/W8DyIENclLRTrYdLt03OD8n5a2Z6+DW8XfAO0FZ058QnyKOo9v1/RKqHkPtV
PwbCjUvCCClsgihOSzxcgcF2oHm2x1JaATBicWNS4cze6LrkmVSI2BL/6liU9hSJ
15MtyNRqe18sQ/7z6cWZBkAfwW9pY4lC0JWNHntFdnQJzPlw/jM9rzHamnBrMas9
R2TDVqfgURqKmJaQBd0lDtc9Zrp2G9dCqmF8UP3OvH3cBa8UKBzo4kPRsjKEf5+r
zzMrwNG7kX47K82JZNhKlwIDAQABMA0GCSqGSIb3DQEBCwUAA4IBAQBBJ3881kMF
DaYXJ9xlIo0lWijt8yoDn5bGXrvT0Q+tLJhbFmVh9Mr+/NwaythKPM4dcXXWKlwN
Ham8OpqfFP2BZ93nv+CXgQxpdNAGQPNmJ3146o8sJpbnNwQCTcoe9nm66DTW6340
SCdDwuwkNRMsc24EnTdmwe7Z0XBgz+jx0WGlzxmeQKJQVUChp7w1qNiUfNWjK3Ud
hCUjmUwiqVpk9+I997a9/DNu6CEt7SIJK3nbuLWDuXa4S3ebMgVlCGXAapb5QfDe
S3BTAjguuygwbpo4M+S6hyObMpdNbr9dVhFLGj02lzL3a+mM1C19kJCpbJggu1Y3
oXDF4V2XHbzJ
-----END CERTIFICATE-----
`
	keyFileName = "key.pem"
	_tlsKey     = `-----BEGIN RSA PRIVATE KEY-----
MIIEpgIBAAKCAQEA2mMDnAkHbpmMPMgZHTh5VKnUXRrHKMY3OEzTyDs4MxBSrxBs
IrRYXjXBi6A75IRUXD9/W8DyIENclLRTrYdLt03OD8n5a2Z6+DW8XfAO0FZ058Qn
yKOo9v1/RKqHkPtVPwbCjUvCCClsgihOSzxcgcF2oHm2x1JaATBicWNS4cze6Lrk
mVSI2BL/6liU9hSJ15MtyNRqe18sQ/7z6cWZBkAfwW9pY4lC0JWNHntFdnQJzPlw
/jM9rzHamnBrMas9R2TDVqfgURqKmJaQBd0lDtc9Zrp2G9dCqmF8UP3OvH3cBa8U
KBzo4kPRsjKEf5+rzzMrwNG7kX47K82JZNhKlwIDAQABAoIBAQDQJY9LKU/sGm2P
gShusWTzTOsb0moAcuwuvQsdzVPDV8t3EDAA4+NV5+aRLifnpjjBs8OvsDcWiR20
nisjOdDw5TeB1P/lXcfWy2C+KA/2gnDqdgt1MIfa4cJrsB2GEgcuC0NjaNGG9fR2
GfSFwQJqqfpm+Zs8X0Fp4LPzXregfd//sgnNi5dorWxZ142lJvAStC/inEzLFBLW
hC+tDq9zIXUmAhlMzfmJ3cf8gU7z+RMOYkNFaz7EGM6wWZSppiWBk9A7BiknV5AJ
cQRv2woGy2ZgP7MXZVg8RNaX5w6P6GFEK5NbdoyHkGL2olvf8tN7f9oNLdv9apQf
6F3l7OABAoGBAP6sX+tSqs/oAouyZQ4v9NnrnhBKgPgnMwcKaohg4jo58TMJ5ldQ
U10AkZyfVcQ/gE7531N+6D/fzEYSwiiZdsOFVEMHQitIXIZMDeyU+EPoZawyHCpn
h6NuaStkXqowtEdkscJgiCRBNncnKwvCuLu8copoglfwPaaLMzrBilzNAoGBANuG
P6f3XLfvyDyVDM6oAbLVQGIfEBrSueyoLIackSe1a1mJ7pTmMnY9S/9W+i3ZR6Kp
tAKUnEkoN90l8R/1V0x7AobOhMWicblo23eAw9r6jXKZtUxlhbjNKYzfQRVetbT4
ix/qKdme1dXLAeM4YgF1CKxO1ccf6fOJArWpSwTzAoGBAOoux+U0ly2nQvACkzqA
jr71EtwYJpAKO7n1shDGRkEUlt8/8zfG/WE/7KYBPnS/j9UPoHS+9gIGYWjuRuve
cn9IUztvqUDzwWEc/pDWS5TmVtgJHC1CFlAKb1sfaI1HS/96cJs0+Pudm9/lfIfL
/uNjXlA32ePTXl2PEwSsg/bhAoGBAIthmss/8LvM4BsvG9merK1qXx2t0WDmiSws
v1Cc2kEXHFjWjgg2fLW8R6ORCvnPan9qNqQozW5ZvdaJP6bl9I7Xz4veVkjR0llB
rY8bz78atHKeC5G9KAFlKkuKeN1jrAWChXs3B2loQyciZUlqxDdeoqocx/lNVxLM
3E6RddNnAoGBAMCjs0qKwT5ENMsaQxFlwPEKuC5Sl0ejKgUnoHsVl9VuhAMcwE70
hMJMGXv2p1BbBuuW35jH92LBSBjS/Zv4b86DG2VQsDWNI4u3lPFd1zif6dhE8yvU
bKS1uxKukPFp6zxFwR7YZIiwo3tGkcudpHdTNurNMQiSTN97LTo8KL8y
-----END RSA PRIVATE KEY-----
`
)

func generateCertFiles() {
	_ = ioutil.WriteFile(certFileName, []byte(_tlsCert), 0644)
	_ = ioutil.WriteFile(keyFileName, []byte(_tlsKey), 0644)
}

func deleteCertFiles() {
	_ = os.Remove(certFileName)
	_ = os.Remove(keyFileName)
}

func TestServer(t *testing.T) {
	cfg := &config.Config{HTTPListen: "127.0.0.1:0"}
	_, err := NewServer(cfg)
	assert.Nil(t, err, "see non-nil error: ", err)
}

func TestServerRun(t *testing.T) {
	cfg := &config.Config{
		HTTPListen:   "127.0.0.1:0",
		HTTPSListen:  "127.0.0.1:0",
		CertFilePath: certFileName,
		KeyFilePath:  keyFileName,
	}
	generateCertFiles()
	defer deleteCertFiles()

	srv, err := NewServer(cfg)
	assert.Nil(t, err, "see non-nil error: ", err)

	stopCh := make(chan struct{})
	go func() {
		time.Sleep(2 * time.Second)
		close(stopCh)
	}()

	err = srv.Run(stopCh)
	assert.Nil(t, err, "see non-nil error: ", err)
}

func TestProfileNotMount(t *testing.T) {
	cfg := &config.Config{HTTPListen: "127.0.0.1:0"}
	srv, err := NewServer(cfg)
	assert.Nil(t, err, "see non-nil error: ", err)
	stopCh := make(chan struct{})
	go func() {
		err := srv.Run(stopCh)
		assert.Nil(t, err, "see non-nil error: ", err)
	}()

	u := (&url.URL{
		Scheme: "http",
		Host:   srv.httpListener.Addr().String(),
		Path:   "/debug/pprof/cmdline",
	}).String()

	resp, err := http.Get(u)
	assert.Nil(t, err, nil)
	assert.Equal(t, resp.StatusCode, http.StatusNotFound)
	close(stopCh)
}

func TestProfile(t *testing.T) {
	cfg := &config.Config{HTTPListen: "127.0.0.1:0", EnableProfiling: true}
	srv, err := NewServer(cfg)
	assert.Nil(t, err, "see non-nil error: ", err)
	stopCh := make(chan struct{})
	go func() {
		err := srv.Run(stopCh)
		assert.Nil(t, err, "see non-nil error: ", err)
	}()

	u := (&url.URL{
		Scheme: "http",
		Host:   srv.httpListener.Addr().String(),
		Path:   "/debug/pprof/cmdline",
	}).String()

	resp, err := http.Get(u)
	assert.Nil(t, err, nil)
	assert.Equal(t, resp.StatusCode, http.StatusOK)
	close(stopCh)
}
