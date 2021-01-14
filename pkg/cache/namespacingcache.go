// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cache

// NamespacingCache defines the necessary behaviors that the cache object should have.
// Note this interface is for APISIX, not for generic purpose, the namespace concept here
// is used to distinguish the different resource types in APISIX. So it should only support
// standard APISIX resources, i.e. Route, Upstream, Service and SSL.
type NamespacingCache interface {
	// Insert adds or updates the object to the specified namespace.
	Insert(string, interface{}) error
	// Get finds the object in the specified namespace according to
	// the key.
	Get(string, string) (interface{}, error)
	// List lists all objects in the specified namespace.
	List(string) ([]interface{}, error)
	// Delete deletes the specified object in the specified namesapce.
	Delete(string, interface{}) error
}
