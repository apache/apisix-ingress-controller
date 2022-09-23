#!/bin/sh
#
# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

while [[ $# -gt 0 ]]; do
    case ${1} in
        --service)
            service="$2"
            shift
            ;;
        --secret)
            secret="$2"
            shift
            ;;
        --namespace)
            namespace="$2"
            shift
            ;;
        *)
            usage
            ;;
    esac
    shift
done

tmpdir=$(mktemp -d)
cd ${tmpdir}

go install github.com/cloudflare/cfssl/cmd/cfssl@latest
go install github.com/cloudflare/cfssl/cmd/cfssljson@latest
GOBIN=$(go env GOPATH)/bin
wget https://github.com/stedolan/jq/releases/download/jq-1.6/jq-linux64 -O jq 2>/dev/null && chmod +x jq

PATH=$PATH:$GOBIN:$tmpdir

service=${service:-webhook}
namespace=${namespace:-ingress-apisix}
secret=${secret:-webhook-certs}

svc_dns=${service}.${namespace}.svc

cat <<EOF | cfssl genkey - | cfssljson -bare server
{
  "hosts": [
    "${svc_dns}",
    "192.0.2.24",
    "10.0.34.2"
  ],
  "CN": "${svc_dns}",
  "key": {
    "algo": "ecdsa",
    "size": 256
  }
}
EOF

csrName=${service}.${namespace}

kubectl delete csr ${csrName} 2>/dev/null | true

cat <<EOF | kubectl apply -f -
apiVersion: certificates.k8s.io/v1
kind: CertificateSigningRequest
metadata:
  name: ${csrName}
spec:
  request: $(cat server.csr | base64 | tr -d '\n')
  signerName: example.com/serving
  usages:
  - digital signature
  - key encipherment
  - server auth
EOF


kubectl certificate approve ${csrName}

cat <<EOF | cfssl gencert -initca - | cfssljson -bare ca
{
  "CN": "My Example Signer",
  "key": {
    "algo": "rsa",
    "size": 2048
  }
}
EOF


echo '{
    "signing": {
        "default": {
            "usages": [
                "digital signature",
                "key encipherment",
                "server auth"
            ],
            "expiry": "876000h",
            "ca_constraint": {
                "is_ca": false
            }
        }
    }
}' > server-signing-config.json


kubectl get csr ${csrName} -o jsonpath='{.spec.request}' | \
  base64 --decode | \
  cfssl sign -ca ca.pem -ca-key ca-key.pem -config server-signing-config.json - | \
  cfssljson -bare ca-signed-server




kubectl get csr ${csrName} -o json | \
  jq '.status.certificate = "'$(base64 ca-signed-server.pem | tr -d '\n')'"' | \
  kubectl replace --raw /apis/certificates.k8s.io/v1/certificatesigningrequests/${csrName}/status -f -

sleep 2

kubectl get csr ${csrName} -o jsonpath='{.status.certificate}' \
    | base64 --decode > server.crt


kubectl -n ${namespace} delete secret ${secret} -n ${namespace} 2>/dev/null | true
kubectl -n ${namespace} create secret generic ${secret} --from-file=key.pem=server-key.pem --from-file=cert.pem=server.crt  -n ${namespace}

cd -
rm -rf ${tmpdir}