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

	v1 "github.com/api7/ingress-controller/pkg/types/apisix/v1"
)

func TestMemDBCacheRoute(t *testing.T) {
	c, err := NewMemDBCache()
	assert.Nil(t, err, "NewMemDBCache")

	name := "abc"
	sid := "1"
	r1 := &v1.Route{
		FullName:  &name,
		Name:      &name,
		ServiceId: &sid,
	}
	assert.Nil(t, c.InsertRoute(r1), "inserting route 1")

	r, err := c.GetRoute("abc")
	assert.Equal(t, r1, r)

	name2 := "def"
	sid2 := "2"
	r2 := &v1.Route{
		FullName:  &name2,
		Name:      &name2,
		ServiceId: &sid2,
	}
	name3 := "ghi"
	sid3 := "3"
	r3 := &v1.Route{
		FullName:  &name3,
		Name:      &name3,
		ServiceId: &sid3,
	}
	assert.Nil(t, c.InsertRoute(r2), "inserting route r2")
	assert.Nil(t, c.InsertRoute(r3), "inserting route r3")

	r, err = c.GetRoute("ghi")
	assert.Equal(t, r3, r)

	assert.Nil(t, c.DeleteRoute(r3), "delete route r3")

	routes, err := c.ListRoutes()
	assert.Nil(t, err, "listing routes")

	if *routes[0].FullName > *routes[1].FullName {
		routes[0], routes[1] = routes[1], routes[0]
	}
	assert.Equal(t, routes[0], r1)
	assert.Equal(t, routes[1], r2)

	name4 := "name4"
	sid4 := "4"
	r4 := &v1.Route{
		FullName:  &name4,
		Name:      &name4,
		ServiceId: &sid4,
	}
	assert.Error(t, ErrNotFound, c.DeleteRoute(r4))
}

func TestMemDBCacheService(t *testing.T) {
	c, err := NewMemDBCache()
	assert.Nil(t, err, "NewMemDBCache")

	name := "abc"
	uid := "1"
	s1 := &v1.Service{
		FullName:   &name,
		Name:       &name,
		UpstreamId: &uid,
	}
	assert.Nil(t, c.InsertService(s1), "inserting service 1")

	s, err := c.GetService("abc")
	assert.Equal(t, s1, s)

	name2 := "def"
	uid2 := "2"
	s2 := &v1.Service{
		FullName:   &name2,
		Name:       &name2,
		UpstreamId: &uid2,
	}
	name3 := "ghi"
	uid3 := "3"
	s3 := &v1.Service{
		FullName:   &name3,
		Name:       &name3,
		UpstreamId: &uid3,
	}
	assert.Nil(t, c.InsertService(s2), "inserting service 2")
	assert.Nil(t, c.InsertService(s3), "inserting service 3")

	s, err = c.GetService("ghi")
	assert.Equal(t, s3, s)

	assert.Nil(t, c.DeleteService(s3), "delete service 3")

	services, err := c.ListServices()
	assert.Nil(t, err, "listing services")

	if *services[0].FullName > *services[1].FullName {
		services[0], services[1] = services[1], services[0]
	}
	assert.Equal(t, services[0], s1)
	assert.Equal(t, services[1], s2)

	name4 := "name4"
	uid4 := "4"
	s4 := &v1.Service{
		FullName:   &name4,
		Name:       &name4,
		UpstreamId: &uid4,
	}
	assert.Error(t, ErrNotFound, c.DeleteService(s4))
}

func TestMemDBCacheSSL(t *testing.T) {
	c, err := NewMemDBCache()
	assert.Nil(t, err, "NewMemDBCache")

	id := "abc"
	s1 := &v1.Ssl{
		ID: &id,
	}
	assert.Nil(t, c.InsertSSL(s1), "inserting ssl 1")

	s, err := c.GetSSL("abc")
	assert.Equal(t, s1, s)

	id2 := "def"
	s2 := &v1.Ssl{
		ID: &id2,
	}
	id3 := "ghi"
	s3 := &v1.Ssl{
		ID: &id3,
	}
	assert.Nil(t, c.InsertSSL(s2), "inserting ssl 2")
	assert.Nil(t, c.InsertSSL(s3), "inserting ssl 3")

	s, err = c.GetSSL("ghi")
	assert.Equal(t, s3, s)

	assert.Nil(t, c.DeleteSSL(s3), "delete ssl 3")

	ssl, err := c.ListSSL()
	assert.Nil(t, err, "listing ssl")

	if *ssl[0].ID > *ssl[1].ID {
		ssl[0], ssl[1] = ssl[1], ssl[0]
	}
	assert.Equal(t, ssl[0], s1)
	assert.Equal(t, ssl[1], s2)

	id4 := "id4"
	s4 := &v1.Ssl{
		ID: &id4,
	}
	assert.Error(t, ErrNotFound, c.DeleteSSL(s4))
}

func TestMemDBCacheUpstream(t *testing.T) {
	c, err := NewMemDBCache()
	assert.Nil(t, err, "NewMemDBCache")

	name := "abc"
	u1 := &v1.Upstream{
		FullName: &name,
		Name:     &name,
	}
	assert.Nil(t, c.InsertUpstream(u1), "inserting upstream 1")

	u, err := c.GetUpstream("abc")
	assert.Equal(t, u1, u)

	name2 := "def"
	u2 := &v1.Upstream{
		FullName: &name2,
		Name:     &name2,
	}
	name3 := "ghi"
	u3 := &v1.Upstream{
		FullName: &name3,
		Name:     &name3,
	}
	assert.Nil(t, c.InsertUpstream(u2), "inserting upstream 2")
	assert.Nil(t, c.InsertUpstream(u3), "inserting upstream 3")

	u, err = c.GetUpstream("ghi")
	assert.Equal(t, u3, u)

	assert.Nil(t, c.DeleteUpstream(u3), "delete upstream 3")

	upstreams, err := c.ListUpstreams()
	assert.Nil(t, err, "listing upstreams")

	if *upstreams[0].FullName > *upstreams[1].FullName {
		upstreams[0], upstreams[1] = upstreams[1], upstreams[0]
	}
	assert.Equal(t, upstreams[0], u1)
	assert.Equal(t, upstreams[1], u2)

	name4 := "name4"
	u4 := &v1.Upstream{
		FullName: &name4,
		Name:     &name4,
	}
	assert.Error(t, ErrNotFound, c.DeleteUpstream(u4))
}

func TestMemDBCacheReference(t *testing.T) {
	rname := "route"
	sid := "service"
	r := &v1.Route{
		FullName:  &rname,
		Name:      &rname,
		ServiceId: &sid,
	}
	sname := "service"
	uid := "upstream"
	s := &v1.Service{
		FullName:   &sname,
		Name:       &sname,
		UpstreamId: &uid,
	}
	uname := "upstream"
	u := &v1.Upstream{
		FullName: &uname,
		Name:     &uname,
	}

	db, err := NewMemDBCache()
	assert.Nil(t, err, "NewMemDBCache")
	assert.Nil(t, db.InsertRoute(r))
	assert.Nil(t, db.InsertService(s))
	assert.Nil(t, db.InsertUpstream(u))

	assert.Error(t, ErrStillInUse, db.DeleteService(s))
	assert.Error(t, ErrStillInUse, db.DeleteUpstream(u))
	assert.Nil(t, db.DeleteRoute(r))
	assert.Nil(t, db.DeleteService(s))
	assert.Nil(t, db.DeleteUpstream(u))
}
