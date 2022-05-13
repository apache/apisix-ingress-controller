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

cd testdata/wolf-rbac/

wget https://raw.githubusercontent.com/iGeeky/wolf/master/server/script/db-psql.sql

# start database
docker-compose up -d database

# start wolf-server
docker-compose up -d server restful-demo agent-or agent-demo

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
