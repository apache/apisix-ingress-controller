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
	"time"

	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("secret Testing", func() {
	s := scaffold.NewDefaultScaffold()
	ginkgo.It("create a SSL and then update secret ", func() {
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
		// key compare
		key_compare := "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"
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
		tls, err := s.ListApisixTls()
		assert.Nil(ginkgo.GinkgoT(), err, "list tls error")
		assert.Len(ginkgo.GinkgoT(), tls, 1, "tls number not expect")
		assert.Equal(ginkgo.GinkgoT(), cert, tls[0].Cert, "tls cert not expect")
		assert.Equal(ginkgo.GinkgoT(), key_compare, tls[0].Key, "tls key not expect")

		certUpdate := `-----BEGIN CERTIFICATE-----
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
		keyUpdate := `-----BEGIN RSA PRIVATE KEY-----
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
		// key update compare
		err = s.NewSecret(secretName, certUpdate, keyUpdate)
		assert.Nil(ginkgo.GinkgoT(), err, "create secret error")
		// check ssl in APISIX
		time.Sleep(10 * time.Second)
		tlsUpdate, err := s.ListApisixTls()
		assert.Nil(ginkgo.GinkgoT(), err, "list tlsUpdate error")
		assert.Len(ginkgo.GinkgoT(), tlsUpdate, 1, "tls number not expect")
		assert.Equal(ginkgo.GinkgoT(), certUpdate, tlsUpdate[0].Cert, "tls cert not expect")
		assert.Equal(ginkgo.GinkgoT(), keyUpdate, tlsUpdate[0].Key, "tls key not expect")

		// delete ApisixTls
		err = s.DeleteApisixTls(tlsName, host, secretName)
		assert.Nil(ginkgo.GinkgoT(), err, "delete tls error")
		// check ssl in APISIX
		time.Sleep(10 * time.Second)
		tls, err = s.ListApisixTls()
		assert.Nil(ginkgo.GinkgoT(), err, "list tls error")
		assert.Len(ginkgo.GinkgoT(), tls, 0, "tls number not expect")
	})
})
