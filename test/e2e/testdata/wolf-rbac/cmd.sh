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

cd test/e2e/testdata/wolf-rbac/

OPTION=$1
COMPOSE_CMD=""

if command -v "docker-compose" > /dev/null 2>&1; then
    COMPOSE_CMD="docker-compose"
elif command -v "docker" > /dev/null 2>&1; then
    COMPOSE_CMD="docker compose"
else
    echo "docker-compose or docker compose not found"
    exit 1
fi

if  [ $OPTION = "ip" ]; then
    echo -n `docker inspect -f '{{range .NetworkSettings.Networks}}{{.Gateway}}{{end}}' wolf-server`
elif [ $OPTION = "start" ]; then
    $COMPOSE_CMD -f 'docker-compose.yaml'  -p 'wolf-rbac' down
    rm -rf db-psql.sql

    wget https://raw.githubusercontent.com/iGeeky/wolf/f6ddeb75a37bff90406f0f0a2b7ae5d16f6f3bd4/server/script/db-psql.sql

    # start database
    $COMPOSE_CMD up -d database

    # start wolf-server
    $COMPOSE_CMD up -d server restful-demo agent-or agent-demo

    sleep 6

    WOLF_TOKEN=`curl http://127.0.0.1:12180/wolf/user/login  -H "Content-Type: application/json"  -d '{ "username": "root", "password": "wolf-123456"}' -s | grep token| tr -d ':",' | awk '{print $2}'`

    curl http://127.0.0.1:12180/wolf/application \
    -H "Content-Type: application/json" \
    -H "x-rbac-token: $WOLF_TOKEN" \
    -d '{
        "id": "test-app", 
        "name": "application for test"
    }'

    curl http://127.0.0.1:12180/wolf/resource \
    -H "Content-Type: application/json" \
    -H "x-rbac-token: $WOLF_TOKEN" \
    -d '{
        "appID": "test-app",
        "matchType": "prefix",
        "name": "/",
        "action": "GET",
        "permID": "ALLOW_ALL"
    }'

    curl http://127.0.0.1:12180/wolf/user \
    -H "Content-Type: application/json" \
    -H "x-rbac-token: $WOLF_TOKEN" \
    -d '{
        "username": "test",
        "nickname": "test",
        "password": "test-123456",
        "appIDs": ["test-app"]
    }'
elif [ $OPTION = "stop" ]; then
    $COMPOSE_CMD -f 'docker-compose.yaml'  -p 'wolf-rbac' down
    rm -rf db-psql.sql
else
    echo "argument is one of [ip, start, stop]"
fi