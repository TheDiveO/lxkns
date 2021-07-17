// barebones no-thrills example that lists *all* namespaces and containers.

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
	"context"
	"fmt"

	"github.com/thediveo/gons/reexec"
	"github.com/thediveo/lxkns"
	"github.com/thediveo/lxkns/containerizer/whalefriend"
	"github.com/thediveo/lxkns/model"
	"github.com/thediveo/whalewatcher/watcher"
	"github.com/thediveo/whalewatcher/watcher/moby"
)

func main() {
	reexec.CheckAction() // must be called before a standard discovery

	// Set up a Docker engine-connected containerizer
	moby, err := moby.NewWatcher("")
	if err != nil {
		panic(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cizer := whalefriend.New(ctx, []watcher.Watcher{moby})

	// Run the discovery, including containerization.
	result := lxkns.Discover(
		lxkns.WithStandardDiscovery(), lxkns.WithContainerizer(cizer))

	for nsidx := model.MountNS; nsidx < model.NamespaceTypesCount; nsidx++ {
		for _, ns := range result.SortedNamespaces(nsidx) {
			fmt.Println(ns.String())
		}
	}
}