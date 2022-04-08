#!/bin/bash

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

# NOTE:
# This script uses `GNU expr index`, but MacOS uses `BSD expr` which does not have the `index` command.
# You could install `GNU expr` by `brew install coreutils` and replace all `expr` with `gexpr` in the script if you are using MacOS.

# Search for the pattern in all files recursively in `test/e2e`, showing line numbers of matches, ignoring binary files.
# The results are separated by '\n' and stored in the array lines.
# Each line in lines looks like: `test/e2e/suite-endpoints/endpoints.go:28:var _ = ginkgo.Describe("suite-endpoints: endpoints", func() {`
IFS=$'\n' lines=($(grep --recursive --line-number --binary-files=without-match "ginkgo.Describe(" test/e2e))

# How many lines do not have the "suite-<suite-name>" prefix.
err=0

for (( i=0;i<${#lines[@]};i++)); do
  # Find the second colon in the line to split the line into two parts: left_str and right_str.
  pos1=$(expr index "${lines[i]}" ":")
  temp_str=${lines[i]:$pos1}
  pos2=$(expr index "$temp_str" ":")

  # left_str looks like: `test/e2e/suite-endpoints/endpoints.go:28`
  left_str=$(echo "${lines[i]}" | cut -c1-$(expr $pos1 + $pos2 - 1))
  # right_str looks like: `var _ = ginkgo.Describe("suite-endpoints: endpoints", func() {`
  right_str=${lines[i]:$pos1+$pos2}

  l_name=$(echo "$left_str" | grep --extended-regexp -o 'suite-\w+')
  r_name=$(echo "$right_str" | grep --extended-regexp -o 'suite-\w+')

  if [ -n "$l_name" ] && [ "$l_name" != "$r_name" ]; then
    echo "[ERROR]$left_str: $l_name is required"
    err+=1
  fi
done;

if [ $err -gt 0 ]; then
  echo "-------------------------------------------------------------------------------------"
  echo 'The prefix "suite-<suite-name>" is required, see test/e2e/README.md for more details.'
  echo "-------------------------------------------------------------------------------------"
  exit 1
fi
