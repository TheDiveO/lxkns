// Copyright 2021 Harald Albrecht.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may not
// use this file except in compliance with the License. You may obtain a copy
// of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package types

import (
	"bytes"
	"encoding/json"
	"strconv"

	"github.com/thediveo/lxkns/model"
)

// ContainerModel wraps containers & co.
type ContainerModel struct {
	Containers       ContainerMap
	ContainerEngines EngineMap
	Groups           GroupMap
}

func NewContainerModel(containers []*model.Container) *ContainerModel {
	cm := &ContainerModel{}
	cm.Containers = NewContainerMap(cm, containers)
	cm.ContainerEngines = NewEngineMap(cm, containers)
	cm.Groups = NewGroupMap(cm, containers)
	return cm
}

// ----

type ContainerMap struct {
	Containers map[uint]*model.Container
	cm         *ContainerModel
}

func NewContainerMap(cosco *ContainerModel, containers []*model.Container) ContainerMap {
	m := ContainerMap{
		Containers: map[uint]*model.Container{},
		cm:         cosco,
	}
	for _, container := range containers {
		m.Containers[uint(container.PID)] = container
	}
	return m
}

func (m ContainerMap) ContainerByRefID(refid uint) *model.Container {
	container, ok := m.Containers[refid]
	if !ok {
		container = &model.Container{}
		m.Containers[refid] = container
	}
	return container
}

func (m *ContainerMap) MarshalJSON() ([]byte, error) {
	b := bytes.Buffer{}
	b.WriteRune('{')
	first := true
	for refid, container := range m.Containers {
		if first {
			first = false
		} else {
			b.WriteRune(',')
		}
		b.WriteRune('"')
		b.WriteString(strconv.FormatUint(uint64(refid), 10))
		b.WriteString(`":`)
		gids := make([]uint, len(container.Groups))
		for idx, group := range container.Groups {
			gids[idx] = m.cm.Groups.GroupRefID(group)
		}
		cntrjson, err := json.Marshal(&struct {
			Engine uint   `json:"engine"`
			Groups []uint `json:"groups"`
			*model.Container
		}{
			Engine:    m.cm.ContainerEngines.EngineRefID(container.Engine),
			Groups:    gids,
			Container: container,
		})
		if err != nil {
			return nil, err
		}
		b.Write(cntrjson)
	}
	b.WriteRune('}')
	return b.Bytes(), nil
}

// ----

type EngineMap struct {
	enginesByRefID map[uint]*model.ContainerEngine // associate (ref) IDs with the engines.
	engineRefIDs   map[*model.ContainerEngine]uint // map ref IDs to engines.
	cm             *ContainerModel
}

// NewEngineMap creates a new map for ContainerEngines, optionally building
// using a discovered list of containers (with their ContainerEngines).
func NewEngineMap(cosco *ContainerModel, containers []*model.Container) EngineMap {
	m := EngineMap{
		enginesByRefID: map[uint]*model.ContainerEngine{},
		engineRefIDs:   map[*model.ContainerEngine]uint{},
		cm:             cosco,
	}
	// If containers were discovered, then associate (ref) IDs with the engines
	// managing the containers.
	eid := uint(0)
	for _, container := range containers {
		if _, ok := m.engineRefIDs[container.Engine]; !ok {
			eid++
			m.engineRefIDs[container.Engine] = eid // associate a new ID with the engine
			m.enginesByRefID[eid] = container.Engine
		}
	}
	return m
}

// EngineByRefID returns the ContainerEngine associated with a (ref) ID,
// creating a new zero ContainerEngine if necessary.
func (m EngineMap) EngineByRefID(refid uint) *model.ContainerEngine {
	engine, ok := m.enginesByRefID[refid]
	if !ok {
		engine = &model.ContainerEngine{}
		m.enginesByRefID[refid] = engine
	}
	return engine
}

// EngineRefID returns the (ref) ID associated with a particular
// ContainerEngine.
func (m EngineMap) EngineRefID(engine *model.ContainerEngine) uint {
	return m.engineRefIDs[engine]
}

func (l *EngineMap) MarshalJSON() ([]byte, error) {
	b := bytes.Buffer{}
	b.WriteRune('{')
	first := true
	for refid, engine := range l.enginesByRefID {
		if first {
			first = false
		} else {
			b.WriteRune(',')
		}
		b.WriteRune('"')
		b.WriteString(strconv.FormatUint(uint64(refid), 10))
		b.WriteString(`":`)
		cids := make([]uint, len(engine.Containers))
		for idx, container := range engine.Containers {
			cids[idx] = uint(container.PID)
		}
		engjson, err := json.Marshal(&struct {
			CIDs []uint `json:"containers"`
			*model.ContainerEngine
		}{
			CIDs:            cids,
			ContainerEngine: (*model.ContainerEngine)(engine),
		})
		if err != nil {
			return nil, err
		}
		b.Write(engjson)
	}
	b.WriteRune('}')
	return b.Bytes(), nil
}

// -----

type GroupMap struct {
	groupsByRefID map[uint]*model.Group // associate (ref) IDs with the groups.
	groupRefIDs   map[*model.Group]uint // map ref IDs to groups.
	cm            *ContainerModel
}

// NewEngineMap creates a new map for ContainerEngines, optionally building
// using a discovered list of containers (with their ContainerEngines).
func NewGroupMap(cosco *ContainerModel, containers []*model.Container) GroupMap {
	m := GroupMap{
		groupsByRefID: map[uint]*model.Group{},
		groupRefIDs:   map[*model.Group]uint{},
		cm:            cosco,
	}
	// If containers were discovered, then associate (ref) IDs with the groups
	// grouping these containers.
	gid := uint(0)
	for _, container := range containers {
		for _, group := range container.Groups {
			if _, ok := m.groupRefIDs[group]; !ok {
				gid++
				m.groupRefIDs[group] = gid // associate a new ID with the group
				m.groupsByRefID[gid] = group
			}
		}
	}
	return m
}

// GroupByRefID returns the ContainerEngine associated with a (ref) ID, creating
// a new zero Group if necessary.
func (m GroupMap) GroupByRefID(refid uint) *model.Group {
	group, ok := m.groupsByRefID[refid]
	if !ok {
		group = &model.Group{}
		m.groupsByRefID[refid] = group
	}
	return group
}

// GroupRefID returns the (ref) ID associated with a particular Group.
func (m GroupMap) GroupRefID(group *model.Group) uint {
	return m.groupRefIDs[group]
}

func (l *GroupMap) MarshalJSON() ([]byte, error) {
	b := bytes.Buffer{}
	b.WriteRune('{')
	first := true
	for refid, group := range l.groupsByRefID {
		if first {
			first = false
		} else {
			b.WriteRune(',')
		}
		b.WriteRune('"')
		b.WriteString(strconv.FormatUint(uint64(refid), 10))
		b.WriteString(`":`)
		cids := make([]uint, len(group.Containers))
		for idx, container := range group.Containers {
			cids[idx] = uint(container.PID)
		}
		engjson, err := json.Marshal(&struct {
			CIDs []uint `json:"containers"`
			*model.Group
		}{
			CIDs:  cids,
			Group: (*model.Group)(group),
		})
		if err != nil {
			return nil, err
		}
		b.Write(engjson)
	}
	b.WriteRune('}')
	return b.Bytes(), nil
}
