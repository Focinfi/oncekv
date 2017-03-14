## oncekv

A key/value database, each pair of key/value can be set only one time.

### Design

1. Performance, using groupcache for read cache.
1. Reliable, using raft.
1. Easy horizontal scaling.

