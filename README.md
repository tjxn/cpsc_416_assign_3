<h1 align="center"> 
	CPSC 416 - Distributed Systems: Assignment 3
</h1>

<h2 align="center"> 
	University of British Columbia
</h2>

<h3>
	<b>High-Level Overview</b>
</h3>
This assignment outline comes from the [UBC CPSC 416 course webpage](https://www.cs.ubc.ca/~bestchai/teaching/cs416_2015w2/assign3/index.html).

You will be provided with an implementation of a simple key-value service that 
implements an RPC-accessible hash table in which keys are associated with values. 
Nodes in the system should not communicate with each other directly and must use 
this service for all communication. The following diagram is a high-level 
view of the setup in this assignment: 

<p align="center">
	<img alt="Key-Value Diagram" src="/arch.jpg">
</p>

Your task is to implement node logic that allows an arbitrary set of active nodes 
to agree on a "leader" node. If the leader fails, the remaining nodes should elect
a new leader. Once elected, the leader must determine the active nodes in the system 
and advertise this set to all the nodes in the system (through the key-value service).
The set of active nodes may change (as nodes may fail or join the system) and the 
leader must re-advertise the node set to reflect these events. Active nodes should 
periodically retrieve this list of active nodes and print it out.

Individual keys in the key-value service may experience permanent unavailability. 
Your node implementation must be robust to such unavailability and continue to elect
leaders that will properly advertise the set of active nodes.

<h3>
	<b>Key-Value Service</b>
</h3>

The key value service associates string keys with string values. Think of it as a 
remote hash table, providing the same strong consistency semantics that you would 
expect of a local hash table. The service starts with an empty hash table: every 
key is mapped to the empty string "". The service supports the following three 
*atomic* operations via RPC:

 - ``curr-value`` ← get(``key``)
	- Retrieves the current value for a key. ``curr-value`` contains the current 
	value associated with ``key``, or is set to "unavailable" if ``key`` is unavailable.

 - ``ret-val`` ← put(``key``, ``value``)
	- Associates ``value`` with ``key``. ``ret-val`` is either "", which indicates
	success, or is set to "unavailable" if the key is unavailable.

 - ``curr-value`` ← testset(``key``, ``test-value``, ``new-value``)
	- Tests if the current value associated with ``key`` is ``test-value``. If yes, then 
	it associates ``key`` with ``new-value`` and returns ``new-value``. Otherwise, it returns 
	the value currently associated with ``key``. ``curr-value`` is set to "unavailable" if 
	the key is unavailable.
	
One of the arguments to the key-value service implementation is a *key failure 
probability*. This controls the likelihood of a key becoming unavailable during 
any one of the above three operations. Initially all keys are available. Once a 
key becomes unavailable, it is a permanent unavailability (i.e., until the service
is restarted). A key's availability is independent from the availability of other
keys in the key-value service. When a key is unavailable, the return value for an 
operation is always set to "unavailable".	

Download the [key-value service](https://github.com/tjxn/cpsc_416_assign_3/blob/master/kvservicemain.go) implementation and an example [client](https://github.com/tjxn/cpsc_416_assign_3/blob/master/kvclientmain.go) that exercises the service.

<h3>
	<b>Implementation Requirements</b>
</h3>

 - All nodes run the same code.

 - All nodes communicate only indirectly, through the key-value service. 
	This means that a node does not know which other nodes are participating 
	in the system, how many there are in total, where they are located, etc.
 
 - Given a sufficiently long time during which failures do not occur, an active 
	(i.e., alive) node is eventually elected as a leader.
 
 - Given a sufficiently long time during which failures do not occur, the elected 
	leader will eventually advertise an accurate list of all active nodes in the system.
	And, each active node (including the leader) will retrieve the latest 
	version of this list from the key-value service.
		
		- Each node must continually print a listing of the set of active nodes and
		the leader node id to stdout, one listing per line, in the following format:
		
		``ID1 ID2 ID3 ... IDn``
		
		Where IDi is an active node's id and the first id in the list (i.e., ID1) 
		indicates the id of the current leader node. The other active node ids in 
		the listing do not have to be appear in any particular order. Note that it 
		is sufficient to print a new listing whenever the leader/active nodes 
		information changes.
		
 - Your implementation must be robust to node halting failures, including leader halting failures.
 
 - Your implementation must be robust to nodes that restart (i.e., halt and later re-join the 
	system with the same identity).
	
 - Your implementation must be robust to varying RPC times.
 
 - You cannot change the implementation of the key-value service. I gave you the key-value service 
	code so that you can experiment with it locally. But, your solution should work 
	even if I administer the key value service (i.e., you don't have control over the 
	service).
 
 
 
 
<h3>
	<b>Assumptions You Can Make</b>
</h3>

 - The key-value service does not fail/restart and does not misbehave.
 - No network failures.
 - Each node has a unique identifier (specified on the command line).
 - The key-value service is dedicated to nodes in your system (one KV-service per student group).
	
<h3>
	<b>Assumptions You Cannot Make</b>
</h3>

 - Nodes have synchronized clocks (e.g., running on the same physical host).
 
<h3>
	<b>Solution Spec</b>
</h3>

Write a go program ``node.go`` that implements a node in the system, as described above, 
and has the following usage:

``go run node.go [ip:port] [id]``

 - [ip:port] : address of the key-value service
 - [id] : a unique string identifier for the node (no spaces)

