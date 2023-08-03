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

cd test/e2e/testdata/ldap/

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
    echo -n `docker inspect -f '{{range .NetworkSettings.Networks}}{{.Gateway}}{{end}}' openldap`
elif [ $OPTION = "start" ]; then
    $COMPOSE_CMD -f 'docker-compose.yaml'  -p 'openldap' down

    # start openldap
    $COMPOSE_CMD -f 'docker-compose.yaml'  -p 'openldap' up -d

elif [ $OPTION = "stop" ]; then
    $COMPOSE_CMD -f  'docker-compose.yaml'  -p 'openldap' down
else
    echo "argument is one of [ip, start, stop]"
fi
