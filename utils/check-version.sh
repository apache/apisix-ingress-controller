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

ver=$1

red='\e[0;41m'
RED='\e[1;31m'
green='\e[0;32m'
GREEN='\e[1;32m'
NC='\e[0m'

# Makefile $ver
matched=`grep "VERSION ?= [0-9][0-9.]*" -r Makefile`
expected=`grep "VERSION ?= $ver" -r Makefile`

if [ "$matched" = "$expected" ]; then
    echo -e "${green}passed: version $ver ${NC}"
else
    echo -e "${RED}failed: version $ver ${NC}" 1>&2
    echo
    echo "-----maybe wrong version-----"
    echo "$matched"
    exit 1
fi
