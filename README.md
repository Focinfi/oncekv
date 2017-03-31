## oncekv 
[![Go Report Card](https://goreportcard.com/badge/github.com/Focinfi/oncekv)](https://goreportcard.com/report/github.com/Focinfi/oncekv)
[![Build Status](https://travis-ci.org/Focinfi/oncekv.png)](https://travis-ci.org/Focinfi/oncekv.svg?branch=master)

A key/value database, each pair of key/value can be set **only one time**.

Why limit the set time?

Cause we design oncekv for the scenarios like once a key set, it will never be changed, for example, a versioned cache file database.

At the same time, limitation gives us chances to improve performance but still easily to scale.

### Design

![diagram](http://on78mzb4g.bkt.clouddn.com/architeture.png)

1. Performance, using [groupcache](http://github.com/golang/groupcache) for read cache.
1. Reliable, using [raft](https://github.com/hashicorp/raft) for consensus and [boltdb](github.com/boltdb/bolt) for underlying database.
1. Easy horizontal scaling, simple HTTP API, inspired by [hraftf](https://github.com/otoolep/hraftd).

### Have a try

0. Change working directory into `example`

```
cd $GOPATH/src/github.com/Focinfi/oncekv/example
```

1. Start the admin server

```
go run ./admin/admin.go
```

This server provide API of known server list for the front end porject and starts the database master server.

1. Start the admin front end server
```
cd $GOPATH/src/github.com/Focinfi/oncekv/admin/oncekv
npm run dev
```

It shows only title, now, let's add a database node.

1. Start one database node

```
bash ./databases/node/node1.sh
```

Cause it's the only node of this raft cluster, so it becomes the **Leader**.
And the admin page will show its stats.

Then add another node by `bash ./databases/node/node2.sh`

1. Start the cache cluster

Start the master:

```
go run ./cache/master/master.go;
```

Start the cache first node:

```
bash ./cache/node/node1.sh
```

Start the cache second node:

```
bash ./cache/node/node1.sh
```

Start the cache third node:

```
bash ./cache/node/node1.sh
```

1. Set/Get some key/vlaue pairs:

```
# set foo/bar into database leader
curl -XPOST http://127.0.0.1:11000/key -d '{"key":"foo","value":"bar"}'

# get for cache node-1
curl -XGET http://127.0.0.1:5551/key/foo

# get for cache node-2
curl -XGET http://127.0.0.1:5552/key/foo

# get for cache node-3
curl -XGET http://127.0.0.1:5553/key/foo

#... try more key/value
```

And then your admin page will look like this:


![admin](http://on78mzb4g.bkt.clouddn.com/oncekv-admin.jpeg.webp)

Powered by [vuejs](https://github.com/vuejs/vue) and [element](https://github.com/ElemeFE/element)