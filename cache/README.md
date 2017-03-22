### cache

Based-on `github.com/golang/groupcache`, simple HTTP master and nodes system.

#### Example

```go 
import (
  "github.com/Focinfi/oncekv/cache/master"
  "github.com/Focinfi/oncekv/cache/ndoe"
)

// create a master
masterAddr := ":5550"
master := master.New(masterAddr)

// create three nodes with HTTP server addr, groupcache server addr and the masterAddr
node1 := node.New(":5551", ":5541", masterAddr) 
node1 := node.New(":5552", ":5542", masterAddr) 
node1 := node.New(":5553", ":5543", masterAddr) 

// Run the master and the three nodes separately
master.Start()
node1.Start()
node2.Start()
node3.Start()

```

#### How the master and nodes keep in touch?

1. Master
    
    1. Serve `POST /join` for the new node
    1. Refresh the upderlying raft cluster address list
    1. Send `POST /meta` to all known nodes to keep heartbeat


2. Node
    
    1. Serve `GET /key/:key`, delegate to `groupcache`.
    1. If one node should serve some pairs of key/value but has no caches, it will get the data from raft cluster.
    1. Serve `POST /meta` from master to update the peers list.

