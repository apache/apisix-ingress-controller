package ingress

import (
	"time"

	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"

	"github.com/api7/ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.FDescribe("SSL Testing", func() {
	s := scaffold.NewDefaultScaffold()
	ginkgo.It("create a SSL from ApisixTls ", func() {
		secretName := "test-atls"
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
		err = s.NewApisixTls(tlsName, secretName)
		assert.Nil(ginkgo.GinkgoT(), err, "create tls error")
		// check ssl in APISIX
		time.Sleep(10 * time.Second)
		tls, err := s.ListApisixTls()
		assert.Nil(ginkgo.GinkgoT(), err, "list tls error")
		assert.Len(ginkgo.GinkgoT(), tls, 1, "tls number not expect")
	})
})
