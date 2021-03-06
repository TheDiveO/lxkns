// Working with short auxiliary test shell scripts whose script sources can be
// kept together with the golang test code for better maintenance. Focuses on
// BASH shell scripts.

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

package nstest

import (
	"github.com/thediveo/lxkns/species"
	"github.com/thediveo/testbasher"
)

// NamespaceUtilsScript defines some convenience common script functions when
// working with namespace test auxiliary scripts.
const NamespaceUtilsScript = `
# prints the namespace ID for the namespace referenced by path $1. The
# namespace ID is printed in JSON format {"dev":..., "ino":...}.
namespaceid () {
	LC_ALL=C stat -L $1 | awk '/Inode: [0-9]+/ { print $4 }'
}
# prints the namespace ID for the namespace type $1 of the current shell process.
process_namespaceid () {
	LC_ALL=C stat -L /proc/$$/ns/$1 | awk '/Inode: [0-9]+/ { print $4 }'
}
`

// CmdDecodeNSId decodes a namespace identifier (expecting only the plain
// inode number) read from a test command and returns it.
func CmdDecodeNSId(cmd *testbasher.TestCommand) species.NamespaceID {
	var ino uint64
	cmd.Decode(&ino)
	return species.NamespaceIDfromInode(ino)
}
