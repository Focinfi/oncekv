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

### Admin Page

![admin](http://on78mzb4g.bkt.clouddn.com/oncekv-admin.jpeg.webp)

Powered by [vuejs](https://github.com/vuejs/vue) and [element](https://github.com/ElemeFE/element)