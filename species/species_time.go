// Defines CLONE_NEWTIME based on unix.CLONE_NEWTIME for go 1.14+.

// Copyright 2020 Harald Albrecht.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build linux
// +build go1.14

package species

import (
	"golang.org/x/sys/unix"
)

// Since Go 1.14 the unix package provides the required CLONE_NEWTIME constant,
// so we're using it if possible, otherwise a fallback definition.
const cloneNewtime = NamespaceType(unix.CLONE_NEWTIME)
