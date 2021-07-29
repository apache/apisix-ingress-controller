<!--
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
-->

## apisix-ingress-controller release process

1. Create release pull request with release notes.

   1. Compile release notes detailing features added since the last release and
      add release template file to `releases/` directory. The template is defined
      by containerd's release tool but refer to previous release files for style
      and format help. Name the file using the version.
      See [release-tool](https://github.com/containerd/release-tool)

      You can use the following command to generate content

      ```sh
      release-tool -l -d -n -t 1.0.0 releases/v1.0.0.toml
      ```

2. Vote for release

3. Create tag

4. Push tag and Github release

5. Promote on Slack, Twitter, mailing lists, etc
