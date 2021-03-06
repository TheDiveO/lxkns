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
	"encoding/json"
	"net/http"

	"github.com/thediveo/lxkns"
	"github.com/thediveo/lxkns/api/types"
	"github.com/thediveo/lxkns/log"
	"github.com/thediveo/lxkns/species"
)

// GetNamespacesHandler returns the results of a namespace discovery, as JSON.
// Additionally, we opt in to mount path+point discovery.
func GetNamespacesHandler(w http.ResponseWriter, req *http.Request) {
	opts := lxkns.FullDiscovery
	opts.WithMounts = true
	allns := lxkns.Discover(opts)
	// Note bene: set header before writing the header with the status code;
	// actually makes sense, innit?
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(
		types.NewDiscoveryResult(types.WithResult(allns))) // ...brackets galore!!!
	if err != nil {
		log.Errorf("namespaces discovery error: %s", err.Error())
	}
}

// GetProcessesHandler returns the process table with namespace references, as
// JSON.
func GetProcessesHandler(w http.ResponseWriter, req *http.Request) {
	opts := lxkns.NoDiscovery
	opts.SkipProcs = false
	disco := lxkns.Discover(opts)

	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(
		types.NewProcessTable(types.WithProcessTable(disco.Processes)))
	if err != nil {
		log.Errorf("processes discovery error: %s", err.Error())
	}
}

// GetPIDMapHandler returns data for translating PIDs between hierarchical PID
// namespaces, as JSON.
func GetPIDMapHandler(w http.ResponseWriter, req *http.Request) {
	opts := lxkns.FullDiscovery
	opts.NamespaceTypes = species.CLONE_NEWPID
	pidmap := lxkns.NewPIDMap(lxkns.Discover(opts))

	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(
		types.NewPIDMap(types.WithPIDMap(pidmap)))
	if err != nil {
		log.Errorf("pid translation map discovery error: %s", err.Error())
	}
}
