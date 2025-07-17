// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package framework

import (
	"cmp"
	_ "embed"
	"os"
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

var (
	ProviderType       = cmp.Or(os.Getenv("PROVIDER_TYPE"), "apisix")
	ProviderSyncPeriod = cmp.Or(os.Getenv("PROVIDER_SYNC_PERIOD"), "200ms")
)

var (
	//go:embed manifests/apisix.yaml
	apisixStandaloneTemplate string
	APISIXStandaloneTpl      *template.Template

	//go:embed manifests/etcd.yaml
	EtcdSpec string
)

var (
	//go:embed manifests/cert.pem
	TestServerCert string
	//go:embed manifests/key.pem
	TestServerKey string
)

const (
	TestCert = `-----BEGIN CERTIFICATE-----
MIIC1TCCAb2gAwIBAgIJANm/NDY0xwZUMA0GCSqGSIb3DQEBBQUAMBoxGDAWBgNV
BAMTD3d3dy5leGFtcGxlLmNvbTAeFw0yMTEyMjcwNzI0MTNaFw0zMTEyMjUwNzI0
MTNaMBoxGDAWBgNVBAMTD3d3dy5leGFtcGxlLmNvbTCCASIwDQYJKoZIhvcNAQEB
BQADggEPADCCAQoCggEBAPOeWroWLvnbzmRtvYsyRkFVePoMY3LvZqNaxCQpZHD6
ra/fRDTem01YvJjm5qUwrn9YXKBUgcoTfA3vHGYFHE4lifdfCbxlb0otMCbEdEsX
P8kOMszB5SlxIPiCLVhc1LOKmHDzzw7axrRStbgN/RJUQ9Fp1QXVAnvEMWcLNopD
E7I148dkpHrxmjW8vuB7apWhcVW+QiOYn4rGyqoilhrL4nRCOJiCVqESMgPcu5dO
Dxf6KcAVd/IMMFTQ/X4+e2dUJpYyhCe8ApnCqrumjfXKqIEfyyTCavKeQEfvPgK4
PhP2BFpWrxRWkn4VVTxIxS0/EVJaAaC/4gmVMeYg+wUCAwEAAaMeMBwwGgYDVR0R
BBMwEYIPd3d3LmV4YW1wbGUuY29tMA0GCSqGSIb3DQEBBQUAA4IBAQAKiJaa1FNC
p9NwoJvGyhK5UO2Tci3H63xZs2tFj5UZGxAIqJSxVo80ExhUXuDAM3evryM193uz
uNxbB/oIWEMNLBnacXQi8Evob14gkIwRmQ/iACSIGTupazBLwiZM6infPE2/OoYR
YihMgeWtW9U4XOkRhm013GgueeWP8v1jtyB2p3hoLK5UcLOAhkAOaJZXLDW0rznx
jkNC6NcjYvMHkm3bZYqGsRmZfNGvm5rM8s9c3n4MPgWlllt6RuMSimzIRQSKu2E4
oGKUqgeOOf5BXunHEgkyOTYittlg6MRwBET6ymHjYvwz87Loot7ji2IomyP08jdS
tYKdaDOJg+su
-----END CERTIFICATE-----`

	TestKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEA855auhYu+dvOZG29izJGQVV4+gxjcu9mo1rEJClkcPqtr99E
NN6bTVi8mObmpTCuf1hcoFSByhN8De8cZgUcTiWJ918JvGVvSi0wJsR0Sxc/yQ4y
zMHlKXEg+IItWFzUs4qYcPPPDtrGtFK1uA39ElRD0WnVBdUCe8QxZws2ikMTsjXj
x2SkevGaNby+4HtqlaFxVb5CI5ifisbKqiKWGsvidEI4mIJWoRIyA9y7l04PF/op
wBV38gwwVND9fj57Z1QmljKEJ7wCmcKqu6aN9cqogR/LJMJq8p5AR+8+Arg+E/YE
WlavFFaSfhVVPEjFLT8RUloBoL/iCZUx5iD7BQIDAQABAoIBAGVBMQZdCANTh5IY
RoqfR7IJ+3E6Su9Pb4J/zDwXdCa9GgmaK3gp+bSJKEII3l5UQIKvUDhXR2ac+Je2
BUCl6SDV22UUfDBwnHPhGj1Ss98t95XyL80I3d1+pqyDNqOeWc2R0lBIFYxgA+yY
3+xy6/d9TH6ylRaKdTDJ15qzf2SxMtR/SiXyILWU7xWiYxINoHh2IVDte/KlNa0q
iCbIiyX1xdYmcD0rCEVxrWlo1XNjmyO/MPTBhJf/DyZhQNHBDJa2fWzbPOr1I+Nb
vh0GiJVwhkENtucnjmt+jCLqTkTNAAv2mJ1DxbY/DcM+TgTxHmAlTpBM0bh3WsS9
De8hefUCgYEA/b9LP0fVXTv1K/whKcgi0AW1GCcUrdWVrdN3/K+yPnEZBtl8VY5t
SvQkJPkQsVFJWdZUdRDpHhqYFa0I6zIiNF7DbIxF+Ag+N7uiZ6xzP0L9k1wmgKR8
PT47fJVuHECxgxexz9FGQwXH7eroJjLPEoxD2Z56COVIJOlYO6e9sXMCgYEA9cgK
WxE2NsYIjrgOqs93GKYY+TmmoSHWiy1bl7p3sUolobPThSd31hdk6ZdMlPPbpr3+
MYgZoFLud+3l+/6+tttGNNVkB6lkVXzd2WWG6xOrErRwYIz57yiWKGLeWg17jXXf
zqjFNTLpd8U9lM8Lf/XNyfs2tU5oxkUD6teCo6cCgYAtwdMl5CQ7ndZGSj8Is8hj
TsQrSNDX0A4fvGSEsoIn9GkY7RsYqohW3dOuvyMddpUNmDK+sX/4J7+JGRzknLPC
UdxXtKvhYEsn7bQJkfVuUPw9GH7w77hfqts7Sg8DFT9tblZoLUrIR0CYTKX0TXE9
3QFXOtayx/XMgi+hAkyYtQKBgQCtgKGO1/+levbfiR8RhZNVWyuWBBSU+wYxCbv2
yDNmfClElWVkQhBemfUq0RvGqr8MXmLrJGCyxNiC4PXRhmurOe+9rEYJApNJpfQW
W416tU+2zJnoDp0BL22Q5PqCJ7JokiWEBa/xdhdJ7XsjaWV811CGnUhphQiBroat
aaVXUQKBgDXHbRmEBo/1fB4Gn7i2bjYOl1Z1e3klNvbdMT/ClNSFy8VsU3HP5XoL
jnzTc80ABlT1PQgrnQxhPpL3wbkSyv0lux5mcM0U89KxpR/SLlvVFAag6UWODt53
hhdq+X/vrgK+uicSx8Q1zL2iCLdfsZ0fPryMdTZrN3ytEBEWMPeX
-----END RSA PRIVATE KEY-----`
)

func init() {
	tpl, err := template.New("apisix-standalone").Funcs(sprig.TxtFuncMap()).Parse(apisixStandaloneTemplate)
	if err != nil {
		panic(err)
	}
	APISIXStandaloneTpl = tpl
}
