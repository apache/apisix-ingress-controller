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

docker-compose -f 'docker-compose.yaml'  -p 'wolf-rbac' down

rm -rf db-psql.sql

wget https://raw.githubusercontent.com/iGeeky/wolf/f6ddeb75a37bff90406f0f0a2b7ae5d16f6f3bd4/server/script/db-psql.sql

# start database
docker-compose up -d database >> info.log 2>&1

# start wolf-server
docker-compose up -d server restful-demo agent-or agent-demo  >> info.log 2>&1

cat info.log && rm info.log

sleep 6