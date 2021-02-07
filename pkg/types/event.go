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

package types

// EventType is the type of event.
type EventType int

const (
	// EventAdd means an add event.
	EventAdd = iota + 1
	// EventUpdate means an update event.
	EventUpdate
	// EventDelete means a delete event.
	EventDelete
)

func (ev EventType) String() string {
	switch ev {
	case EventAdd:
		return "add"
	case EventUpdate:
		return "update"
	case EventDelete:
		return "delete"
	default:
		return "unknown"
	}
}

// Event represents a typed event.
type Event struct {
	// Type is the type of event.
	Type EventType
	// Object is the event subject.
	Object interface{}
	// Tombstone is the final state before object was delete,
	// it's useful for DELETE event.
	Tombstone interface{}
}
