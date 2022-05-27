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

docker-compose -f 'docker-compose.yaml'  -p 'wolf-rbac' down >> info.log 2>&1

docker stop wolf-agent-demo restful-demo wolf-agent-or wolf-server wolf-database >> info.log 2>&1
docker rm wolf-agent-demo restful-demo wolf-agent-or wolf-server wolf-database >> info.log 2>&1

rm -rf db-psql.sql

wget https://raw.githubusercontent.com/iGeeky/wolf/f6ddeb75a37bff90406f0f0a2b7ae5d16f6f3bd4/server/script/db-psql.sql >> info.log 2>&1

# start database
docker-compose up -d database >> info.log 2>&1

sleep 2

# start wolf-server
docker-compose up -d server restful-demo agent-or agent-demo  >> info.log 2>&1

sleep 10

docker inspect -f '{{range .NetworkSettings.Networks}}Gateway:{{.Gateway}} IPAdress:{{.IPAddress}}{{end}}' wolf-server >> info.log 2>&1

netstat -atnp | grep ":12180" >> info.log 2>&1

cat info.log && rm info.log