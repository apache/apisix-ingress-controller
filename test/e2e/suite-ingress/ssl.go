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

package ingress

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("suite-ingress: SSL Testing", func() {
	suites := func(scaffoldFunc func() *scaffold.Scaffold) {
		s := scaffoldFunc()
		ginkgo.It("create a SSL from ApisixTls ", func() {
			secretName := "test-apisix-tls"
			cert := `-----BEGIN CERTIFICATE-----
MIIDSjCCAjICCQC/34ZwGz7ZXjANBgkqhkiG9w0BAQsFADBmMQswCQYDVQQGEwJD
TjEQMA4GA1UECAwHSmlhbmdzdTEPMA0GA1UEBwwGU3V6aG91MQ8wDQYDVQQKDAZ6
aGlsaXUxEDAOBgNVBAsMB3NlY3Rpb24xETAPBgNVBAMMCHRlc3QuY29tMCAXDTIx
MDIwMzE0MjkwOVoYDzIwNTEwMTI3MTQyOTA5WjBmMQswCQYDVQQGEwJDTjEQMA4G
A1UECAwHSmlhbmdzdTEPMA0GA1UEBwwGU3V6aG91MQ8wDQYDVQQKDAZ6aGlsaXUx
EDAOBgNVBAsMB3NlY3Rpb24xETAPBgNVBAMMCHRlc3QuY29tMIIBIjANBgkqhkiG
9w0BAQEFAAOCAQ8AMIIBCgKCAQEA3DEQ5K9PVYicINTHt3arqrsrftrhotyBuGqM
xxqGMVO/E2SAa/81fC1UCcjYV4Wila0kl8i5fa8HjtVm5UWlrqxeFLOS3E0Wv2QY
w46BGZJY4InE9zKwYyC2DkBxE6p14JRjmtW/MQPNaOFjJ4bmCuRHsEzmQIGRM0b7
oKHjfFwv6l7BahgGf9ShHOMdHSkgWj6+2RU3282lrO9bY1JBTKu2Znv9M79nu1Px
Tn1wCfcuCwA7WQT/QSrE2R43I2vmbIbuSmeg9ivjMazRYQQ+qxQn/6zhiHvP3QZG
dKmp8imdYi+r84PKOLDEe/yxlgIdr2Au5WCPWwyYMYPWHzeD1wIDAQABMA0GCSqG
SIb3DQEBCwUAA4IBAQBYzNe83mPVuz96TZ3fmxtOIuz9b6q5JWiJiOzjAD9902Se
TNYzMM6T/5e0dBpj8Z2qQlhkfNxJJgTwGEE8SdrZIr8DhswR9a0bXDCZjLatCdeU
iYpt+TDAuySnLhAcd3GfE5ml6am2dOsOKpxHU/8clUSaz+21fckRopWo+xL6rSVC
4vvKqiU+LWLTZPQNoOqowl7bxoQO2jMWfN/5zvQOFxAbEufIPa9ti3qonDCXbkYn
PpET/mPDrcb4bGsZkW/cu0LrPSUVp12br5TAYaXqYS0Ex+jAVTXML9SeEQuvU3dH
5Uw2wVHxQXHglsdCYUXXFd3HZffb4rSQH+Mk0CBI
-----END CERTIFICATE-----`
			key := `-----BEGIN RSA PRIVATE KEY-----
MIIEpQIBAAKCAQEA3DEQ5K9PVYicINTHt3arqrsrftrhotyBuGqMxxqGMVO/E2SA
a/81fC1UCcjYV4Wila0kl8i5fa8HjtVm5UWlrqxeFLOS3E0Wv2QYw46BGZJY4InE
9zKwYyC2DkBxE6p14JRjmtW/MQPNaOFjJ4bmCuRHsEzmQIGRM0b7oKHjfFwv6l7B
ahgGf9ShHOMdHSkgWj6+2RU3282lrO9bY1JBTKu2Znv9M79nu1PxTn1wCfcuCwA7
WQT/QSrE2R43I2vmbIbuSmeg9ivjMazRYQQ+qxQn/6zhiHvP3QZGdKmp8imdYi+r
84PKOLDEe/yxlgIdr2Au5WCPWwyYMYPWHzeD1wIDAQABAoIBAQDGmSKpgE1H0k0v
d3siyFART3vtkLHOWKBPmxqaQhwixWwjq5QA1FCDTcbshFBMsGVyJpZIqGxVJdbl
RyjlRaooH6NDfKvLM2R+/2Mujot2av7qlpgmdXuODOTnecwDds2W33/vGTa2mL1e
CVuLPSqjTD40j0dlivdRjoZJ3Xn2oOrpZ812XU8KeZAjuSEAwcyl2nSbyLGDchBB
kfYZold3FaaLAf2LoVJ2fs+FwEPzDKoNYEvij9OyC0kwI94T5jQ+Z6XGtHXhb2Hy
Ek3EZeIhV3YcDIid5AjSvcrNtDI24hwszSmhYVc53EKYkpXHf581a3U/SEEhXDlw
Y0x6j9QRAoGBAPEP0LDgP7DGXxno4h+nf0AMru0pxlrNVQhLcNQB+dYI0pFTwsg+
AKenoaaE/EGR1KfiY0uf3rVWNrA5kyX1/i18iJx9LSf9NvNgMo84JVvXINgyE6sd
hvdqxFlV5FBnh8b7ldvYQy3YI0EQNx+/rmeUYPjInbkdiksAtAey4ADNAoGBAOnW
K0FoX1ljq3rc9uVasiRi6Ix50NHdZ17RcEpMgwWPenbP1aiWkvA8yFhU708lBaZC
WIUZ6XbfiG0Y9cMtxhtikoouDs5Ifia8juZ2bhkmSGP2FvZCBJJ/sHhnhpzSZNhW
SyLBUjnynoXwHoQvkoGnVTHAk1VsY7jLNJdr2MczAoGAMYvMmu4caRr8pPimsVbd
4q44reouKK+XUJMg55JYZVN+4/vRRxLnU44yvWUL6/YrPS5ctkhvn9nOd739rom2
6mZ0NaXMyDFVQAR/n8wscYnv6D+ypzL0cJnzLWFoAdalo5JGJN94P03zQQYyLkZZ
dFSc8cVaFZgqumu0lPiA7ekCgYEAiMeVL8Jcm84YXVrpNMmzkGMmwhzzT/8hWy5J
b7yHm3YM3Xi+8sl5E/uJ+VldTj9KqbD/VIQOs1EX3TEPeObKjfQ/4YIFeRagbAo5
0IcP6bgh+g7V6aA+Sm9Ui2mLLSpIgN8hPig0796CabhGMW4eVabKx7pstDgdsNd0
YOpduE8CgYEAu9k9WOQuRX4f6i5LBIxyaYn6Hw6oJn8e/w+p2+HNBXdyVQNqUHBG
V5rgnBwhc5LeIFbehKvQOvYSWbwbA1VunMpdYgV6+EBLayumJNqV6jGei4okx2of
wrw7im4TNSAdwVX4Y1F4svJ2as5SJn5QYGAzXDixNuwzXYrpP9rzA2s=
-----END RSA PRIVATE KEY-----`
			// create secret
			err := s.NewSecret(secretName, cert, key)
			assert.Nil(ginkgo.GinkgoT(), err, "create secret error")
			// create ApisixTls resource
			tlsName := "tls-name"
			host := "api6.com"
			err = s.NewApisixTls(tlsName, host, secretName)
			assert.Nil(ginkgo.GinkgoT(), err, "create tls error")
			// check ssl in APISIX
			time.Sleep(10 * time.Second)
			tls, err := s.ListApisixSsl()
			assert.Nil(ginkgo.GinkgoT(), err, "list tls error")
			assert.Len(ginkgo.GinkgoT(), tls, 1, "tls number not expect")
		})
		ginkgo.It("update a SSL from ApisixTls ", func() {
			secretName := "test-apisix-tls"
			cert := `-----BEGIN CERTIFICATE-----
MIIDSDCCAjACCQDf02nwtW2VrzANBgkqhkiG9w0BAQsFADBmMQswCQYDVQQGEwJj
bjEQMA4GA1UECAwHamlhbmdzdTEPMA0GA1UEBwwGc3V6aG91MQ8wDQYDVQQKDAZ6
aGlsaXUxEDAOBgNVBAsMB3NlY3Rpb24xETAPBgNVBAMMCGFwaTYuY29tMB4XDTIx
MDEyNTA2MDQ0MVoXDTIxMDIyNDA2MDQ0MVowZjELMAkGA1UEBhMCY24xEDAOBgNV
BAgMB2ppYW5nc3UxDzANBgNVBAcMBnN1emhvdTEPMA0GA1UECgwGemhpbGl1MRAw
DgYDVQQLDAdzZWN0aW9uMREwDwYDVQQDDAhhcGk2LmNvbTCCASIwDQYJKoZIhvcN
AQEBBQADggEPADCCAQoCggEBAMQFzmobVVuixOa0cEItZLzt3gKifUS1b+sN5d0y
7SGKeApjhgSl0bl1xFXEGyFttCNlFU0+adbKHXJLsNFbO/l8xi+218ihYZ1mM708
8T8IJM4d4jpx0OKFZSU9two+VxNLTwFsat2GiB39KMiNpLOShhIdK9BKT8+v6Uqq
MbkYoCCRObcBsCuA7hhyteSdN7ccuxuMS28862R4gvhXGF2+BBXLnegzHE3PKexF
0vekJcfVH/LKS0iwl+Gcn6isJXQQTx6+llko+Flh7fqbrDIKV4EJm/5GfULJkjlp
SviTHJ5rJgZUjdkozA2O8ELpb3vsjEs44M+3h6v+AQ8LSrkCAwEAATANBgkqhkiG
9w0BAQsFAAOCAQEABt98FafJfmZ2Gaf/Fip9bf4qxGUlRfJpZ8K775VRSXAcI/by
Bh4wjd3DwUMVFFarx8CxcGHgjpK6bWE3tkQjc7R24xhPVaF/zyiPakrTHkWENHPZ
HbkOmZOY8wfZ8pPGUwHGA6bCmytWSD0lseEhxaHcZ27MmKI5CdUsgJXbc1q9gr3F
x4cosJI+W55Kzejiqgm/wzBbr4OpjW4DDz1YBJFXCc1TN9pf2ALkWZ8j3HfMrn2y
HvOefA8g628WpNtPZodWe/zC8hanCzRMp37JPbh85+RwlGhi7gIkhvjf78EiAZBy
eHg1iDgdVUzlXn+LNPCAbjxCaTqn6zmIb+GkhA==
-----END CERTIFICATE-----`
			key := `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAxAXOahtVW6LE5rRwQi1kvO3eAqJ9RLVv6w3l3TLtIYp4CmOG
BKXRuXXEVcQbIW20I2UVTT5p1sodckuw0Vs7+XzGL7bXyKFhnWYzvTzxPwgkzh3i
OnHQ4oVlJT23Cj5XE0tPAWxq3YaIHf0oyI2ks5KGEh0r0EpPz6/pSqoxuRigIJE5
twGwK4DuGHK15J03txy7G4xLbzzrZHiC+FcYXb4EFcud6DMcTc8p7EXS96Qlx9Uf
8spLSLCX4ZyfqKwldBBPHr6WWSj4WWHt+pusMgpXgQmb/kZ9QsmSOWlK+JMcnmsm
BlSN2SjMDY7wQulve+yMSzjgz7eHq/4BDwtKuQIDAQABAoIBAQCfVbTmDNfCR7lT
agIB2NIUvCkE7T1z1yNk5gQdXCLia6KNlz49kue5n596s4/2CS0uxCCfjAcN/3YW
DK5qToWekyypZi9aNsuY3JVb0iiqupzoKeRU62UGa7W+or6hBfFTjphmqNDoxkzo
S3qWIoRpLsXp/Wo6wdzEDdJMzbDjKVHUdcoeJ2IQdMG4dKKdf9NMZOhohZs+K0Kf
oroLTbrjCf5wI16KPxHVKe/6vw3098GKJc+MTfHtANJbwmI4dAlLcfbZ1I6VUoL6
JkCphK8BJ2jxeu0xTu7TXkHcMd/yK4pKmEQwjSpDOl0qWgFYAXJR2RHCaduR6w4l
XJcbnARtAoGBAPmwYjGHeCpzQdHA6Atkc9ETSdzfRShG7H/cRdluS6J4KEAJAFW7
i+Xc3rQf67CR/3JJgXObL1ZvQeIZ0Q0UD0WbBopJc2hfGRKN9lsFclMqDTzBHvvi
ZukE/IvL3elhtuskLyc9Wf0JGoEsdkQkMQT+wMyxbrZ6im2MWm/xswrnAoGBAMj6
LIysCK2LbOcPoi33nOGBC2ITUwhJGbbCeBho0xqpzcD20aQszJmYJkDng2WVkjdf
3MO2HDULA2JvEMdCrjvG5U1smLdbBQ89aIhy6clDKb5PMlOo9fo3E9ICyL5StFyy
09H0UGoCocZlBPOZQ70k5kLYOKf7QB9TeTyaIulfAoGAHDww7m7mTM6Zy9FnrBog
6qymtp5c4LAcgFz1XSAW13mE+7DI4+kAae7vFClj6qSn4VGknOEYmkqchafrtvHk
xDdCpxKlRVEzsaByElrsUbE4q/0ettckUgdpU5mrL4AIQlDmMCbE7VNBNwhDG3OI
Q4tXXA5YebQjwT2U4IHRgFMCgYEAxc82Od65S9aHAYUpowSrrGhOw+ExQF5yqKcP
fTbvULcAhIRqIqTVW/ec7xTvBvUITOhVaWu8p5iHZELcyMKgqsVAu8u/I/i6Kh3O
3T39TNKGK4HXjvAl6nh7UaDb5DeSvgpk4akN3MlqYNLc5MZdHbVLzU7ztKJeonaO
RU+QPRECgYB6XW24EI5+w3STbpnc6VoTS+sy9I9abTJPYo9LpCJwfMYc9Tg9Cx2K
29PnmSrLFpU2fvE0ijpyHRr7gGmINTxbrmTmfMBI01m+GpPuvDcBQ2tsFJ+A3DzN
9xJulR2NZUZdDIIIqx983ANE6S4Zb8rAbsoHQdqpjUrcVxI2OJBp3Q==
-----END RSA PRIVATE KEY-----`
			// create secret
			err := s.NewSecret(secretName, cert, key)
			assert.Nil(ginkgo.GinkgoT(), err, "create secret error")
			// create ApisixTls resource
			tlsName := "tls-name"
			host := "api6.com"
			err = s.NewApisixTls(tlsName, host, secretName)
			assert.Nil(ginkgo.GinkgoT(), err, "create tls error")
			// update ApisixTls resource
			host = "api7.com"
			err = s.NewApisixTls(tlsName, host, secretName)
			assert.Nil(ginkgo.GinkgoT(), err, "update tls error")

			// check ssl in APISIX
			time.Sleep(10 * time.Second)
			tls, err := s.ListApisixSsl()
			assert.Nil(ginkgo.GinkgoT(), err, "list tls error")
			assert.Len(ginkgo.GinkgoT(), tls, 1, "tls number not expect")
			assert.Equal(ginkgo.GinkgoT(), tls[0].Snis[0], host, "tls host is error")
			assert.Equal(ginkgo.GinkgoT(), tls[0].Labels, map[string]string{
				"managed-by": "apisix-ingress-controller",
			})
		})
		ginkgo.It("delete a SSL from ApisixTls ", func() {
			secretName := "test-apisix-tls"
			cert := `-----BEGIN CERTIFICATE-----
MIIDSDCCAjACCQDf02nwtW2VrzANBgkqhkiG9w0BAQsFADBmMQswCQYDVQQGEwJj
bjEQMA4GA1UECAwHamlhbmdzdTEPMA0GA1UEBwwGc3V6aG91MQ8wDQYDVQQKDAZ6
aGlsaXUxEDAOBgNVBAsMB3NlY3Rpb24xETAPBgNVBAMMCGFwaTYuY29tMB4XDTIx
MDEyNTA2MDQ0MVoXDTIxMDIyNDA2MDQ0MVowZjELMAkGA1UEBhMCY24xEDAOBgNV
BAgMB2ppYW5nc3UxDzANBgNVBAcMBnN1emhvdTEPMA0GA1UECgwGemhpbGl1MRAw
DgYDVQQLDAdzZWN0aW9uMREwDwYDVQQDDAhhcGk2LmNvbTCCASIwDQYJKoZIhvcN
AQEBBQADggEPADCCAQoCggEBAMQFzmobVVuixOa0cEItZLzt3gKifUS1b+sN5d0y
7SGKeApjhgSl0bl1xFXEGyFttCNlFU0+adbKHXJLsNFbO/l8xi+218ihYZ1mM708
8T8IJM4d4jpx0OKFZSU9two+VxNLTwFsat2GiB39KMiNpLOShhIdK9BKT8+v6Uqq
MbkYoCCRObcBsCuA7hhyteSdN7ccuxuMS28862R4gvhXGF2+BBXLnegzHE3PKexF
0vekJcfVH/LKS0iwl+Gcn6isJXQQTx6+llko+Flh7fqbrDIKV4EJm/5GfULJkjlp
SviTHJ5rJgZUjdkozA2O8ELpb3vsjEs44M+3h6v+AQ8LSrkCAwEAATANBgkqhkiG
9w0BAQsFAAOCAQEABt98FafJfmZ2Gaf/Fip9bf4qxGUlRfJpZ8K775VRSXAcI/by
Bh4wjd3DwUMVFFarx8CxcGHgjpK6bWE3tkQjc7R24xhPVaF/zyiPakrTHkWENHPZ
HbkOmZOY8wfZ8pPGUwHGA6bCmytWSD0lseEhxaHcZ27MmKI5CdUsgJXbc1q9gr3F
x4cosJI+W55Kzejiqgm/wzBbr4OpjW4DDz1YBJFXCc1TN9pf2ALkWZ8j3HfMrn2y
HvOefA8g628WpNtPZodWe/zC8hanCzRMp37JPbh85+RwlGhi7gIkhvjf78EiAZBy
eHg1iDgdVUzlXn+LNPCAbjxCaTqn6zmIb+GkhA==
-----END CERTIFICATE-----`
			key := `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAxAXOahtVW6LE5rRwQi1kvO3eAqJ9RLVv6w3l3TLtIYp4CmOG
BKXRuXXEVcQbIW20I2UVTT5p1sodckuw0Vs7+XzGL7bXyKFhnWYzvTzxPwgkzh3i
OnHQ4oVlJT23Cj5XE0tPAWxq3YaIHf0oyI2ks5KGEh0r0EpPz6/pSqoxuRigIJE5
twGwK4DuGHK15J03txy7G4xLbzzrZHiC+FcYXb4EFcud6DMcTc8p7EXS96Qlx9Uf
8spLSLCX4ZyfqKwldBBPHr6WWSj4WWHt+pusMgpXgQmb/kZ9QsmSOWlK+JMcnmsm
BlSN2SjMDY7wQulve+yMSzjgz7eHq/4BDwtKuQIDAQABAoIBAQCfVbTmDNfCR7lT
agIB2NIUvCkE7T1z1yNk5gQdXCLia6KNlz49kue5n596s4/2CS0uxCCfjAcN/3YW
DK5qToWekyypZi9aNsuY3JVb0iiqupzoKeRU62UGa7W+or6hBfFTjphmqNDoxkzo
S3qWIoRpLsXp/Wo6wdzEDdJMzbDjKVHUdcoeJ2IQdMG4dKKdf9NMZOhohZs+K0Kf
oroLTbrjCf5wI16KPxHVKe/6vw3098GKJc+MTfHtANJbwmI4dAlLcfbZ1I6VUoL6
JkCphK8BJ2jxeu0xTu7TXkHcMd/yK4pKmEQwjSpDOl0qWgFYAXJR2RHCaduR6w4l
XJcbnARtAoGBAPmwYjGHeCpzQdHA6Atkc9ETSdzfRShG7H/cRdluS6J4KEAJAFW7
i+Xc3rQf67CR/3JJgXObL1ZvQeIZ0Q0UD0WbBopJc2hfGRKN9lsFclMqDTzBHvvi
ZukE/IvL3elhtuskLyc9Wf0JGoEsdkQkMQT+wMyxbrZ6im2MWm/xswrnAoGBAMj6
LIysCK2LbOcPoi33nOGBC2ITUwhJGbbCeBho0xqpzcD20aQszJmYJkDng2WVkjdf
3MO2HDULA2JvEMdCrjvG5U1smLdbBQ89aIhy6clDKb5PMlOo9fo3E9ICyL5StFyy
09H0UGoCocZlBPOZQ70k5kLYOKf7QB9TeTyaIulfAoGAHDww7m7mTM6Zy9FnrBog
6qymtp5c4LAcgFz1XSAW13mE+7DI4+kAae7vFClj6qSn4VGknOEYmkqchafrtvHk
xDdCpxKlRVEzsaByElrsUbE4q/0ettckUgdpU5mrL4AIQlDmMCbE7VNBNwhDG3OI
Q4tXXA5YebQjwT2U4IHRgFMCgYEAxc82Od65S9aHAYUpowSrrGhOw+ExQF5yqKcP
fTbvULcAhIRqIqTVW/ec7xTvBvUITOhVaWu8p5iHZELcyMKgqsVAu8u/I/i6Kh3O
3T39TNKGK4HXjvAl6nh7UaDb5DeSvgpk4akN3MlqYNLc5MZdHbVLzU7ztKJeonaO
RU+QPRECgYB6XW24EI5+w3STbpnc6VoTS+sy9I9abTJPYo9LpCJwfMYc9Tg9Cx2K
29PnmSrLFpU2fvE0ijpyHRr7gGmINTxbrmTmfMBI01m+GpPuvDcBQ2tsFJ+A3DzN
9xJulR2NZUZdDIIIqx983ANE6S4Zb8rAbsoHQdqpjUrcVxI2OJBp3Q==
-----END RSA PRIVATE KEY-----`
			// create secret
			err := s.NewSecret(secretName, cert, key)
			assert.Nil(ginkgo.GinkgoT(), err, "create secret error")
			// create ApisixTls resource
			tlsName := "tls-name"
			host := "api6.com"
			err = s.NewApisixTls(tlsName, host, secretName)
			assert.Nil(ginkgo.GinkgoT(), err, "create tls error")

			// check ssl in APISIX
			time.Sleep(10 * time.Second)
			tls, err := s.ListApisixSsl()
			assert.Nil(ginkgo.GinkgoT(), err, "list tls error")
			assert.Len(ginkgo.GinkgoT(), tls, 1, "tls number not expect")
			assert.Equal(ginkgo.GinkgoT(), tls[0].Snis[0], host, "tls host is error")

			// delete ApisixTls
			err = s.DeleteApisixTls(tlsName, host, secretName)
			assert.Nil(ginkgo.GinkgoT(), err, "delete tls error")
			// check ssl in APISIX
			time.Sleep(10 * time.Second)
			tls, err = s.ListApisixSsl()
			assert.Nil(ginkgo.GinkgoT(), err, "list tls error")
			assert.Len(ginkgo.GinkgoT(), tls, 0, "tls number not expect")
		})
	}

	ginkgo.Describe("suite-ingress: scaffold v2beta3", func() {
		suites(scaffold.NewDefaultScaffold)
	})
	ginkgo.Describe("suite-ingress: scaffold v2", func() {
		suites(scaffold.NewDefaultV2Scaffold)
	})
})

var _ = ginkgo.Describe("suite-ingress: ApisixTls mTLS Test", func() {
	// RootCA -> Server
	// RootCA -> UserCert
	// These certs come from mTLS practice

	rootCA := `-----BEGIN CERTIFICATE-----
MIIGEjCCA/qgAwIBAgIUJgGBjfayA0ZB+iWklvizaah7eq0wDQYJKoZIhvcNAQEL
BQAwgYgxCzAJBgNVBAYTAkNOMRAwDgYDVQQIDAdCZWlqaW5nMRAwDgYDVQQHDAdC
ZWlqaW5nMRYwFAYDVQQKDA1BcGFjaGUtQVBJU0lYMRcwFQYDVQQDDA5BUElTSVgt
SW5ncmVzczEkMCIGCSqGSIb3DQEJARYVZGV2QGFwaXNpeC5hcGFjaGUub3JnMB4X
DTIyMDUzMDIzMTE0MVoXDTMyMDUyNzIzMTE0MVowgYgxCzAJBgNVBAYTAkNOMRAw
DgYDVQQIDAdCZWlqaW5nMRAwDgYDVQQHDAdCZWlqaW5nMRYwFAYDVQQKDA1BcGFj
aGUtQVBJU0lYMRcwFQYDVQQDDA5BUElTSVgtSW5ncmVzczEkMCIGCSqGSIb3DQEJ
ARYVZGV2QGFwaXNpeC5hcGFjaGUub3JnMIICIjANBgkqhkiG9w0BAQEFAAOCAg8A
MIICCgKCAgEArikPh5fAuFOOYZGoYp3DR8wrWE5hM02/OeOykj0gPXPUsU8/VWd6
stYjGu62hA1MY1hxWbzN7BDnIo/irOhJu1d1pSeRiGlEZAOqV7EKg0/t4ZrQr/TB
iUcQwiIIEVZwNKn1YWfTneq67vSMQ+jyMebyMkFIDGmApyayoZPEGtLSW8azqge+
wMvAayM3uCoeo1WUnX3oTV/VyNuPwcOzSwbHvJrAIkjnob68jkvZUNZPL5l50hGz
x+L7VnMVmzz9OVUCMEA/uZ7eM/bh1jPes56ySvyLMF0RDhKUdURRo6xWyGJRnZcs
B4AHJEHHahFJFoctKkCNP2gC9oLW8/sH7HwZWwfmS6Dn/YVe7dJznN0n78orPf0r
fgJCly2KGxeJFfPROsFMGfZ79NLL8Q3CylCc/hwpMLIVFBKA9NePmpKY/0TtYp8C
TSu+KENzAEvOPXw0o9Kmr+/oGaLzZW61OFh1GkpEQt2deDT3LBBiGbk7YMm8eLcg
DrsDhH1lumag00Xw34fL67jEt541OeekHBcw8kEgWHWmS5xepRTeVBu9xh8HeoVp
kO2bF56teuh4jWkmclP/1gKD/kGt+2qPDNcyKXCjd45DxWa7/ryHiAqYuzgRHYs2
j+4GJlH/UNwc8kSHviyiVKVDU7lzgww4gUQEfaARk8/MjmX1fnZcqzMCAwEAAaNy
MHAwHQYDVR0OBBYEFCD4iZRLmy3bPxKyk8+46G6xHZ1aMB8GA1UdIwQYMBaAFCD4
iZRLmy3bPxKyk8+46G6xHZ1aMA8GA1UdEwEB/wQFMAMBAf8wHQYDVR0RBBYwFIIS
bXRscy5odHRwYmluLmxvY2FsMA0GCSqGSIb3DQEBCwUAA4ICAQCIgBcfghjJe1ZF
k+JdCn0aDUaNokChTuOlJ4CvT8hr/08xI8DGK8uVCzPDe1OPvc/0B9cYlguGlJIo
tOUtOdhulDRGmw4sGoOLRb9leZXtPdWoCNFZBhf0i5V+rQffFlZbdqu7fJcXCwFi
2xUA8O2Lqu/6T3GO27GRDAmL+Qfn1G4PHI/+tRy5P1DNC5qTaZM7pe5D9n2XJVaE
Z+PDsI3ZnY6s2raFM+6Yjq3l/S5F8HWkIv+C1RNp52Nq3UpL1qeG/XuOLot1lI0t
wWTFYdCcyFGJRyW0V/Bk/PNlZ24wvXv2O+4xRVc139+VumDCb/qopVyHvkjLRUL3
ILHa0AUJF8LF4Sh5NloPeE+p8y2f/fY5tCdr4Tz8/mLx8APrf2wbXH07lwI7F3TC
bGkmSA9GqyVdZiM1XDyBKYpfJInzjOLPuzsd8xYAOJsg972jSD7oTHMKRiWkTEKC
WTUIkcN8dZO3CxthRAWvIuXhkpGoN3OYRBODkUW1aJ6+cP25icMYXSK6uBudOYpY
/yIdh6g3VCz7mN8KdEqeYKrw2cR0j01r1pqnh4YqLnC2TYar91MTK/6uQNPvaKIz
yL0TPM7LtpidKkHe49y9XWgRUGfmq+1/gFCY/Vk0xrXNcCJYpzoTMyZ9zEZFP2iu
djgSHn8f4ZS1qTtukbKsWFkq82jesw==
-----END CERTIFICATE-----
`

	serverCertSecret := `server-secret`
	serverCert := `-----BEGIN CERTIFICATE-----
MIIF0DCCA7igAwIBAgIUf2P2yoh6+RjG0ZJ72PNn7HbfKkcwDQYJKoZIhvcNAQEL
BQAwgYgxCzAJBgNVBAYTAkNOMRAwDgYDVQQIDAdCZWlqaW5nMRAwDgYDVQQHDAdC
ZWlqaW5nMRYwFAYDVQQKDA1BcGFjaGUtQVBJU0lYMRcwFQYDVQQDDA5BUElTSVgt
SW5ncmVzczEkMCIGCSqGSIb3DQEJARYVZGV2QGFwaXNpeC5hcGFjaGUub3JnMB4X
DTIyMDUzMDIzMTMyMloXDTMyMDQwNzIzMTMyMlowgYwxCzAJBgNVBAYTAkNOMRAw
DgYDVQQIDAdCZWlqaW5nMRAwDgYDVQQHDAdCZWlqaW5nMRYwFAYDVQQKDA1BcGFj
aGUtQVBJU0lYMRswGQYDVQQDDBJtdGxzLmh0dHBiaW4ubG9jYWwxJDAiBgkqhkiG
9w0BCQEWFWRldkBhcGlzaXguYXBhY2hlLm9yZzCCAiIwDQYJKoZIhvcNAQEBBQAD
ggIPADCCAgoCggIBAKzseBvWEooy23QdRIIkKlKsMv720PiUeEIGsIGOjtNhcTSp
7e2zMw7kfrsvkmFUGdgajNCtQ3x/2jLncnS1kxKgS1ssOe0Fc5E7JuqiMtGPEhi3
6YjzWUYqPC9mSr27EdtYiNATTytZJs1Xa7VoElZjtOpxy5MEFz3HPCjafGCdpRFY
HEZ8vB890mPSEwOKspApYBPJfoUnby1n4PtMCt2xGeiaZ3NjNCqoS7OkCX9MjE4U
KYZc2nhZi7BfpasTB9s5UbEmaZoSpSCAjtT5dWPEPmo3vpzAxd7ol8nbXJC+p/oq
nVwhgiYc6wDHlSUDpua4lFoB9TBlN//saR+9/nfiT//YlgzlLrO92GmGIRzZNXzd
g5G1cbPJWK72wKAY9fVTcVhvc7XEfLj1Qd9LJdhqEzQe/VfLyITqGKA5pVNBA5aD
9Yb95fckgUq3me5Udb5dtE8Ua7oFF8ma4Vu9ao7bPujPdGFSb1NDc3T256HxhzQy
cNyH2/WvsceGlv29ADc87j8v5t5BURSUgWylgshyehFdQ63co66D6RW7nDokaMHL
UkRbm50EHZCjdSnRagra3UWKfjRK2JN631G6pXGyfncykzyV48qtmlzrboC9bb7T
8A30d18Z3ZiB3QO/eMCC4FK+6XqJvYzPMxYvkWMAe6AIz0Jbw918xEeUVfdJAgMB
AAGjLDAqMAkGA1UdEwQCMAAwHQYDVR0RBBYwFIISbXRscy5odHRwYmluLmxvY2Fs
MA0GCSqGSIb3DQEBCwUAA4ICAQCG9X4hl3Bnd5Pc0vYeUFo/FLkg8z6BnM2/fUmL
ty5uUAnU9NQOOQEKjQF4tU4o8+oexWlrjKXw//MOwpYAOJ3BrYNcqPOvX3mqhtVB
ULBzLhQeFxmbwQ7ZNoB2nh+7tH+Z/fqddVE3QPG4TxyEOHEa4Zr6WZ0Ienh9c9nW
GZGS3ggSNwdcxjapWPvwRNdFKylJTr57ebocoWc4NbRJxOny4E3Cl7yzT6MbYrsZ
R2QyjVKG7pi/gbq2fpFip+BG2PyV9HRba47KoxF/29LQkdr4R16afdcrPdXT5qCA
aDgEBHV7lHKz3yqGBrksB0LCG0MaT3mu80bhj4b5VhZ3ORrQ6gm1eyevJGZRaNgA
s7POltX9sxLcEQO6F7FWGXs2tq+gTl1KT23QuFj7UgVQ0SMrefR6GibAwHO0TFmN
HLlpYma1/zuU4QxI/+S+l4b6qx31TnUayRgOQn/J6vs+qZiD6fdugijQif4DsI3K
QBDwI9IxqyvgCleMbvlRfFFIENUpf+uWjkqOO/2yax3egvOfoCojt193vlXHxOjK
UROxuVcLmJgIUFR+KhqGibrk55oB5az7KVmV1OcUBhll6HI/SRDN/XRoJpDObczB
B3vZ3vLBZTgVBX+8TATMAaRhIoa1dupvOmBc0DQRibpWhJvpl4WPsQRMVFM3p1Kw
4e5kxA==
-----END CERTIFICATE-----
`
	serverKey := `-----BEGIN PRIVATE KEY-----
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
3h0xluQw6uolHPequOpCJAbnSSfasUE=
-----END PRIVATE KEY-----
`
	clientCASecret := `client-ca-secret`
	clientCert := `-----BEGIN CERTIFICATE-----
MIIF0DCCA7igAwIBAgIUf2P2yoh6+RjG0ZJ72PNn7HbfKkgwDQYJKoZIhvcNAQEL
BQAwgYgxCzAJBgNVBAYTAkNOMRAwDgYDVQQIDAdCZWlqaW5nMRAwDgYDVQQHDAdC
ZWlqaW5nMRYwFAYDVQQKDA1BcGFjaGUtQVBJU0lYMRcwFQYDVQQDDA5BUElTSVgt
SW5ncmVzczEkMCIGCSqGSIb3DQEJARYVZGV2QGFwaXNpeC5hcGFjaGUub3JnMB4X
DTIyMDUzMDIzMTY0NloXDTMyMDQwNzIzMTY0NlowgYwxCzAJBgNVBAYTAkNOMRAw
DgYDVQQIDAdCZWlqaW5nMRAwDgYDVQQHDAdCZWlqaW5nMRYwFAYDVQQKDA1BcGFj
aGUtQVBJU0lYMRswGQYDVQQDDBJtdGxzLmh0dHBiaW4ubG9jYWwxJDAiBgkqhkiG
9w0BCQEWFWRldkBhcGlzaXguYXBhY2hlLm9yZzCCAiIwDQYJKoZIhvcNAQEBBQAD
ggIPADCCAgoCggIBAJzowwrITI3QTFjyADv6zoBAQFZ9Z5qcmvVFsEjdp97THFop
MdImJ4qobZgKKwaYjQAopOTSqnU+WYlVFRyNTv4xOBbSWPx7xyed6IFWjg6U3DEE
0N+pswg6sa9LtLNVl3Ddw+54YL1IZTLnCmP380nI0neh90sqFqMBN8SnmwGKfO4W
T6/sosRr5ZQ+31FJW/BrguWbpLE79I71Un0+hjkCJbwPQaUrSBhVcy928VAvHLOL
twdm6QPcFAtQORKRsTyBT6aXyImP9VHxQs+a0kM8gHvtwC2PkD5Ekag3hP5hHtV/
Xc3mganw0UJOaWzEkhVhw5i2byDxfXPx6fd8lO6gowU2jV1LeT9j3kFQf1Jor2mT
S2xTeCur771jFSMyCKjFrE5ePeTzyrUrJ4t9LB1ynocEXbWH63EobMYU5uW/UPaK
svNZihHiX+xntJPDXK5w3pm9m4q4kJm3PXqXjXpN5G9LHliP9T+c/oCnf46iT4om
yq2sWegsr+V7GfT4w44/1S/eMklf3fz2BbOCnk53aQDfqAmCBFLttddn0l1yWo8P
e4saEpbu4W8uqaIjs7rZstKaxbTb4rbsHHORSc8Rvp12lmq0tdH1GnCJrkpluxYP
2hn/6KgLQuPT1WkQeAkxrinlbuHk4MpqU8dglwhUnKVUbLIjYBpklGp+axxPAgMB
AAGjLDAqMAkGA1UdEwQCMAAwHQYDVR0RBBYwFIISbXRscy5odHRwYmluLmxvY2Fs
MA0GCSqGSIb3DQEBCwUAA4ICAQBrAYU/4+TDDD5qyeMoS/a86SRcd3+idphz8G2E
8AAb7m+AGx6CDrmlloF8x7O3AHfmpgv8WZUx8gtVfC0hyWMUuoi98EA/5IdEVFTZ
/4vpKtLrXtuUBlV1hiWOFpLlt6tOZW7jLJRemwZXvGvETvp6hJZwek5v/AU5Eq9E
DCAjvs8uEOIWTflXanBe92tCGfhq6XzpRd3kQi/NZy7V3yCoqqXRLCz7rd9OtN32
KBYAprIFJ75BlwoZ5e8AArVQ5InSfJEx/oLbymSitIjSJn+R7psofwEs6mXNGN5H
ednGWq7l5bhp5mbYBEbwHmb9lOSfRtT7s4/SCgkJWTieU5klowF+RvSkjbbhOcaK
x/DQJTrnayCQpaRQThLEkk91vxr70WnsSvO0PxZ6lYJN1m5xD5V6iid4OKHbqMf2
yiYrma8Cdaf2ClQjrfgpNyXG9uB4j2uj9hSivsVTrKdhMdNID83O6ushcEhGufy9
8ADLg7w6Oe6DNQufcegNFXaVX2AqN92MlGBLEJc1WwLk3q2Nac9gT1JbxgJdZeLv
xoTwnu2/Tu1HVOUPemgG/0GwaePLFKxrhQWNIV69v17aj4QwnbbovQRKGtAP2fsk
JY6zpF5d11iGwCSS3GnoIUKn7Xo4g/V5Xf5YcP2UPAGLGGxS+zE52mVemVpY9ZqS
ZxAoPg==
-----END CERTIFICATE-----
`

	clientKey := `-----BEGIN PRIVATE KEY-----
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
yXwQ0N/qK7uMh9w0d+yac8h8bjMa
-----END PRIVATE KEY-----
`

	suites := func(scaffoldFunc func() *scaffold.Scaffold) {
		s := scaffoldFunc()
		ginkgo.It("create a SSL with client CA", func() {
			// create secrets
			err := s.NewSecret(serverCertSecret, serverCert, serverKey)
			assert.Nil(ginkgo.GinkgoT(), err, "create server cert secret error")
			err = s.NewClientCASecret(clientCASecret, rootCA, "")
			assert.Nil(ginkgo.GinkgoT(), err, "create client CA cert secret error")

			// create ApisixTls resource
			tlsName := "tls-with-client-ca"
			host := "mtls.httpbin.local"
			err = s.NewApisixTlsWithClientCA(tlsName, host, serverCertSecret, clientCASecret)
			assert.Nil(ginkgo.GinkgoT(), err, "create ApisixTls with client CA error")
			// check ssl in APISIX
			time.Sleep(10 * time.Second)
			apisixSsls, err := s.ListApisixSsl()
			assert.Nil(ginkgo.GinkgoT(), err, "list ssl error")
			assert.Len(ginkgo.GinkgoT(), apisixSsls, 1, "ssl number not expect")

			// create route
			backendSvc, backendSvcPort := s.DefaultHTTPBackend()
			apisixRoute := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2beta3
kind: ApisixRoute
metadata:
  name: httpbin-route
spec:
  http:
  - name: rule1
    match:
      hosts:
      - mtls.httpbin.local
      paths:
      - /*
    backends:
    - serviceName: %s
      servicePort: %d
`, backendSvc, backendSvcPort[0])
			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(apisixRoute))
			time.Sleep(10 * time.Second)

			apisixRoutes, err := s.ListApisixRoutes()
			assert.Nil(ginkgo.GinkgoT(), err, "list routes error")
			assert.Len(ginkgo.GinkgoT(), apisixRoutes, 1, "route number not expect")

			// Without Client Cert
			s.NewAPISIXHttpsClient(host).GET("/ip").WithHeader("Host", host).Expect().Status(http.StatusBadRequest).Body().Raw()

			// With client cert
			caCertPool := x509.NewCertPool()
			ok := caCertPool.AppendCertsFromPEM([]byte(rootCA))
			assert.True(ginkgo.GinkgoT(), ok, "Append cert to CA pool")

			cert, err := tls.X509KeyPair([]byte(clientCert), []byte(clientKey))
			assert.Nil(ginkgo.GinkgoT(), err, "generate cert")

			s.NewAPISIXHttpsClientWithCertificates(host, true, caCertPool, []tls.Certificate{cert}).
				GET("/ip").WithHeader("Host", host).Expect().Status(http.StatusOK)
		})
	}

	ginkgo.Describe("suite-ingress: scaffold v2beta3", func() {
		suites(scaffold.NewDefaultScaffold)
	})
	ginkgo.Describe("suite-ingress: scaffold v2", func() {
		suites(scaffold.NewDefaultV2Scaffold)
	})
})
