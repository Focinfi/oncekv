## db

`db` contains master and node using raft consensus algorithm. 

### Master

We only need one master to mantain the system status, so the master only for:

1. Wraps the meta data query: `register/get/update` raft peers.
2. Send heartbeats to every known node, remove any of them wich down or network partition.
###