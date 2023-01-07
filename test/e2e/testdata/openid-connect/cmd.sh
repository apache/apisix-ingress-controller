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

cd test/e2e/testdata/openid-connect/ || exit

OPTION=$1

if  [ "$OPTION" = "ip" ]; then
    echo $(docker inspect -f '{{range .NetworkSettings.Networks}}{{.Gateway}}{{end}}' keycloak)
elif [ "$OPTION" = "start" ]; then
    docker-compose -f 'docker-compose.yaml'  -p 'openid-connect' up -d

    sleep 6

    ACCESS_TOKEN=$(curl -X POST -H "Content-Type: application/x-www-form-urlencoded" -d "username=admin" -d "password=password" -d "grant_type=password" -d 'client_id=admin-cli' "http://127.0.0.1:8222/auth/realms/master/protocol/openid-connect/token"|jq -r '.access_token')

    # update access token lifespan
    curl --location --request PUT 'http://127.0.0.1:8222/auth/admin/realms/master' \
    --header "Authorization: Bearer $ACCESS_TOKEN" \
    --header 'Content-Type: application/json' \
    --data '{
       "accessTokenLifespan": 604800
    }'

    ACCESS_TOKEN=$(curl -X POST -H "Content-Type: application/x-www-form-urlencoded" -d "username=admin" -d "password=password" -d "grant_type=password" -d 'client_id=admin-cli' "http://127.0.0.1:8222/auth/realms/master/protocol/openid-connect/token"|jq -r '.access_token')

    # create realm apisix
    curl --location --request POST 'http://127.0.0.1:8222/auth/admin/realms' \
    --header "Authorization: Bearer $ACCESS_TOKEN" \
    --header 'Content-Type: application/json' \
    --data '{
       "realm":"apisix-realm",
       "notBefore":0,
       "enabled":true,
       "sslRequired":"none",
       "bruteForceProtected":true,
       "failureFactor":10,
       "eventsEnabled":false,
       "accessTokenLifespan": 604800
    }'

    # create client apisix
    curl --location --request POST 'http://127.0.0.1:8222/auth/admin/realms/apisix-realm/clients' \
    --header "Authorization: Bearer $ACCESS_TOKEN" \
    --header 'Content-Type: application/json' \
    --data '{
       "clientId":"apisix",
       "rootUrl":"https://apisix.com/apisix/",
       "adminUrl":"https://apisix.com/apisix/"
    }'

elif [ "$OPTION" = "stop" ]; then
    docker-compose -f 'docker-compose.yaml'  -p 'openid-connect' down
elif [ "$OPTION" = "secret" ]; then
    IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.Gateway}}{{end}}' keycloak)
    ACCESS_TOKEN=$(curl -X POST -H "Content-Type: application/x-www-form-urlencoded" -d "username=admin" -d "password=password" -d "grant_type=password" -d 'client_id=admin-cli' "http://$IP:8222/auth/realms/master/protocol/openid-connect/token"|jq -r '.access_token')
    CLIENT_ID=$(curl --location --request GET "http://$IP:8222/auth/admin/realms/apisix-realm/clients?clientId=apisix" --header 'clientId: example' --header "Authorization: Bearer $ACCESS_TOKEN"|jq -r '.[0]|.id')
    SECRET=$(curl --location --request GET "http://$IP:8222/auth/admin/realms/apisix-realm/clients/$CLIENT_ID/client-secret" --header "Authorization: Bearer $ACCESS_TOKEN"|jq -r '.value')
    echo "$SECRET"
elif [ "$OPTION" = "access_token" ]; then
    IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.Gateway}}{{end}}' keycloak)
    ACCESS_TOKEN=$(curl -X POST -H "Content-Type: application/x-www-form-urlencoded" -d "username=admin" -d "password=password" -d "grant_type=password" -d 'client_id=admin-cli' "http://$IP:8222/auth/realms/master/protocol/openid-connect/token"|jq -r '.access_token')
    echo "$ACCESS_TOKEN"
else
    echo "argument is one of [ip, start, stop, secret, access_token]"
fi
