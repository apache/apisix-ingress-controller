// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package types

import (
	"fmt"
	"strings"

	k8stypes "k8s.io/apimachinery/pkg/types"
)

type NamespacedNameKind struct {
	Namespace string
	Name      string
	Kind      string
}

func (n NamespacedNameKind) NamespacedName() k8stypes.NamespacedName {
	return k8stypes.NamespacedName{
		Namespace: n.Namespace,
		Name:      n.Name,
	}
}

func (n NamespacedNameKind) String() string {
	return n.Kind + "/" + n.Namespace + "/" + n.Name
}

func (n NamespacedNameKind) MarshalText() ([]byte, error) {
	return []byte(n.String()), nil
}

func (n *NamespacedNameKind) UnmarshalText(text []byte) error {
	return n.FromString(string(text))
}

func (n *NamespacedNameKind) FromString(s string) error {
	parts := strings.Split(s, "/")
	if len(parts) != 3 {
		return fmt.Errorf("invalid format for NamespacedNameKind: %q, expected Kind/Namespace/Name", s)
	}

	n.Kind = parts[0]
	n.Namespace = parts[1]
	n.Name = parts[2]
	return nil
}
