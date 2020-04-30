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

package lxkns

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/thediveo/lxkns/nstest"
	"github.com/thediveo/lxkns/species"
	"github.com/thediveo/testbasher"
)

var _ = Describe("Discover from fds", func() {

	It("finds fd-referenced namespaces", func() {
		scripts := testbasher.Basher{}
		defer scripts.Done()
		scripts.Common(nstest.NamespaceUtilsScript)
		scripts.Script("main", `
unshare -Urn $stage2 # set up the stage with a new user ns.
`)
		scripts.Script("stage2", `
process_namespaceid net # print ID of first new net ns.
exec unshare -n 3</proc/self/ns/net $stage3 # fd-ref net ns and then replace our shell.
`)
		scripts.Script("stage3", `
process_namespaceid net # print ID of second new net ns.
read # wait for test to proceed()
`)
		cmd := scripts.Start("main")
		defer cmd.Close()
		var fdnetnsid, netnsid species.NamespaceID
		cmd.Decode(&fdnetnsid)
		cmd.Decode(&netnsid)
		Expect(fdnetnsid).ToNot(Equal(netnsid))
		// correctly misses fd-referenced namespaces without proper discovery
		// method enabled.
		opts := NoDiscovery
		opts.SkipProcs = false
		allns := Discover(opts)
		Expect(allns.Namespaces[NetNS]).To(HaveKey(netnsid))
		Expect(allns.Namespaces[NetNS]).ToNot(HaveKey(fdnetnsid))
		// correctly finds fd-referenced namespaces now.
		opts = NoDiscovery
		opts.SkipFds = false
		allns = Discover(opts)
		Expect(allns.Namespaces[NetNS]).To(HaveKey(fdnetnsid))
	})

	It("skips /proc/*/fd/* nonsense", func() {
		r := DiscoveryResult{
			Options: NoDiscovery,
			Processes: ProcessTable{
				1234: &Process{PID: 1234},
				5678: &Process{PID: 5678},
			},
		}
		r.Options.SkipFds = false
		r.Namespaces[NetNS] = NamespaceMap{}
		discoverFromFd(0, "./test/fdscan/proc", &r)
		Expect(r.Namespaces[NetNS]).To(HaveLen(1))
		Expect(r.Namespaces[NetNS]).To(HaveKey(species.NamespaceID(12345678)))

		origns := r.Namespaces[NetNS][species.NamespaceID(12345678)]
		discoverFromFd(0, "./test/fdscan/proc", &r)
		Expect(r.Namespaces[NetNS]).To(HaveLen(1))
		Expect(r.Namespaces[NetNS][species.NamespaceID(12345678)]).To(BeIdenticalTo(origns))
	})

})
