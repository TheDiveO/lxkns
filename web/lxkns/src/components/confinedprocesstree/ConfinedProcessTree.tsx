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

import React, { useContext, useState } from 'react';

import ExpandMoreIcon from '@material-ui/icons/ExpandMore';
import ChevronRightIcon from '@material-ui/icons/ChevronRight';

import TreeView from '@material-ui/lab/TreeView';
import TreeItem from '@material-ui/lab/TreeItem';

import { DiscoveryContext } from 'components/discovery';
import { compareNamespaceById, compareProcessByNameId, Namespace, NamespaceType, Process } from 'models/lxkns';
import NamespaceInfo from 'components/namespaceinfo/NamespaceInfo';
import ProcessInfo from 'components/processinfo'


/**
 * Searches for sub-processes of a given process which are still in the same
 * PID namespace as the process we started from, but which have different
 * controllers (cgroup paths). Returns a flat list of the next-level sub
 * processes.
 *
 * @param proc process to start the search from.
 */
const findSubProcesses = (proc: Process): Process[] => {
    // We'll work only on children which are still in the same namespace, all
    // other children can immediately be filtered out.
    const children = proc.children
        .filter(child => child.namespaces.pid === proc.namespaces.pid)
    // We need to recursively check children which are controlled by the same
    // controller as our process, because a change in the controller might be
    // further down the process tree.
    const subprocs = children
        .filter(child => child.cgroup === proc.cgroup)
        .map(child => findSubProcesses(child))
        .flat(1)
    // Finally return the concatenation of all immediate child processes as
    // well as processes further down the hierarchy with controllers differing
    // to our controller.
    return children
        .filter(child => child.cgroup !== proc.cgroup)
        .concat(subprocs)
}

/**
 * Renders a process and then recursively decends down to find and render
 * deeper processes which still belong to the same PID namespace, yet have a
 * different controller (cgroup path).
 *
 * @param proc process
 */
const confinedProcessTreeItem = (proc: Process) => {

    const children = findSubProcesses(proc)
        .sort(compareProcessByNameId)
        .map(child => confinedProcessTreeItem(child))

    return (
        <TreeItem
            key={proc.pid}
            nodeId={proc.pid.toString()}
            label={<ProcessInfo process={proc} />}
        >{children}</TreeItem>
    )
}


const namespaceProcesses = (namespace: Namespace) => {

    const procs = namespace.leaders
        .sort(compareProcessByNameId)
        .map(proc => confinedProcessTreeItem(proc))

    return (<>
        <TreeItem
            key={namespace.nsid}
            nodeId={namespace.nsid.toString()}
            label={<NamespaceInfo namespace={namespace} />}
        >
            {procs}
            {namespace.children &&
                namespace.children.map(childns => namespaceProcesses(childns))}
        </TreeItem>
    </>)
}

export interface ConfinedProcessTreeProps {
    type?: string
}

export const ConfinedProcessTree = (props: ConfinedProcessTreeProps) => {

    const type = props.type as NamespaceType || NamespaceType.pid

    // Discovery data comes in via a dedicated discovery context.
    const discovery = useContext(DiscoveryContext)

    // Tree node expansion is a component-local state.
    const [expanded, setExpanded] = useState([])

    const rootnsItems = Object.values(discovery.namespaces)
        .filter(ns => ns.type === type && ns.parent == null)
        .sort(compareNamespaceById)
        .map(ns => namespaceProcesses(ns));

    // Whenever the user clicks on the expand/close icon next to a tree item,
    // update the tree's expand state accordingly. This allows us to
    // explicitly take back control (ha ... hah ... HAHAHAHA!!!) of the expansion
    // state of the tree.
    const handleToggle = (event, nodeIds) => {
        setExpanded(nodeIds);
    }

    return (
        <TreeView
            className="namespacetree"
            onNodeToggle={handleToggle}
            defaultCollapseIcon={<ExpandMoreIcon />}
            defaultExpandIcon={<ChevronRightIcon />}
            expanded={expanded}
        >{rootnsItems}</TreeView>
    );

};

export default ConfinedProcessTree;
