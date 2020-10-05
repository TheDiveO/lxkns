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

import React, { useEffect, useContext, useState, useRef } from 'react';

import ExpandMoreIcon from '@material-ui/icons/ExpandMore';
import ChevronRightIcon from '@material-ui/icons/ChevronRight';

import TreeView from '@material-ui/lab/TreeView';
import TreeItem from '@material-ui/lab/TreeItem';

import { DiscoveryContext } from 'components/discovery';
import { compareNamespaceById, compareProcessByNameId, Namespace, NamespaceMap, NamespaceType, Process } from 'models/lxkns';
import NamespaceInfo from 'components/namespaceinfo/NamespaceInfo';
import ProcessInfo from 'components/processinfo'
import { Typography } from '@material-ui/core';

// TODO:
const hideSystemProcs = true

const showProcess = (process: Process) =>
    !hideSystemProcs ||
    (process.pid > 2 &&
        !process.cgroup.startsWith('/system.slice/') &&
        !process.cgroup.startsWith('/init.scope/') &&
        process.cgroup !== '/user.slice')

/**
 * Searches for sub-processes of a given process which are still in the same
 * PID namespace as the process we started from, but which have different
 * controllers (cgroup paths). Returns a flat list of the next-level sub
 * processes.
 *
 * @param proc process to start the search from.
 */
const findSubProcesses = (proc: Process, nstype: NamespaceType): Process[] => {
    // We'll work only on children which are still in the same namespace, all
    // other children can immediately be filtered out.
    const children = proc.children
        .filter(child => child.namespaces[nstype] === proc.namespaces[nstype])
    // We need to recursively check children which are controlled by the same
    // controller as our process, because a change in the controller might be
    // further down the process tree.
    const subprocs = children
        .filter(child => child.cgroup === proc.cgroup)
        .map(child => findSubProcesses(child, nstype))
        .flat(1)
    // Finally return the concatenation of all immediate child processes as
    // well as processes further down the hierarchy with controllers differing
    // to our controller.
    return children
        .filter(child => child.cgroup !== proc.cgroup)
        .concat(subprocs)
}

const findNamespaceProcesses = (namespace: Namespace) =>
    namespace.leaders.concat(
        namespace.leaders.map(leader => findSubProcesses(leader, namespace.type)).flat(1))

/**
 * Renders a process and then recursively decends down to find and render
 * deeper processes which still belong to the same type of namespace, yet have
 * a different controller (cgroup path).
 *
 * @param proc process
 * @param nstype type of namespace confining the search for further
 * sub-processes still considered to be confined in the same namespace.
 */
const controlledProcessTreeItem = (proc: Process, nstype: NamespaceType) => {

    const children = findSubProcesses(proc, nstype)
        .sort(compareProcessByNameId)
        .map(child => controlledProcessTreeItem(child, nstype))
        .flat(1)

    // Special case: this is the only leader process in the namespace and there
    // are no (further) sub-processes with different controllers.
    const hideMe = proc.namespaces[nstype].leaders.length === 1 &&
        proc === proc.namespaces[nstype].ealdorman

    return (
        (!hideMe && showProcess(proc) &&
            <TreeItem
                key={proc.pid}
                nodeId={proc.pid.toString()}
                label={<ProcessInfo process={proc} />}
            >{children}</TreeItem>
        ) || children
    )
}

/**
 * Renders a single namespace node including processes joined to this namespace,
 * as well as child namespaces (hierarchical namespaces only). Instead of just
 * dumping a rather useless plain process tree, this component renders only
 * leaders and then sub-processes in different cgroups.
 *
 * @param namespace namespace information.
 */
const NamespaceTreeItem = (namespace: Namespace) => {

    // Get the leader processes and maybe some sub-processes (in different
    // cgroups), all inside this namespace. Please note that if there is only a
    // single leader process, then it won't show up -- it has already been
    // indicated as part of the namespace information and thus
    // controlledProcessTreeItem will drop it.
    const procs = namespace.leaders
        .sort(compareProcessByNameId)
        .map(proc => controlledProcessTreeItem(proc, namespace.type))
        .flat(1)

    // In case of hierarchical namespaces also render the child namespaces.
    const childnamespaces = namespace.children ?
        namespace.children.map(childns => NamespaceTreeItem(childns)) : []

    return <TreeItem
        key={namespace.nsid}
        nodeId={namespace.nsid.toString()}
        label={<NamespaceInfo namespace={namespace} />}
    >{procs.concat(childnamespaces)}</TreeItem>
}

// FIXME:
export const EXPANDALL_ACTION = "expandall";
export const COLLAPSEALL_ACTION = "collapseall";


const collapsedids = (namespaces: NamespaceMap, type: NamespaceType) => {
    const allrootns = Object.values(namespaces)
        .filter(ns => ns.type === type && ns.parent == null)
    const allleaderids = allrootns.map(ns => ns.leaders).flat(1)
        .map(proc => proc.pid.toString())
    return allrootns.map(ns => ns.nsid.toString())
        .concat(allleaderids)
}

export interface NamespaceProcessTreeProps {
    type?: string
    action: string
}

/**
 * Component `NamespaceProcessTree` renders a tree of namespaces of a specific
 * type only, with their contained processes. Here, contained processes are not
 * only leader processes in a namespace, but also (grand) child processes within
 * the same namespace, but with different controllers (cgroup paths). In case of
 * non-hierarchical namespace types, the namespace tree is flat.
 *
 * @param type type of namespace.
 */
export const NamespaceProcessTree = ({ type, action }: NamespaceProcessTreeProps) => {

    const nstype = type as NamespaceType || NamespaceType.pid

    // Discovery data comes in via a dedicated discovery context.
    const discovery = useContext(DiscoveryContext)

    // Previous discovery information, if any.
    const previousDiscovery = useRef({ namespaces: {}, processes: {} });

    // Tree node expansion is a component-local state.
    const [expanded, setExpanded] = useState([])

    // To emulate actions via react's properties architecture and then getting
    // the dependencies correct, we need to store the previous action. Sigh,
    // bloat react-ion.
    const oldaction = useRef("")

    // Trigger an action when the action "state" changes; we are ignoing any
    // stuff appended to the commands, as we need to add noise to the commands
    // in order to make state changes trigger. Oh, well, bummer.
    useEffect(() => {
        if (action === oldaction.current) {
            return;
        }
        oldaction.current = action;
        if (action.startsWith(EXPANDALL_ACTION)) {
        } else if (action.startsWith(COLLAPSEALL_ACTION)) {
            setExpanded(collapsedids(discovery.namespaces, nstype))
        }
    }, [action, nstype, discovery])

    useEffect(() => {
        // FIXME:
        setExpanded(collapsedids(discovery.namespaces, nstype))
        previousDiscovery.current = discovery
    }, [nstype, discovery])

    const rootnsItems = Object.values(discovery.namespaces)
        .filter(ns => ns.type === nstype && ns.parent == null)
        .sort(compareNamespaceById)
        .map(ns => NamespaceTreeItem(ns));

    // Whenever the user clicks on the expand/close icon next to a tree item,
    // update the tree's expand state accordingly. This allows us to
    // explicitly take back control (ha ... hah ... HAHAHAHA!!!) of the expansion
    // state of the tree.
    const handleToggle = (event, nodeIds) => {
        setExpanded(nodeIds);
    }

    return (rootnsItems.length &&
        <TreeView
            className="namespacetree"
            onNodeToggle={handleToggle}
            defaultCollapseIcon={<ExpandMoreIcon />}
            defaultExpandIcon={<ChevronRightIcon />}
            expanded={expanded}
        >{rootnsItems}</TreeView>
    ) || (Object.keys(discovery.namespaces).length &&
        <Typography variant="body1" color="textSecondary">
            this Linux system doesn't have any "{nstype}" namespaces
            </Typography>
        ) || (
            <Typography variant="body1" color="textSecondary">
                nothing discovered yet, please refresh
            </Typography>
        )
}

export default NamespaceProcessTree;
