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
wYDVR0fBDwwOjA4oDagNIYyaHR0cDovL3JhcGlkc3NsLWNybC5n
ZW90cnVzdC5jb20vY3Jscy9yYXBpZHNzbC5jcmwwHQYDVR0OBBYEFA8nu+rbiNqg
DYmhNE0IgXx6XRHiMAwGA1UdEwEB/wQCMAAwSQYIKwYBBQUHAQEEPTA7MDkGCCsG
gOYD8kmKOsxLRWeZo6Tn8
-----END CERTIFICATE-----
`
		key := `-----BEGIN RSA PRIVATE KEY-----
uiMTxBQnK9ApC5eq1mrBooECgYB4925pDrTWTbjU8bhb/7BXsjBiesBBVO43pDYL
1AOO5EEikir239UoFm6DQkkO7z4Nd+6Ier9fncpN1p1EZtqPxT64nsUTNow/z1Pp
nUVxhqt4DT+4Vp5S7D9FQ+HagbhVInQXKXtT7FNFhpIxpRy512ElSuWvrELiZOwe
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
