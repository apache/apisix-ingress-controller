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
package translation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	configv2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2"
	configv2beta3 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta3"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

func TestTranslateApisixConsumerV2beta3(t *testing.T) {
	ac := &configv2beta3.ApisixConsumer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "jack",
			Namespace: "qa",
		},
		Spec: configv2beta3.ApisixConsumerSpec{
			AuthParameter: configv2beta3.ApisixConsumerAuthParameter{
				BasicAuth: &configv2beta3.ApisixConsumerBasicAuth{
					Value: &configv2beta3.ApisixConsumerBasicAuthValue{
						Username: "jack",
						Password: "jacknice",
					},
				},
			},
		},
	}
	consumer, err := (&translator{}).TranslateApisixConsumerV2beta3(ac)
	assert.Nil(t, err)
	assert.Len(t, consumer.Plugins, 1)
	cfg := consumer.Plugins["basic-auth"].(*apisixv1.BasicAuthConsumerConfig)
	assert.Equal(t, "jack", cfg.Username)
	assert.Equal(t, "jacknice", cfg.Password)

	ac = &configv2beta3.ApisixConsumer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "jack",
			Namespace: "qa",
		},
		Spec: configv2beta3.ApisixConsumerSpec{
			AuthParameter: configv2beta3.ApisixConsumerAuthParameter{
				KeyAuth: &configv2beta3.ApisixConsumerKeyAuth{
					Value: &configv2beta3.ApisixConsumerKeyAuthValue{
						Key: "qwerty",
					},
				},
			},
		},
	}
	consumer, err = (&translator{}).TranslateApisixConsumerV2beta3(ac)
	assert.Nil(t, err)
	assert.Len(t, consumer.Plugins, 1)
	cfg2 := consumer.Plugins["key-auth"].(*apisixv1.KeyAuthConsumerConfig)
	assert.Equal(t, "qwerty", cfg2.Key)

	ac = &configv2beta3.ApisixConsumer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "jack",
			Namespace: "qa",
		},
		Spec: configv2beta3.ApisixConsumerSpec{
			AuthParameter: configv2beta3.ApisixConsumerAuthParameter{
				JwtAuth: &configv2beta3.ApisixConsumerJwtAuth{
					Value: &configv2beta3.ApisixConsumerJwtAuthValue{
						Key:          "foo",
						Secret:       "123",
						PublicKey:    "public",
						PrivateKey:   "private",
						Algorithm:    "HS256",
						Exp:          int64(1000),
						Base64Secret: true,
					},
				},
			},
		},
	}
	consumer, err = (&translator{}).TranslateApisixConsumerV2beta3(ac)
	assert.Nil(t, err)
	assert.Len(t, consumer.Plugins, 1)
	cfg3 := consumer.Plugins["jwt-auth"].(*apisixv1.JwtAuthConsumerConfig)
	assert.Equal(t, "foo", cfg3.Key)
	assert.Equal(t, "123", cfg3.Secret)
	assert.Equal(t, "public", cfg3.PublicKey)
	assert.Equal(t, "private", cfg3.PrivateKey)
	assert.Equal(t, "HS256", cfg3.Algorithm)
	assert.Equal(t, int64(1000), cfg3.Exp)
	assert.Equal(t, true, cfg3.Base64Secret)

	ac = &configv2beta3.ApisixConsumer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "jack",
			Namespace: "qa",
		},
		Spec: configv2beta3.ApisixConsumerSpec{
			AuthParameter: configv2beta3.ApisixConsumerAuthParameter{
				WolfRBAC: &configv2beta3.ApisixConsumerWolfRBAC{
					Value: &configv2beta3.ApisixConsumerWolfRBACValue{
						Server: "https://httpbin.org",
						Appid:  "test01",
					},
				},
			},
		},
	}
	consumer, err = (&translator{}).TranslateApisixConsumerV2beta3(ac)
	assert.Nil(t, err)
	assert.Len(t, consumer.Plugins, 1)
	cfg4 := consumer.Plugins["wolf-rbac"].(*apisixv1.WolfRBACConsumerConfig)
	assert.Equal(t, "https://httpbin.org", cfg4.Server)
	assert.Equal(t, "test01", cfg4.Appid)

	ac = &configv2beta3.ApisixConsumer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "jack",
			Namespace: "qa",
		},
		Spec: configv2beta3.ApisixConsumerSpec{
			AuthParameter: configv2beta3.ApisixConsumerAuthParameter{
				HMACAuth: &configv2beta3.ApisixConsumerHMACAuth{
					Value: &configv2beta3.ApisixConsumerHMACAuthValue{
						AccessKey: "foo",
						SecretKey: "bar",
					},
				},
			},
		},
	}
	consumer, err = (&translator{}).TranslateApisixConsumerV2beta3(ac)
	assert.Nil(t, err)
	assert.Len(t, consumer.Plugins, 1)
	cfg5 := consumer.Plugins["hmac-auth"].(*apisixv1.HMACAuthConsumerConfig)
	assert.Equal(t, "foo", cfg5.AccessKey)
	assert.Equal(t, "bar", cfg5.SecretKey)

	// No test test cases for secret references as we already test them
	// in plugin_test.go.
}

func TestTranslateApisixConsumerV2(t *testing.T) {
	ac := &configv2.ApisixConsumer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "jack",
			Namespace: "qa",
		},
		Spec: configv2.ApisixConsumerSpec{
			AuthParameter: configv2.ApisixConsumerAuthParameter{
				BasicAuth: &configv2.ApisixConsumerBasicAuth{
					Value: &configv2.ApisixConsumerBasicAuthValue{
						Username: "jack",
						Password: "jacknice",
					},
				},
			},
		},
	}
	consumer, err := (&translator{}).TranslateApisixConsumerV2(ac)
	assert.Nil(t, err)
	assert.Len(t, consumer.Plugins, 1)
	cfg := consumer.Plugins["basic-auth"].(*apisixv1.BasicAuthConsumerConfig)
	assert.Equal(t, "jack", cfg.Username)
	assert.Equal(t, "jacknice", cfg.Password)

	ac = &configv2.ApisixConsumer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "jack",
			Namespace: "qa",
		},
		Spec: configv2.ApisixConsumerSpec{
			AuthParameter: configv2.ApisixConsumerAuthParameter{
				KeyAuth: &configv2.ApisixConsumerKeyAuth{
					Value: &configv2.ApisixConsumerKeyAuthValue{
						Key: "qwerty",
					},
				},
			},
		},
	}
	consumer, err = (&translator{}).TranslateApisixConsumerV2(ac)
	assert.Nil(t, err)
	assert.Len(t, consumer.Plugins, 1)
	cfg2 := consumer.Plugins["key-auth"].(*apisixv1.KeyAuthConsumerConfig)
	assert.Equal(t, "qwerty", cfg2.Key)

	ac = &configv2.ApisixConsumer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "jack",
			Namespace: "qa",
		},
		Spec: configv2.ApisixConsumerSpec{
			AuthParameter: configv2.ApisixConsumerAuthParameter{
				JwtAuth: &configv2.ApisixConsumerJwtAuth{
					Value: &configv2.ApisixConsumerJwtAuthValue{
						Key:          "foo",
						Secret:       "123",
						PublicKey:    "public",
						PrivateKey:   "private",
						Algorithm:    "HS256",
						Exp:          int64(1000),
						Base64Secret: true,
					},
				},
			},
		},
	}
	consumer, err = (&translator{}).TranslateApisixConsumerV2(ac)
	assert.Nil(t, err)
	assert.Len(t, consumer.Plugins, 1)
	cfg3 := consumer.Plugins["jwt-auth"].(*apisixv1.JwtAuthConsumerConfig)
	assert.Equal(t, "foo", cfg3.Key)
	assert.Equal(t, "123", cfg3.Secret)
	assert.Equal(t, "public", cfg3.PublicKey)
	assert.Equal(t, "private", cfg3.PrivateKey)
	assert.Equal(t, "HS256", cfg3.Algorithm)
	assert.Equal(t, int64(1000), cfg3.Exp)
	assert.Equal(t, true, cfg3.Base64Secret)

	ac = &configv2.ApisixConsumer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "jack",
			Namespace: "qa",
		},
		Spec: configv2.ApisixConsumerSpec{
			AuthParameter: configv2.ApisixConsumerAuthParameter{
				WolfRBAC: &configv2.ApisixConsumerWolfRBAC{
					Value: &configv2.ApisixConsumerWolfRBACValue{
						Server: "https://httpbin.org",
						Appid:  "test01",
					},
				},
			},
		},
	}
	consumer, err = (&translator{}).TranslateApisixConsumerV2(ac)
	assert.Nil(t, err)
	assert.Len(t, consumer.Plugins, 1)
	cfg4 := consumer.Plugins["wolf-rbac"].(*apisixv1.WolfRBACConsumerConfig)
	assert.Equal(t, "https://httpbin.org", cfg4.Server)
	assert.Equal(t, "test01", cfg4.Appid)

	ac = &configv2.ApisixConsumer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "jack",
			Namespace: "qa",
		},
		Spec: configv2.ApisixConsumerSpec{
			AuthParameter: configv2.ApisixConsumerAuthParameter{
				HMACAuth: &configv2.ApisixConsumerHMACAuth{
					Value: &configv2.ApisixConsumerHMACAuthValue{
						AccessKey: "foo",
						SecretKey: "bar",
					},
				},
			},
		},
	}
	consumer, err = (&translator{}).TranslateApisixConsumerV2(ac)
	assert.Nil(t, err)
	assert.Len(t, consumer.Plugins, 1)
	cfg5 := consumer.Plugins["hmac-auth"].(*apisixv1.HMACAuthConsumerConfig)
	assert.Equal(t, "foo", cfg5.AccessKey)
	assert.Equal(t, "bar", cfg5.SecretKey)

	ac = &configv2.ApisixConsumer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "jack",
			Namespace: "qa",
		},
		Spec: configv2.ApisixConsumerSpec{
			AuthParameter: configv2.ApisixConsumerAuthParameter{
				LDAPAuth: &configv2.ApisixConsumerLDAPAuth{
					Value: &configv2.ApisixConsumerLDAPAuthValue{
						UserDN: "cn=user01,ou=users,dc=example,dc=org",
					},
				},
			},
		},
	}
	consumer, err = (&translator{}).TranslateApisixConsumerV2(ac)
	assert.Nil(t, err)
	assert.Len(t, consumer.Plugins, 1)
	cfg6 := consumer.Plugins["ldap-auth"].(*apisixv1.LDAPAuthConsumerConfig)
	assert.Equal(t, "cn=user01,ou=users,dc=example,dc=org", cfg6.UserDN)

	// No test test cases for secret references as we already test them
	// in plugin_test.go.
}
