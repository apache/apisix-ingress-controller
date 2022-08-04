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

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("suite-ingress-resource: SSL Testing", func() {
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
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixTlsCreated(1))
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
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixTlsCreated(1))
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
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixTlsCreated(1))
			tls, err := s.ListApisixSsl()
			assert.Nil(ginkgo.GinkgoT(), err, "list tls error")
			assert.Len(ginkgo.GinkgoT(), tls, 1, "tls number not expect")
			assert.Equal(ginkgo.GinkgoT(), tls[0].Snis[0], host, "tls host is error")

			// delete ApisixTls
			err = s.DeleteApisixTls(tlsName, host, secretName)
			assert.Nil(ginkgo.GinkgoT(), err, "delete tls error")
			// check ssl in APISIX
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixTlsCreated(0))
		})
	}

	ginkgo.Describe("suite-ingress-resource: scaffold v2beta3", func() {
		suites(scaffold.NewDefaultV2beta3Scaffold)
	})
	ginkgo.Describe("suite-ingress-resource: scaffold v2", func() {
		suites(scaffold.NewDefaultV2Scaffold)
	})
})

var _ = ginkgo.Describe("suite-ingress-resource: ApisixTls mTLS Test", func() {
	// RootCA -> Server
	// RootCA -> UserCert
	// These certs come from mTLS practice

	rootCA := `-----BEGIN CERTIFICATE-----
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
-----END CERTIFICATE-----`

	serverCertSecret := `server-secret`
	serverCert := `-----BEGIN CERTIFICATE-----
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
+KFB4CWLG1UPFpkLB22o8CWMnFdn2fji3jk=
-----END CERTIFICATE-----`

	serverKey := `-----BEGIN RSA PRIVATE KEY-----
MIIJJwIBAAKCAgEAv6korMaGhJUZ46YuCXuZ13s+AQC/Y2utw/T1/qwQRNzQO/RQ
6hFSNTunws2AvYzKzxWyKMK+5rHrkbQ2s9UCtpDd1LLpA0W8t4dmwe3geDtlwW31
tfeR1b0Cikg2YSiDpyS3M/lQPhaKa1mnM66ntJ3QggI3HDZIlbQslG8L6/Fw6ozP
UQTIo+p5elDRpla0srO68+mQzhXTEoIzkDaiUb3RJkoM8g34H4BXikh73u527iDx
KxdHoSrSMMQSkn4KNmJ+lSll2UiAbL87fJhLlhqJ4hn3xzOdRCxYIC7JgXwBIMHE
ShsxafvH1p3mZJp33n8K/af7SVr7ge3a50xbNQuiXg3xR1pUy7BwYK37KMuHGebP
S415m4E71X6bN/0pL2M5dqu+Vj4dwSNjPVIHEzPDZL8gibRG+eBCr3gC3JW+l1Ac
ngltY+GEcbEgTacSRzrHKA+f/WVYBsAkAIvtiBhD8+MB5MLXlnrXISPIwMTZSYJ8
ng2XO1WMt2PudCDRdSOgQ6c1xCNUO6QfAtBIzU7G24iE1WxZbBahJiH2JFK6OWPx
j+0yv4AQiY9Ng/MQ811qTrXs/M9cEXyu+xft5NPFcIhvwgFHbr5b8ABj+viSDan9
MdYdT6OlE0DrCJQZPaCwO8OEpl1+hV9vz63peWK+lJVYGcjA6gupXLj5rOUCAwEA
AQKCAgBHkItOkEZ0RRRIq6lvAwb7rdoGF9he8DsO+23LLUZZ4DWk3WJFNDiFBgRr
Ob7DiEnGL2y5yZXsoCy82BTA6126+7bJEBDvlt+Ti+xzpzX0zwD8y+k+i/WZYJ0N
M0+S0cTu6Ue7EXHD7Ti8Qtqq8qFOUMslcFxRnXdW5tLqjdhevmWSPwe+UdH1Wr0H
ThwqRx/rxi6dmu3l9cI9m/5S8AOGECGDcY0J6OtoH80QJmaSZGpmGkjS9Tta05lu
ehgRORzpF7f6TF3qVycU9AbrTBaVMs2fbmDVsdEcPo6dXbsCLWJib9eycBrwXwJM
geMgV4lAvCFHe6zZxC47YqwlR56aNkzGdsq0RLW8gA0rVCKCTBUCqv8yOuwGneNp
9sxbxYgEKj/MjKOgbclytYZWrGpMMiHqMgNkVEIbGmZ61jKslioAfse+wp9eQto4
eerJDOtjJQTQPAqQpc/ES429IfYlrC58taxERslnYV1HjBAmEvICljQObLxB94mK
90Fm3Uaxtg/H0b4sVnr2XGEIRnOXY3rRuBAGKqdGsE+4Sf0SlbUFCLVHZTg7CJ2R
TJnzQ0yq0OkDsRwc/BYdCIFn1JBbt2tR3IUkpvY3KOvnEo2chA6blJpTSiu3sxBu
LdLLj7GSn3IamtSljsC8we1ZBUWYRn2HAPUscsbcwy6BAjfr1QKCAQEA+t84bB3H
aeaJA5ouYFYGF5dZRaZyo1cR0OSvy5MSUCbDNe5nTFdMHKNyTcxekgC0GUC3rzeG
p1FFy00bWpDIU+y09uYr/sEkGt7gc0CmQ/zK1V1y3e3aeZIN15QwQGqWyF3j2hAR
is01EZv/1985FvUL2qN24gn47XhrK4z1XGRLuSxxbMiAJmcjDOAokgRQjRE+Sm5M
1CnjSEVFlRZELFuNTAv83VxPDJjG9hebO/oiMZcJtUXeff69J4qC/j2nEVgg75dh
qMb8NQRrTbcuOFZN9yuxplL3TluwocIkEHU+/EHDEAOcWcLgqNxOqC+WesBI+IW0
tKotb7tNZ2I+3wKCAQEAw5QYIYwUVEt5ZpeUzecvH/Jm3Enhxi0xPSkWYbZYBF0y
8KYo4iLkv3zK5VqnBmYWKZWI0HS1mS3ol7Wy8rH1Xcw/wqEal2B33hM9p2pkjW/5
3Y/mSnMUHT2jgk01ncwyaOOjAUQoEgmq2iznTjzie5+z8KcRYZvl4MGYbEPuHGaP
kCkWI06qTcIJ659sypj+8pUukrEWXBn4Tr82Q2aMljoSFYIgrr08NWeb06/eeI1F
Fmb07hNRvmmsLj4V469WsPTFrRaBLCBDrbnQOWZU00hBAltp+4PQ3wpfuKTVt3xX
wzDX07jwyb4Ys6q1NSbIjZv2vrRQbaKEDfS7cUJAuwKCAQBnJNms0f2IE+mnWn/Q
ye2NS4O/uDSP5Z+ElFGW0GwKGjXOeats3sODTswTIoCLZNCnRU2AM8MgDbE1agli
Df7fSoYIsQ/LmRtAFPyRRjZV45x9ZwNwLXfS3fLk/J9uDKTb0oZ4xHyB5eb4y3vA
BJ4TS0LJbMXXH6SB9i2R5U2H5BCiHJyxzimqIGNvysXDaxS3OyyyK3FZFbPFpf16
04HJ/wY0CwW2+Vni4vmCeqgvW6MtYlzyc7yLbu3UUQWUhEKpReOcvk+/tbhCEAQS
GstdDFbX1dYffSMCy33us8RiI+J2ko8hiWqCGTaHFrUcPxyOcXpO+6IVWZZ+xrKH
XARfAoIBAAGBzaHMi4eOwVO6DUp84o8TdhlydEvrozp+a467Mfhuo2rZTO1ZKXwU
QRf9V9YjyT3uygwZKiERCn7IxqU6G9LqNP+R8DuEYcgTS+FTX4z7dOhxKGwgcOI8
zFq/r48UuLq4LlRfKxPggTGHMQ0YSQJ8240aLHcdFWti8oK7D0WmwKpytpn1DDjn
Kt5m7xaskSJbZe15cdup05D/xjJEwwaRUfxacVgHW0RqFPhPnZ4+MG8YwgBno7Sc
6de9YLvNaRSZ/j/0MXCemwbmrKUUlci/AMk83Rc0D9L4KH6qvn7YdXCqmq8l+K0F
SvlvclADiX4V2pPjnc8KdowI+7zGrusCggEAQZ8TlrcRZ3RnGUuQ9pVgi1+JFehN
qLmIXu89grekMAdHs+wsJ9L5XSEyI2LQDTo0Kmw396WFOiMHXO+hd4hu55j6KIC4
9kqt8bX1PthAlwuZmThQQ29HJUX+YHZ0G98Zu7FSoHDokHj+E1dnjF/QO6w6nvB9
r2LfnMEchQOHG+fVedkkxy+jckBqA9Qf2tsDYG+KYUj/nVsBQzfRiC4bp/04x16b
j+0ntnGGA9b4ZaWhybh4UKEY9YwUOAvq/Y3gHqwaWDZscd/5s8MHZwKNfKriateP
6tz2gipPdAsLQNaCS1d1OXHgh960gKziQNZM/wNgJQO/7ZeGvMkk6ml+0w==
-----END RSA PRIVATE KEY-----`

	clientCASecret := `client-ca-secret`
	clientCert := `-----BEGIN CERTIFICATE-----
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
PLApoDRtuZpYJBY4acuPqyBLs+xn0fZ5pmM=
-----END CERTIFICATE-----`

	clientKey := `-----BEGIN RSA PRIVATE KEY-----
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
-----END RSA PRIVATE KEY-----
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
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixTlsCreated(1))

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
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1))

			// Without Client Cert
			// From APISIX v2.14, If the client does not carry a certificate request, it will fail directly.
			// Previous versions would return 400.
			// s.NewAPISIXHttpsClient(host).GET("/ip").WithHeader("Host", host).Expect().Status(http.StatusBadRequest).Body().Raw()

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

	ginkgo.Describe("suite-ingress-resource: scaffold v2beta3", func() {
		suites(scaffold.NewDefaultV2beta3Scaffold)
	})
	ginkgo.Describe("suite-ingress-resource: scaffold v2", func() {
		suites(scaffold.NewDefaultV2Scaffold)
	})
})
