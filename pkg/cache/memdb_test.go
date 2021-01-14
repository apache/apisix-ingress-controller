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

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type miniRoute struct {
	FullName  string
	Name      string
	ServiceId string
}

func TestMemDBCacheBadNamespace(t *testing.T) {
	c, err := NewMemDBCache()
	assert.Nil(t, err, "NewMemDBCache")
	r1 := &miniRoute{}
	err = c.Insert("aaaa", r1)
	assert.Equal(t, "invalid namespace", err.Error())
}

func TestMemDBCache(t *testing.T) {
	c, err := NewMemDBCache()
	assert.Nil(t, err, "NewMemDBCache")

	r1 := &miniRoute{
		FullName:  "abc",
		Name:      "abc",
		ServiceId: "1",
	}
	assert.Nil(t, c.Insert("route", r1), "inserting route 1")

	raw, err := c.Get("route", "abc")
	assert.Nil(t, err, "getting route r1")
	r := raw.(*miniRoute)
	assert.Equal(t, r1, r)

	r2 := &miniRoute{
		FullName:  "def",
		Name:      "def",
		ServiceId: "2",
	}
	r3 := &miniRoute{
		FullName:  "ghi",
		Name:      "ghi",
		ServiceId: "3",
	}
	assert.Nil(t, c.Insert("route", r2), "inserting route r2")
	assert.Nil(t, c.Insert("route", r3), "inserting route r3")

	raw, err = c.Get("route", "ghi")
	assert.Nil(t, err, "getting route r3")
	r = raw.(*miniRoute)
	assert.Equal(t, r3, r)

	assert.Nil(t, c.Delete("route", r3), "delete route r3")

	objs, err := c.List("route")
	assert.Nil(t, err, "listing routes")
	assert.Len(t, objs, 2)

	routes := []*miniRoute{
		objs[0].(*miniRoute),
		objs[1].(*miniRoute),
	}
	if routes[0].FullName > routes[1].FullName {
		routes[0], routes[1] = routes[1], routes[0]
	}
	assert.Equal(t, routes[0], r1)
	assert.Equal(t, routes[1], r2)
}
