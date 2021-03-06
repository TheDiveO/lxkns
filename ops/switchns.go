// Switching namespaces.

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

package ops

import (
	"fmt"
	"runtime"

	"github.com/thediveo/lxkns/ops/internal/opener"
	"github.com/thediveo/lxkns/ops/relations"
	"golang.org/x/sys/unix"
)

// Go runs the specified function as a new Go routine and from a locked OS
// thread, while joined to the specified namespaces. When the specified function
// returns, its Go routine will also terminate and the underlying OS thread will
// be destroyed. This avoids subtle problems further down the road in case there
// were namespace switching issues which overwise would carry over into any code
// executed after invoking Go(). Go() returns nil if switching namespaces
// succeeded, else an error. Please note that Go() returns as soon as switching
// namespaces has finished. The specified function is then run in its own Go
// routine.
func Go(f func(), nsrefs ...relations.Relation) error {
	started := make(chan error)
	go func() {
		// Lock, but never unlock the OS thread exclusively powering our Go
		// routine. This ensures that the Golang runtime will destroy the OS
		// thread and never attempts to reuse it.
		runtime.LockOSThread()
		// Switch our highly exclusive OS thread into the specified
		// namespaces...
		for _, nsref := range nsrefs {
			// Important: since nsref.Reference() returns a file descriptor
			// which potentially is derived from an open os.File, the latter
			// must not get garbage collected while we attempt to use the file
			// descriptor, as otherwise the os.File's finalizer will have closed
			// the fd prematurely. Luckily (hopefully not!) the (varargs) slice
			// won't be collectible until the iteration terminates, keeping its
			// slice elements and thus its os.Files (if any) alive. In
			// consequence, we don't need an explicit runtime.KeepAlive(...)
			// here.
			fd, closer, err := nsref.(opener.Opener).NsFd()
			if err != nil {
				started <- fmt.Errorf("lxkns.Go: cannot reference namespace, %w", err)
				return // ex-terminate ;)
			}
			err = unix.Setns(fd, 0)
			closer()
			if err != nil {
				started <- fmt.Errorf(
					"lxkns.Go: cannot enter namespace %s, %w",
					nsref.(fmt.Stringer).String(), err)
				return
			}
		}
		// Our preparations are finally done, so let's call the desired function and
		// then call it a day.
		started <- nil
		f()
	}()
	// Wait for the goroutine to have finished switching namespaces and about to
	// invoke the specified function. We're lazy and are never closing the
	// channel, but it will get garbage collected anyway.
	return <-started
}

// Execute a function synchronously while switched into the specified
// namespaces, then returns the interface{} outcome of calling the specified
// function. If switching fails, Execute returns an error instead.
func Execute(f func() interface{}, nsrefs ...relations.Relation) (interface{}, error) {
	result := make(chan interface{})
	if err := Go(func() {
		result <- f()
	}, nsrefs...); err != nil {
		return nil, err
	}
	return <-result, nil
}

// Visit locks the OS thread executing the current go routine, then switches the
// thread into the specified namespaces, and executes f(). Afterwards, it
// switches the namespaces back to their original settings and unlocks the
// underlying OS thread.
//
// If switching namespaces back fails, then the OS thread is tainted and will
// remain locked. As simple as this sounds, Visit() is a dangerous thing: as
// long as the wheels keep spinning we're in the sunlit uplands. But as soon as
// the gears jam ... good luck. Visit() mainly exists for those optimizations
// where creating new OS threads is deemed to much overhead and namespace
// switch-back usually is possible. However, such uses must be prepared for
// Visit() to fail and then act accordingly: namely, terminate the Go routine so
// the runtime can kill its locked OS thread.
//
// Visit() should never be called from the main Go routine, as any failure in
// switching namespaces leaves us with a tainted OS thread for the main Go
// routine. Yuk!
func Visit(f func(), nsrefs ...relations.Relation) (err error) {
	runtime.LockOSThread()
	tid := unix.Gettid()
	// Keep record of the namespaces we are leaving, so we can switch back
	// afterwards. We're recording only those namespaces we're switching.
	switchback := make([]origNS, 0, len(nsrefs))

	// Ensure that switching back namespaces and error reporting is always done,
	// even if f() panics. This tries to unwind the mess we've created before.
	defer func() {
		// Switch back into the original namespaces which were active when
		// calling Visit(), but switch back in reverse order. If this causes an
		// error and we haven't registered any error so far, then report the
		// switch-back problem.
		for idx := len(switchback) - 1; idx >= 0; idx-- {
			if seterr := unix.Setns(switchback[idx].fd, 0); seterr != nil && err == nil {
				// In case we didn't registered an err so far, take this ... but
				// ignore any subsequent errs.
				err = fmt.Errorf(
					"lxkns.Visit: cannot switch back to previously active namespace: %w",
					seterr)
			}
			switchback[idx].closer()
		}
		// If there was an error and we already switched at least one namespace,
		// then this OS thread is toast and we won't unlock it.
		if err == nil || len(switchback) == 0 {
			runtime.UnlockOSThread()
		}
	}()

	// With unwinding out of the way, let's start switching namespaces. This is
	// ugly business as we need to record the originally active namespaces at
	// the same time, and still handle potential errors everywhere. We cannot
	// simply record all 8 original namespaces, but only those we're switching.
	// And as we don't know the types of namespaces to switch into, we need to
	// query that information too. Again, more error handling, without any
	// chance to defer().
	for _, nsref := range nsrefs {
		// Use the optimization which opens a typed namespace for us, so we get
		// not only an OS-level file descriptor for referencing the namespace,
		// but also its (optionally foretold) type.
		var openref relations.Relation
		var refcloser opener.ReferenceCloser
		// no ":=", don't shadow err!
		openref, refcloser, err = nsref.(opener.Opener).OpenTypedReference()
		if err != nil {
			err = fmt.Errorf("lxkns.Visit: cannot reference namespace, %w", err)
			return // ...unwind entangled namespaces
		}
		var fd int
		var fdcloser opener.FdCloser
		// no ":=", don't shadow err!
		fd, fdcloser, err = openref.(opener.Opener).NsFd()
		if err != nil {
			refcloser()
			err = fmt.Errorf("lxkns.Visit: cannot reference namespace, %w", err)
			return // ...unwind entangled namespaces
		}
		nstype, _ := openref.Type() // already fetched during OpenTypedReference().
		nstypename := nstype.Name()
		if nstypename == "" {
			fdcloser()
			refcloser()
			err = fmt.Errorf(
				"lxkns.Visit: cannot determine type of %s", nsref)
			return // ...unwind entangled namespaces
		}
		oldnsref := NamespacePath(fmt.Sprintf("/proc/%d/ns/%s", tid, nstype.Name()))
		var oldfd int
		var oldfdcloser opener.FdCloser
		oldfd, oldfdcloser, err = oldnsref.NsFd()
		if err != nil {
			fdcloser()
			refcloser()
			err = fmt.Errorf(
				"lxkns.Visit: cannot save currently active namespace, %w", err)
			return // ...unwind entangled namespaces
		}
		// Finally, we're ready to jump, erm, switch into this specific
		// namespace.
		err = unix.Setns(fd, 0)
		fdcloser()
		refcloser()
		if err != nil {
			oldfdcloser()
			err = fmt.Errorf("lxkns.Visit: cannot enter namespace, %w", err)
			return // ...unwind entangled namespaces
		}
		// We succeeded, so record the old namespace to later switch back to it.
		switchback = append(switchback, origNS{
			nsref:  oldnsref, // ...keep it alive and away from early GC!
			fd:     oldfd,
			closer: oldfdcloser,
		})
	}
	// Call the specified f() with namespaces switches as desired.
	f()
	return
}

// origNS stores information for switching back to a namespace and cleaning up.
type origNS struct {
	nsref  relations.Relation // keeps the original namespace ref from getting gc'ed.
	fd     int                // open fd referencing the original namespace.
	closer opener.FdCloser    // don't leak fds; this knows how to act correctly.
}
