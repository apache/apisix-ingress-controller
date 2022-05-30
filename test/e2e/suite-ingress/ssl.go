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

	"github.com/onsi/ginkgo"
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
MIIGODCCBCCgAwIBAgIUMpUTqqBsbHoEmS3nu1SMJqZu3/cwDQYJKoZIhvcNAQEL
BQAwgZsxCzAJBgNVBAYTAkNOMRAwDgYDVQQIDAdCZWlqaW5nMRAwDgYDVQQHDAdC
ZWlqaW5nMRYwFAYDVQQKDA1BcGFjaGUtQVBJU0lYMRQwEgYDVQQLDAtBUElTSVgt
VGVzdDEUMBIGA1UEAwwLQVBJU0lYLlJPT1QxJDAiBgkqhkiG9w0BCQEWFWRldkBh
cGlzaXguYXBhY2hlLm9yZzAeFw0yMjA1MzAxNjU1MTRaFw0zMjA1MjcxNjU1MTRa
MIGbMQswCQYDVQQGEwJDTjEQMA4GA1UECAwHQmVpamluZzEQMA4GA1UEBwwHQmVp
amluZzEWMBQGA1UECgwNQXBhY2hlLUFQSVNJWDEUMBIGA1UECwwLQVBJU0lYLVRl
c3QxFDASBgNVBAMMC0FQSVNJWC5ST09UMSQwIgYJKoZIhvcNAQkBFhVkZXZAYXBp
c2l4LmFwYWNoZS5vcmcwggIiMA0GCSqGSIb3DQEBAQUAA4ICDwAwggIKAoICAQCm
KPSfaMHBxdvLuoF+h2zj88irpb5XNpzoXLshfhqv01Y1rC8wiPu0TI1bkfxD9X4G
Pl0f1cV5Q1LosXa8BOEpuaOwYceMQzl7xi+dhg7ujBCyQwjFW0lP7F29Rts/dBNP
NB+fywKvwrq3EglFygR8649uZTfBPbuTCaKCrmVpSGJpB8AH0+yTBOgXKrIsN0Db
4tdYu9feN8wurz5mki+dy2mnUC7lHOISMMGmUvpMs18el0V+8p1wse63N+Q0lqlt
8mTikO3187+JVo/+gSEqDAsppOJRbA12NjYjk+0a76XAPtMUr/US1MpkeYqu/HGl
JdeIWi+Sn0O1RkGJVRI+bSKgE7PdrVfehMuYoqlLheeemoNeHdyHM8V2EQs+BPwF
j0aw+VVzcuTZLibliB49IQu+6KBZdKWmGvLsURglT6hHfvVs1OXM/dvwXYsDnjqk
gtcRaukJufhcasi7KF8Hcq57jk6svQxWK3b1ckp1EtUZlx57cU0z3aZdzKHFO4pV
YRisZVq+Q6xqIDL8o4PVtqdJBrtQ5b18Nwi74gleOL92/6SMSBZAawyxr2sNXEzM
BkzkbwOMp/C/tVKCVFs5F/N8W6A8rKKh+hVw21F3jco4VSvL4qT8uokL/4k0rla+
Ns/adh+aHt2RUQKs6zG1990nAu3xHCjFFeWuf0aqZQIDAQABo3IwcDAdBgNVHQ4E
FgQUI+Amr2EGnBMfTCwV2KziDWCm6HQwHwYDVR0jBBgwFoAUI+Amr2EGnBMfTCwV
2KziDWCm6HQwDwYDVR0TAQH/BAUwAwEB/zAdBgNVHREEFjAUghJtdGxzLmh0dHBi
aW4ubG9jYWwwDQYJKoZIhvcNAQELBQADggIBADseGkLzpJqVYqEMk7RmSsqh25n0
a2t1e23tbgQLqPfp3Sga3+2TF18JTgC4JkP4LUBzrVF6PGPBl5U+llHti8Yb0opg
IpQdObHm2SNcO0Z9ZvuXR2MNoxs7UiCsqEHsG9Set1RpJiS61cPHlLS5MvE3CXRB
hogcXaX7wFzizBKouE0e71Z98d4ng+CBALnd8aQmA3z88nrSfNWb67JZDFOiCK/i
wYF/0ZJzIHbQ0TifMZlVvkDDyYVmqEwYqnl+PwIw0wP8Bj6qXJMh2ckCtaBWKWfQ
OfnywU12MgtkogkOotlZmUElo7XZfWo4f2rdyh+JYlFS4eAAlGFufJM+Oj8B2zY1
abNED0mYLAWnA1H9PFYFMZfEgUQPLAyCuGZfxir3K8I9NCNuB+Daqk6p+LAzIj2O
Khj2Gi8DaRvcYVLQrAyZG1Iv3VlmCMFHdSeW8/yM7bJqDCPAM8KcOkwlflnqGD86
WafTURWxH0p1LxZQFFQed3Lnf9OJ517SpqK7ZoHpx1JqMQnVCv+bfeN+CxrHdnkO
IoDRvtmXg9AgPHtqA9x4BXy0xDkYnj1hp4PW+fBtMPiUWfWQQAg/gsI4t2TbsycB
90gU26o0VjDkGt9PRaFV7dFdA2rDP7uQ12WWQHp/YH5jQeJLPbTiNf1lh4lFXYlo
GTrJyCVogP0ykFBg
-----END CERTIFICATE-----
`

	serverCertSecret := `server-secret`
	serverCert := `-----BEGIN CERTIFICATE-----
MIIF8jCCA9qgAwIBAgIUIvpv+rL2HU0BOVsY+Y7eHuI2LGMwDQYJKoZIhvcNAQEL
BQAwgZsxCzAJBgNVBAYTAkNOMRAwDgYDVQQIDAdCZWlqaW5nMRAwDgYDVQQHDAdC
ZWlqaW5nMRYwFAYDVQQKDA1BcGFjaGUtQVBJU0lYMRQwEgYDVQQLDAtBUElTSVgt
VGVzdDEUMBIGA1UEAwwLQVBJU0lYLlJPT1QxJDAiBgkqhkiG9w0BCQEWFWRldkBh
cGlzaXguYXBhY2hlLm9yZzAeFw0yMjA1MzAxNzEzMThaFw0zMjA0MDcxNzEzMTha
MIGbMQswCQYDVQQGEwJDTjEQMA4GA1UECAwHQmVpamluZzEQMA4GA1UEBwwHQmVp
amluZzEWMBQGA1UECgwNQXBhY2hlLUFQSVNJWDEUMBIGA1UECwwLQVBJU0lYLVRl
c3QxFDASBgNVBAMMC0FQSVNJWC5ST09UMSQwIgYJKoZIhvcNAQkBFhVkZXZAYXBp
c2l4LmFwYWNoZS5vcmcwggIiMA0GCSqGSIb3DQEBAQUAA4ICDwAwggIKAoICAQDb
TtGOHufWjgzJgQP2jDJilde68NhQIrPU1HXb3mCmMuoRgeiD2u63N8GlicaP+sZg
/U1DhcC/pVl4dd71ZRaojccAIYCrmWoujF+yMqdIH6LmTsO7cUUlz3PzU5YQQf5Z
PGvRZvkSVmvzJ6neVFX0qp/ZKlFdSbdFve9RX4dglmd0lnBLCCXfdSRdWf5g25TS
wzp0C4E+BsyjZBc7gVk/DUgkd0IoDE2X3aBcpaENTGIw0AXQL0Rd4mAnWDKEuOyw
Kmrw/2qUXbHJtQyoyirdRvy18wY1avgKpinOVJu3LtM1wj6ZhlatG2A3p0XjdICi
oqH5I4YIXzixQItyQ5m65rqxTdb+ZHXcNsc9YEYmWPDplrRibu1guKkEWk0bIMz1
bKJV0ff7bIjhYsFO2Fa4C9MCXQlCRbpUdBjftWCwQYjGKyiCU9+29OclDGNHtul4
+q4f39mMlqEmQObD6muGx2H7yx/MalsC85ygsYgvAu+DrszNvrBCwEMNXSozj4fL
zJ6a1yEVXVtyuThT2+utBvEjYE/cZyMJUIt40DR2LRI+bZH33wUhkbsIVi9gprEX
v60pXgwZUqYrtTCCzKec+FoHA8MjsPoqBlxrWIXy/kF25SZIFw7pG61vhYbEDhuQ
0HuN9U1d3IG16GXc2jx6SUVMsglcBSiLZZv8NUkG4QIDAQABoywwKjAJBgNVHRME
AjAAMB0GA1UdEQQWMBSCEm10bHMuaHR0cGJpbi5sb2NhbDANBgkqhkiG9w0BAQsF
AAOCAgEAElDomrzE3gTtGsr7cpSQdvd8PA2HTadFv5mwdQZDvEWtZ3WbVoxtkgyz
GN+fS2Q9KxvvjX7GiFSEZZ3XGXKzVBG9K/DvCUhb4stNtt1bhIw/EDNEpLIeRPJM
RV46k+50F6MfHsEZsibqbmZY6038mNiNB1QGhH2nNPBSYwY1/vtvRkK1SmMJT7Fu
0oFduRuRW95S1Dlxa/1i+r28ulsCQMWRNuisR7pbFo/T9EhYRZ5i0XwYtjtLRwA8
6AmKJPOEUwkQMS5GlURmamkaoOnEybGFBhyXCz1L/m8OsK2+eF9+y/8T65H7STTL
5e3TZNngg0CXpBNAMvk3nxi6gme1up8QnIxQ2gPmk/gosflxW6m5d4ByjK1RWahr
lzzFUo/htSDRHeC1tflFT86kQBEqsfCXPe3gpsLsMpE06oZH1gjU7DJM7dsfkU5B
nuXXvJSYdF8v6VNkO/f3HZGYs2qyd7X1s2HWdOWRj35fxiAHXkcd+pGKnqdgxvYw
T6xRwJ1JOtlyNeKog8MeY3mT3k0wA8iBLkRDNOtGdWMKacmirEn9YqJs/z7UHEOt
FCK8/aLXMdoRz4mUxXPerYwE4gA1BRyuCZjNP/d/hwdzu3Twaco2FQrwXHL5Cr/F
q6q0ReNe+087HDvwCBMV6Q3PhBJvu+XBk0L9E/I6SMS/DHKCPn4=
-----END CERTIFICATE-----
`
	serverKey := `-----BEGIN PRIVATE KEY-----
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
8e8TkMWqOas4Z9z79u9M3xv0B/EX6w==
-----END PRIVATE KEY-----
`
	clientCASecret := `client-ca-secret`
	clientCert := `-----BEGIN CERTIFICATE-----
MIIF8jCCA9qgAwIBAgIUIvpv+rL2HU0BOVsY+Y7eHuI2LGQwDQYJKoZIhvcNAQEL
BQAwgZsxCzAJBgNVBAYTAkNOMRAwDgYDVQQIDAdCZWlqaW5nMRAwDgYDVQQHDAdC
ZWlqaW5nMRYwFAYDVQQKDA1BcGFjaGUtQVBJU0lYMRQwEgYDVQQLDAtBUElTSVgt
VGVzdDEUMBIGA1UEAwwLQVBJU0lYLlJPT1QxJDAiBgkqhkiG9w0BCQEWFWRldkBh
cGlzaXguYXBhY2hlLm9yZzAeFw0yMjA1MzAxNzE5MDNaFw0zMjA0MDcxNzE5MDNa
MIGbMQswCQYDVQQGEwJDTjEQMA4GA1UECAwHQmVpamluZzEQMA4GA1UEBwwHQmVp
amluZzEWMBQGA1UECgwNQXBhY2hlLUFQSVNJWDEUMBIGA1UECwwLQVBJU0lYLVRl
c3QxFDASBgNVBAMMC0FQSVNJWC5ST09UMSQwIgYJKoZIhvcNAQkBFhVkZXZAYXBp
c2l4LmFwYWNoZS5vcmcwggIiMA0GCSqGSIb3DQEBAQUAA4ICDwAwggIKAoICAQCu
X3t5mXnSMitw+D3NbRTUn0rgOU5VUleIcVaXzqis575M8gi/k0nfvAOUGG0CFi41
RQpP5mLDTCqugbhI/Cbqwzf576qzf+IqArU89bHOEczZR3wXuOdIK+58LvQLLwFH
k+5UttwDQb+hlQRefZBIastoH0tzh66/51cHn2LXH0efhu6Oy3qHz4ur/lFzd3QT
b5yXqZ2n3s2uEig5xiGUDQDR7Ot1nrnUq4vGD6w0ZKXz0ZgNQffXRZs8jFClVLTH
sIDv5TaiPtomvMMg5f7qS5lRKIEmN/s1U09oKLW9ZAasZ/Jn0asxOlTxVIMG8Cxc
WLIQY6+9Xf8giMN3FSu3EyEgZVKSM29CkjfxsGCjQnhTZVuGpYhaoB+KVy/23Nqi
RjLqgiMxlo4xZzj3STSIjkOaEamlCLBwCfs9mVl20UIowE4lL3RmOG8bipSVwjo3
qrbltZw5ngdyzCrTdO3hopg9IME4YJqO2BVUaDkThMQWjZItwp1PFdpm+Gl2rsMM
xO8OYde0pFu8kZTmJZ7HE9+L9q2r6C6uDZrh6C0QJo+jwEaU03Fv/6YJUpC7lVRA
XFpuf3EAzY4mZ+uzw66QljE6QIzpyA0fU4B2exiHrhBL9NnP2Elkvx3vq0OhlGmQ
nuFkZD3vcPhVXkJRLuDVn03jC6CXd6LY21sAUXHMzQIDAQABoywwKjAJBgNVHRME
AjAAMB0GA1UdEQQWMBSCEm10bHMuaHR0cGJpbi5sb2NhbDANBgkqhkiG9w0BAQsF
AAOCAgEAXBBZc4jLtpciwrmBVDPN62VsIujkSEAauLcuz5ieSacRDRPed88gSNpf
pjez2SjXZkSXz3oYhTvsMAfTRnI1RcpDSX3/45BKq6JZbRG6uB5/ifit+8YRRDF5
RVnP/kT1u+q9fNQvmAYELKFc1d4PGyvUGwfC67r6VeFo//YnAvbIPx5us7rW7V9V
TxIKw0iDUDfZJX4rfiZzZGvV1S3/hyrJ2Q/8jk2ODuMhSJvqVdibgJ83vUo0wiA0
AhUIoL7aSslJcv6PyZ8iHmSH187+VymWkNPmIHbMiFKD88WZga4F802QU14Zed97
nWwYEEBMWwR7347hH5Kjd6DB8A2hOv+pHU3gibaqzsyh7Scp/rzSbC9cbb4DgMl0
UrWP46RGadgFjApFTuSI9s/BTKOe9OMIGXNQYSQdymtREOg8I9X2MECQekqBpvgW
LXdJcYcJz1Clc9JCMVscXSWF2kgJWyrQ/LyAkMP9IuwGn41E7osFR7mLsMvlUjqg
gRn9YsCRCut1BfnnbbBDR6kML/ViamzH1C1UM6PHLhuiHydbvPlEGz/Hbv+MVpl2
mnkgf2syyGSM1QvxF/v5RCeqaQRNzEbz1jWtrCCvdEPArX28P4kmWl500Gsr4+DB
aBpTC1lAJu4xF77OYhl051Jk2asqX3vLnmn2Uuz0CEOZBKaMpb4=
-----END CERTIFICATE-----
`

	clientKey := `-----BEGIN PRIVATE KEY-----
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
COWOhAdZng1YeCKsTqnbXsSFyJ0G
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
			assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(apisixRoute))
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
