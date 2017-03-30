## db

`db` contains master and node using raft consensus algorithm. 

### Master

We only need one master to mantain the system status, so the master only for:

1. Wraps the meta data query: `register/get/update` raft peers.
2. Send heartbeats to every known node, remove any of them wich down or network partition.

### Node

1. Every node combines a HTTP server and a Raft instance.
1. HTTP server handles serveral API:
  1. `GET /i/key/:key` for get the value of the `:key`.
  2. `POST /key` for add a pair of key and value.
  3. `POST /join` for join a peer into the cluster, reject if current raft instance is not a *Leader*.
  4. `GET /ping` for master heartbeat.
  5. `GET /stats` for stats of current raft instance.