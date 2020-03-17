// A visitor implementing the view on the process tree and PID namespaces.

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

package main

import (
	"fmt"
	"os/user"
	"reflect"
	"sort"

	"github.com/thediveo/lxkns"
)

// PIDVisitor is an asciitree.Visitor which works on discovery results and
// visits them in order to produce a process tree. Differing from `ps fax`, we
// also show the PID namespaces inbetween the process hierarchy where the PID
// namespace changes from one to another.
type PIDVisitor struct {
	Details   bool
	PIDMap    *lxkns.PIDMap
	RootPIDNS lxkns.Namespace
}

// Roots returns the given "topmost" hierarchical process namespaces sorted;
// it will be called on the list of "topmost" PID namespace(s). It won't be
// called ever afterwards. In our case, we'll only ever pass in a slice of
// exactly one PID namespace, the "root" PID namespace. However, we leave this
// code in for instructional purposes.
func (v *PIDVisitor) Roots(roots reflect.Value) (children []reflect.Value) {
	pidroots := lxkns.SortNamespaces(roots.Interface().([]lxkns.Namespace))
	count := len(pidroots)
	children = make([]reflect.Value, count)
	for idx := 0; idx < count; idx++ {
		children[idx] = reflect.ValueOf(pidroots[idx])
	}
	return
}

// Label returns a node label text, which varies depending on whether the node
// is a Process or a (PID) Namespace. In case of a PID Namespace, the label
// will show the namespace type and its ID, as well as the owner name and UID
// (via the owning user Namespace). If it's a Process instead, then the text
// label contains the name and "global" PID, but also the translated "local"
// PID (which is the PID as seen from inside the PID namespace of the
// Process).
func (v *PIDVisitor) Label(node reflect.Value) (label string) {
	if proc, ok := node.Interface().(*lxkns.Process); ok {
		// It's a Process; do we have namespace information for it? If yes,
		// then we can translate between the process-local PID namespace and
		// the "initial" PID namespace.
		if procpidns := proc.Namespaces[lxkns.PIDNS]; procpidns != nil {
			localpid := v.PIDMap.Translate(proc.PID, v.RootPIDNS, procpidns)
			if localpid != proc.PID {
				return fmt.Sprintf("%q (%d=%d)", proc.Name, proc.PID, localpid)
			}
			return fmt.Sprintf("%q (%d)", proc.Name, proc.PID)
		}
		// PID namespace information is NOT known, so this is a process out of
		// our reach. We thus print it in a way to signal that we don't know
		// about this process' PID namespace
		return fmt.Sprintf("pid:[???] %q (%d=???)", proc.Name, proc.PID)
	}
	// It's a PID namespace, so we give details about the ID and the owner's
	// UID and name.
	pidns := node.Interface().(lxkns.Namespace)
	label = pidns.(lxkns.NamespaceStringer).TypeIDString()
	if pidns.Owner() != nil {
		uid := pidns.Owner().(lxkns.Ownership).UID()
		var userstr string
		if u, err := user.LookupId(fmt.Sprintf("%d", uid)); err == nil {
			userstr = fmt.Sprintf(" (%q)", u.Username)
		}
		label += fmt.Sprintf(", owned by UID %d%s", uid, userstr)
	}
	return
}

// Get is called on nodes which can be either (1) PID namespaces or (2)
// processes. For (1), the visitor returns information about the PID
// namespace, but then specifies processes as children. For (2), the visitor
// returns process children, unless these children are in a different PID
// namespace: then, their PID namespaces are returned instead. Polymorphism
// galore!
func (v *PIDVisitor) Get(node reflect.Value) (
	label string, properties []string, children reflect.Value) {
	// Label for this (1) PID namespace or (2) process.
	label = v.Label(node)
	// Children of this (1) PID namespace are always processes, but for (2)
	// processes the children can be any combination of (a) child processes
	// still in the same namespace and (b) child PID namespaces.
	clist := []interface{}{}
	if proc, ok := node.Interface().(*lxkns.Process); ok {
		// TODO:
		pidns := proc.Namespaces[lxkns.PIDNS]
		childprocesses := lxkns.ProcessListByPID(proc.Children)
		sort.Sort(childprocesses)
		for _, childproc := range childprocesses {
			if childproc.Namespaces[lxkns.PIDNS] == pidns {
				clist = append(clist, childproc)
			} else {
				// We might also end up here in case we have insufficient
				// privileges (capabilities) to discover the PID namespace of
				// a process. In this case, we only can dump the processes,
				// but with a signature indicating that we don't known about
				// their namespaces. Otherwise, we insert a PID namespace
				// node, from which the tree will branch into that PID
				// namespace's leader processes.
				cpidns := childproc.Namespaces[lxkns.PIDNS]
				if cpidns == nil {
					clist = append(clist, childproc)
				} else {
					clist = append(clist, cpidns)
				}
			}
		}
	} else {
		// The child nodes of a PID namespace tree node will be the "leader"
		// (or "topmost") processes inside the PID namespace.
		leaders := lxkns.ProcessListByPID(node.Interface().(lxkns.Namespace).Leaders())
		sort.Sort(leaders)
		for _, proc := range leaders {
			clist = append(clist, proc)
		}
	}
	children = reflect.ValueOf(clist)
	return
}