# Scheduler Redis Cluster persistence

## Scope

- Redis Cluster requires database 0. Multiple endpoints without a Sentinel
  master name select Cluster mode; a nonzero database is rejected explicitly.
- Persistent scheduler host, peer, task, and association keys share a scheduler
  cluster ID hash tag so that multi-key Lua scripts operate in one hash slot.
- Persistent task and peer scans use the primary owning that hash tag rather
  than an arbitrary Cluster node.

## Persistence upgrade warning

The key prefix changes from `scheduler:scheduler-clusters:1:` to
`scheduler:scheduler-clusters:{1}:`, including standalone and Sentinel deployments.

This is a breaking storage-format change, not a transparent in-place upgrade.
The new code does not read old keys. Back up persistent metadata, pause persistent
task writes, and either migrate the old keys or explicitly allow metadata to be
rebuilt. Upgrade all scheduler replicas in the same cluster together; do not run
old and new key formats concurrently. Do not use `FLUSHDB` or `FLUSHALL` on a
shared Redis deployment. File caches are not automatically removed by this change.

All related records for one scheduler cluster occupy one slot. This preserves
the atomicity of existing Lua scripts but does not distribute one scheduler
cluster's metadata evenly across Redis shards.
