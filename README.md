## oncekv [![Build Status](https://travis-ci.org/Focinfi/oncekv.png)](https://travis-ci.org/Focinfi/oncekv.svg?branch=master)

A key/value database, each pair of key/value can be set only one time.

### Design

![diagram](http://on78mzb4g.bkt.clouddn.com/architeture.png)

1. Performance, using groupcache for read cache.
1. Reliable, using raft.
1. Easy horizontal scaling, simple HTTP API.

### Admin Page

![admin](http://on78mzb4g.bkt.clouddn.com/oncekv-admin.jpeg.webp)

Powered by [vuejs](https://github.com/vuejs) and [element](https://github.com/ElemeFE/element)