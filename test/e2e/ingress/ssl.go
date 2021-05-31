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

var _ = ginkgo.Describe("SSL Testing", func() {
	s := scaffold.NewDefaultScaffold()
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
})

var _ = ginkgo.Describe("ApisixTls mTLS Test", func() {
	// RootCA -> Server
	// RootCA -> UserCert
	// These certs come from mTLS practice

	rootCA := `-----BEGIN CERTIFICATE-----
MIIF9zCCA9+gAwIBAgIUFKuzAJZgm/fsFS6JDrd+lcpVZr8wDQYJKoZIhvcNAQEL
BQAwgZwxCzAJBgNVBAYTAkNOMREwDwYDVQQIDAhaaGVqaWFuZzERMA8GA1UEBwwI
SGFuZ3pob3UxGDAWBgNVBAoMD0FQSVNJWC1UZXN0LUNBXzEYMBYGA1UECwwPQVBJ
U0lYX0NBX1JPT1RfMRUwEwYDVQQDDAxBUElTSVguUk9PVF8xHDAaBgkqhkiG9w0B
CQEWDXRlc3RAdGVzdC5jb20wHhcNMjEwNTI3MTMzNjI4WhcNMjIwNTI3MTMzNjI4
WjCBnDELMAkGA1UEBhMCQ04xETAPBgNVBAgMCFpoZWppYW5nMREwDwYDVQQHDAhI
YW5nemhvdTEYMBYGA1UECgwPQVBJU0lYLVRlc3QtQ0FfMRgwFgYDVQQLDA9BUElT
SVhfQ0FfUk9PVF8xFTATBgNVBAMMDEFQSVNJWC5ST09UXzEcMBoGCSqGSIb3DQEJ
ARYNdGVzdEB0ZXN0LmNvbTCCAiIwDQYJKoZIhvcNAQEBBQADggIPADCCAgoCggIB
ALJR0lQW/IBqQTE/Oa0Pi4LlmlYUSGnqtFNqiZyOF0PjVzNeqoD9JDPiM1QRyC8p
NCd5L/QhtUIMMx0RlDI9DkJ3ALIWdrPIZlwpveDJf4KtW7cz+ea46A6QQwB6xcyV
xWnqEBkiea7qrEE8NakZOMjgkqkN2/9klg6XyA5FWfvszxtuIHtjcy2Kq8bMC0jd
k7CqEZe4ct6s2wlcI8t8s9prvMDm8gcX66x4Ah+C2/W+C3lTpMDgGqRqSPyCW7na
Wgn0tWmTSf1iybwYMydhC+zpM1QJLvfDyqjp1wJhziR5ttVe2Xc+tDC24s+u16yZ
R93IO0M4lLNjvEKJcMltXyRzrcjvLXOhw3KirSHNL1KfrBEl74lb+DV5eU4pIFCj
cu18gms5FBYs9tpLujwpHDc2MU+zCvRmSPvUA4yCyoXqom3uiSo3g3ymW9IM8dC8
+Bd1GdM6JbpBukvQybc5TQXo1M75I9iEoQa5tQxAfQ/dfwMjOK7skogowBouOuLv
BEFKy3Vd57IWWZXC4p/74M6N4fGYTgHY5FQE3R4Y2phk/eaEm1jS1UPuC98QuTfL
rGuFOIBmK5euOm8uT5m9hnrouG2ZcxEdzHYfjsGDGrLzA0FLu+wtMNBKM4NhsNCa
d+fycLg7jgxWhaLvD5DfkV7WFQlz5LUceYIwYOyhD/chAgMBAAGjLzAtMAwGA1Ud
EwQFMAMBAf8wHQYDVR0RBBYwFIISbXRscy5odHRwYmluLmxvY2FsMA0GCSqGSIb3
DQEBCwUAA4ICAQCNtBmoAc5tv3H38sj9qhTmabvp9RIzZYrQSEcN+A2i3a8FVYAM
YaugZDXDcTycoWn6rcgblUDneow3NiqZ57yYZmN+e4mE3+Q1sGepV7LoRkHDUT8w
jAJndcZ/xxJmgH6B7dImTAPsvLGR7E7gffMH+aKCdnkG9x5Vm+cuBwSEBndiHGfr
yw5cXO6cMUq8M6zJrk2V+1BAucXW2rgLTWy6UTTGD56cgUtbStRO6muOKoElDLbW
mSj2rNv/evakQkV8dgKVRFgh2NQKYKpXmveMaE6xtFFf/dd9OhDFjUh/ksxn94FT
xj/wkhXCEPl+t7tENhr2tNyLbCOVcFzqoi7IyoWKxxZQfvArfj4SmahK8E/BXB/T
4PEmn8kZAxaW7RmGcaekm8MTqGlhCJ3tVJAI2vcYRdd9ZHbXE1jr/4xj0I/Lzglo
O8v5fd4zHyV1SuZ5AH3XbUd7ndl9yDoN2WSqK9Nd9bws3yrf+GwjJAT1InnDvLg1
stWM8I+9FZiDFL255/+iAN0jYcGu9i4TNvC+o6qQ1p85i1OHPJZu6wtUWMgDJN46
uwW3ZLh9sZV6OnhbQJBQaUmcgaPJUQqbXNQmpmpc0NUjET/ltFRZ2hlyvvpf7wwF
2DLY1HRAknQ69DuT6xpYz1aKZqrlkbCWlMMvdosOg6f7+4NxdYJ/rBeS6Q==
-----END CERTIFICATE-----
`

	serverCertSecret := `server-secret`
	serverCert := `-----BEGIN CERTIFICATE-----
MIIF/TCCA+WgAwIBAgIUBbUP7Gk0WAb/JhYYcBBgZEgmhbEwDQYJKoZIhvcNAQEL
BQAwgZwxCzAJBgNVBAYTAkNOMREwDwYDVQQIDAhaaGVqaWFuZzERMA8GA1UEBwwI
SGFuZ3pob3UxGDAWBgNVBAoMD0FQSVNJWC1UZXN0LUNBXzEYMBYGA1UECwwPQVBJ
U0lYX0NBX1JPT1RfMRUwEwYDVQQDDAxBUElTSVguUk9PVF8xHDAaBgkqhkiG9w0B
CQEWDXRlc3RAdGVzdC5jb20wHhcNMjEwNTI3MTMzNjI5WhcNMjIwNTI3MTMzNjI5
WjCBpTELMAkGA1UEBhMCQ04xETAPBgNVBAgMCFpoZWppYW5nMREwDwYDVQQHDAhI
YW5nemhvdTEcMBoGA1UECgwTQVBJU0lYLVRlc3QtU2VydmVyXzEXMBUGA1UECwwO
QVBJU0lYX1NFUlZFUl8xGzAZBgNVBAMMEm10bHMuaHR0cGJpbi5sb2NhbDEcMBoG
CSqGSIb3DQEJARYNdGVzdEB0ZXN0LmNvbTCCAiIwDQYJKoZIhvcNAQEBBQADggIP
ADCCAgoCggIBAMZRPG8gUrOGM4awnV6D8Ds0Xb6jVbiGkx+1YsvPx5oIE4AswJ0l
y6zqhBFnpQozFG63KfsCA6U36/Dty3rIbJzsbO7YaOMJItoiQgqdqF2nrmPpmpCQ
uLGKaVvriRCD55NEmFQPshlRfcU5/EEreNKbRve3zEKHRpCDBZ2Myvrpt3CCVy6D
MbLllbjUvaedrnQxlmI5d7x3UCe4Eunq8vn7c0p4frA1n8TxbX0M4Yr9g3YEEqCv
Q3/9jU4hI5CvujCp+u79EavJZfsaEv3RYgHkoEh7q+OEkUajWXKj4WynizraWsHv
+LvK9pfI300p1HSKK4FqonvW79anRNbK+8BqV4Wt5aBeFU/rW2jHtJxcl1OLRrrh
wftCP5W7vSjvJes5wPDZjDEyv8WP1Aa6yWeGHHtIwrAHPr7556F/JAQS6IPBQQ5U
X45DD2aNXME9xZKdBtyMovItjZm31UUsvoF+YtpAOmbEkX4lMznUO3XZJjM5HWSq
WYyzmFsw+pJEwhXRo4GfSfCHfiZQ6imTLJ7OsZzo9bvmxyfI0kVLe3h3iCe+qYeT
f5AJU6v5vv3thCtfgbxYP2P8b+0MIrfr05e6dCDXbIri1z+nprzWYmyCrZ6H4hVk
DzMktFUlkqenvnsJ2iOV2AZw0Hlk2bwe4zSumzqoIp8Yk/kxbfxhQqr5AgMBAAGj
LDAqMAkGA1UdEwQCMAAwHQYDVR0RBBYwFIISbXRscy5odHRwYmluLmxvY2FsMA0G
CSqGSIb3DQEBCwUAA4ICAQCDDfETCEpWB/KRQZo2JF8n4NEDTeraQ85M3H5luJHp
NdJO4oYq3n8B149ep4FcEYdO20pV+TMeMNWXMfhoRIpGx95JrLuLg6qnw6eNdErn
YupHMC2OEoEWVcmI052LDJcXuKsTXQvU4OeEL2dX4OtNJ+mRODLyh40cg7dA3wry
kGLiprRlLQtiX8pSDG30qPZexL1LcFzBQajriG05QUrJW6Rvbq1JTIlyp7E1T86f
Xljq0Hdzqxy+FklYcAW5ZAxgkQlMmVdTlvDXlD/hQLEQIHGHiW6OMLp8WrnJP6b0
D2HqWmOwuEzqSgXSK0N89rpiWP1FKCpyiKVcsawDNfOpePVuthommVEc2PxacyHf
UCC9V0MS0ZzQ63Tnz2Tja8C6/kMyVX226KQKhcoDxDoS0mQrI96/VXcglwP5hMjF
joth1T1qRVu6+NQmvFPaNjbzWJ+j1R99bnYGihPeLdqDSUxNosV3ULG8T4aN6+f8
hApiqg2dkLJQr8zWf6vWXMlREdPEovb2F7P0Lfn0VeOSRXDUIdqcoRHONi8bWMRs
fjPtGW00Tv8Jg21c9vc8Zh/t1w3wkXQhqYiBMt5cYe6WueIlXdjF7ikSRWAHTwlw
Bfzv/vMftLnbySPovCzQ1PF55D01EWRk0o6PRwUDLfzTQoV+bDKx82LxKtZBtQEX
uw==
-----END CERTIFICATE-----
`
	serverKey := `-----BEGIN RSA PRIVATE KEY-----
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
-----END RSA PRIVATE KEY-----
`
	clientCASecret := `client-ca-secret`
	clientCert := `-----BEGIN CERTIFICATE-----
MIIGDDCCA/SgAwIBAgIUBbUP7Gk0WAb/JhYYcBBgZEgmhbIwDQYJKoZIhvcNAQEL
BQAwgZwxCzAJBgNVBAYTAkNOMREwDwYDVQQIDAhaaGVqaWFuZzERMA8GA1UEBwwI
SGFuZ3pob3UxGDAWBgNVBAoMD0FQSVNJWC1UZXN0LUNBXzEYMBYGA1UECwwPQVBJ
U0lYX0NBX1JPT1RfMRUwEwYDVQQDDAxBUElTSVguUk9PVF8xHDAaBgkqhkiG9w0B
CQEWDXRlc3RAdGVzdC5jb20wHhcNMjEwNTI3MTMzNjMxWhcNMjIwNTI3MTMzNjMx
WjCBtDELMAkGA1UEBhMCQ04xETAPBgNVBAgMCFpoZWppYW5nMREwDwYDVQQHDAhI
YW5nemhvdTEfMB0GA1UECgwWQVBJU0lYLVRlc3QtUm9vdC1Vc2VyXzEaMBgGA1UE
CwwRQVBJU0lYX1JPT1RfVVNFUl8xJDAiBgNVBAMMG0JpZ01pbmdfIDY1NDMxMTEx
MTExMTExMTExMTEcMBoGCSqGSIb3DQEJARYNdGVzdEB0ZXN0LmNvbTCCAiIwDQYJ
KoZIhvcNAQEBBQADggIPADCCAgoCggIBAL41i+W8fgqtKButnkOa0qGApPJ7HSUI
gxBt6Tb7mR8eka8vno8hIE42hwMXYoIiO1FgM9M9fPN8vzTnWr2iTAXbw/qXdn14
RT01JrGzj1OU3M9U0RMmDWVs8VOY3ncdcdPoqctMNr2K+6wlN6EpG1DQDUEvB0Yy
qZpFWpuZP0akzjFqeJNSo3b8hP4J1nsWBxKt6TRInC60jT1mhKCjTB1IsXVC+5SY
f0FhMN2ZQPgCyIIq+j5S7tHta4Qs+UDNY0whIMBkY3ZAGPzUBFExwrX7bGNbeJYz
DahNAKiEj5dlieRvzTn5UisSv5EzDUONLSluEnCU1Ghd/dfZtZwlMBU6OIflwm6W
X7HePD0G8XEDe5qbU/tg/62j+A1hMS2gSlibNisyvNwMNz6Imc3CnIFUL2mTsgl1
bCBLfOXDfrjte5Txnt26ayN0+WhmVSts0+ln6vnGtgVGqH0bmp95yMRYPej+6ARE
MtfvrNA7iZORypspv2NVyV11b6Jx3o/pJOAq9dmAsKvJ6hK0IVf/55t3ZtzRa/fY
w0f78KbWuEc7qmx2tBxnEfppb5iB/RFmI2bDaAEA0lDQgAqmiAu1utetfqsCwA1m
3VXGp8Vp+lexA/M3Bkb5BHa6C11QDHvGjUVwduOymLAHKumvdgf7TJFnlhpg6jYN
unO58bRSTI6LAgMBAAGjLDAqMAkGA1UdEwQCMAAwHQYDVR0RBBYwFIISbXRscy5o
dHRwYmluLmxvY2FsMA0GCSqGSIb3DQEBCwUAA4ICAQCoTBvw11aKah2cuB4XplUB
nmkrWhfLNFJLVrU9jIISP9Wkl3s3PcM+aEWygb/1roqbMqNOgrzDVgGRRFCiS6qi
himUuTJhIBI6TF1PE+XW3gTFWBXkAZ7MzpbS8oP1PehlY3XXKNZgxZi3XaDI8Hfw
5MWBGNbk8tegn8bvYQUz2VxmCo6zufCkj4ADjw2zhiyKBKuHTzg48w66Wn4jLhlK
p91HHrK0lEOIJ4pFmBUpBsSJMlBMbfrzBF87xQhpDO3ScFfCWUatShmXsPMJU0F0
DEuTnaHUefUf/F9wUGNcA4yQ4pH8SxVpRHmrWE8U4uSXpz1bx4ChZurJ9mPzrj9h
U9c/d9F5WndZNPcR1R8Tbzhk/R8GImVR3Lt59cW+SN/+4JVFy+Hye2yslGFn2CAJ
ofNxjLb9OE6+EE3SWW0B5CZSWBS49gtdTW0ApOjIRJU2zipxcjnNf00GFoIoCxjk
Z4eBQz9WVUM9KSrJIQSLZQd35tZAOp0BuwWho0+w8lXUchSqT7oRA7+szZldWF0j
HKPIMJ0iVWmXuZjsS8q8NBIt4DuBcqpevlol5KRXv6tJy4IBVAVEIBdeXotvdxKE
bncvZ6xo9A/waUU7tEyzv34usxefrWxtSlOA1G0Jj4nb5gKPHjn0XIr9WI2RpovT
/XpB6QES1zoBQya3QjnDbQ==
-----END CERTIFICATE-----
`

	clientKey := `-----BEGIN RSA PRIVATE KEY-----
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
-----END RSA PRIVATE KEY-----
`

	s := scaffold.NewDefaultV2Scaffold()
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
apiVersion: apisix.apache.org/v2alpha1
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
    backend:
      serviceName: %s
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
})
