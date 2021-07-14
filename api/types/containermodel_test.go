// Copyright 2020 Harald Albrecht.
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
	"encoding/json"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/onsi/gomega/gstruct"
	"github.com/thediveo/lxkns/model"
)

var (
	ce1 = model.ContainerEngine{
		ID:   "ce1",
		Type: "typeA",
		API:  "/foo",
		PID:  42,
	}
	ce2 = model.ContainerEngine{
		ID:   "ce2",
		Type: "typeB",
		API:  "/bar",
		PID:  666,
	}
	g1 = model.Group{
		Name:   "groupies",
		Type:   "typeG",
		Flavor: "typeG",
		Labels: model.Labels{"foo": "bar"},
	}
	c1 = model.Container{
		ID:     "C1",
		Name:   "C1",
		Type:   ce1.Type,
		Flavor: ce1.Type,
		PID:    123,
		Engine: &ce1,
		Groups: []*model.Group{&g1},
	}
	c2 = model.Container{
		ID:     "C2",
		Name:   "C2",
		Type:   ce2.Type,
		Flavor: ce2.Type,
		PID:    456,
		Engine: &ce2,
		Groups: []*model.Group{&g1},
	}
)

func init() {
	ce1.Containers = []*model.Container{&c1}
	ce2.Containers = []*model.Container{&c2}
	g1.Containers = []*model.Container{&c1, &c2}
}

var _ = Describe("container model JSON", func() {

	var cm *ContainerModel

	BeforeEach(func() {
		cm = NewContainerModel([]*model.Container{&c1, &c2})
	})

	It("marshals containers", func() {
		jtxt, err := json.Marshal(&cm.Containers)
		Expect(err).NotTo(HaveOccurred())
		Expect(jtxt).To(MatchJSON(`
{
	"123": {
		"engine": 1,
		"groups": [
		  1
		],
		"id": "C1",
		"name": "C1",
		"type": "typeA",
		"flavor": "typeA",
		"pid": 123,
		"paused": false,
		"labels": null
	  },
	  "456": {
		"engine": 2,
		"groups": [
		  1
		],
		"id": "C2",
		"name": "C2",
		"type": "typeB",
		"flavor": "typeB",
		"pid": 456,
		"paused": false,
		"labels": null
	  }
}`))
	})

	It("unmarshals containers", func() {
		jtxt, err := json.Marshal(&cm.Containers)
		Expect(err).NotTo(HaveOccurred())
		cmu := NewContainerModel(nil)
		Expect(json.Unmarshal(jtxt, &cmu.Containers)).NotTo(HaveOccurred())
		Expect(cmu.Containers.Containers).To(ConsistOf(
			PointTo(MatchFields(IgnoreExtras, Fields{
				"ID":     Equal(c1.ID),
				"Groups": HaveLen(1),
			})),
			PointTo(MatchFields(IgnoreExtras, Fields{
				"ID":     Equal(c2.ID),
				"Groups": HaveLen(1),
			})),
		))
		Expect(cmu.Containers.Containers[uint(c1.PID)].Groups[0]).To(
			BeIdenticalTo(cmu.Containers.Containers[uint(c2.PID)].Groups[0]))
	})

	It("marshals container engines", func() {
		jtxt, err := json.Marshal(&cm.ContainerEngines)
		Expect(err).NotTo(HaveOccurred())
		Expect(jtxt).To(MatchJSON(`
{
	"1": {
		"containers": [
		  123
		],
		"id": "ce1",
		"type": "typeA",
		"api": "/foo",
		"pid": 42
	},
	"2": {
		"containers": [
			456
		],
		"id": "ce2",
		"type": "typeB",
		"api": "/bar",
		"pid": 666
	}
}`))
	})

	It("unmarshals container engines", func() {
		jtxt, err := json.Marshal(&cm.ContainerEngines)
		Expect(err).NotTo(HaveOccurred())
		cmu := NewContainerModel(nil)
		Expect(json.Unmarshal(jtxt, &cmu.ContainerEngines)).NotTo(HaveOccurred())
		Expect(cmu.ContainerEngines.enginesByRefID).To(ConsistOf(
			PointTo(MatchFields(IgnoreExtras, Fields{
				"ID":         Equal(ce1.ID),
				"Containers": HaveLen(1),
			})),
			PointTo(MatchFields(IgnoreExtras, Fields{
				"ID":         Equal(ce2.ID),
				"Containers": HaveLen(1),
			})),
		))
	})

	It("marshals groups", func() {
		jtxt, err := json.Marshal(&cm.Groups)
		Expect(err).NotTo(HaveOccurred())
		Expect(jtxt).To(MatchJSON(`
{
	"1": {
		"containers": [
		  123,
		  456
		],
		"name": "groupies",
		"type": "typeG",
		"flavor": "typeG",
		"labels": {
		  "foo": "bar"
		}
	}
}`))
	})

})
