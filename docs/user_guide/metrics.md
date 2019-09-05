# Prometheus Metrics

This doc contains all the metrics that Dragonfly components currently support. Now we only support metrics for Dfdaemon and SuperNode. And we will support dfget metrics in the future. For Dfdaemon and SuperNode, the metrics path is fixed to /metrics. The following metrics are exported.

## Supernode

- dragonfly_supernode_build_info{version, revision, goversion, arch, os} - build and version information of supernode
- dragonfly_supernode_http_requests_total{code, handler, method} - total number of http requests
- dragonfly_supernode_http_request_duration_seconds{code, handler, method} - http request latency in seconds
- dragonfly_supernode_http_request_size_bytes{code, handler, method} - http request size in bytes
- dragonfly_supernode_http_response_size_bytes{code, handler, method} - http response size in bytes
- dragonfly_supernode_peers{peer} - dragonfly peers, the label peer consists of the hostname and ip address of one peer.
- dragonfly_supernode_tasks{cdnstatus} - dragonfly tasks
- dragonfly_supernode_tasks_registered_total{} - total times of registering new tasks. counter type.
- dragonfly_supernode_dfgettasks{callsystem, status} - dragonfly dfget tasks
- dragonfly_supernode_dfgettasks_registered_total{callsystem} - total times of registering new dfgettasks. counter type.
- dragonfly_supernode_dfgettasks_failed_total{callsystem} - total times of failed dfgettasks. counter type.
- dragonfly_supernode_schedule_duration_milliseconds{peer} - duration for task scheduling in milliseconds
- dragonfly_supernode_cdn_trigger_total{} - total times of triggering cdn. counter type.
- dragonfly_supernode_cdn_trigger_total{} - total failed times of triggering cdn. counter type.
- dragonfly_supernode_cdn_cache_hit_total{} - total times of hitting cdn cache. counter type.
- dragonfly_supernode_cdn_download_total{} - total times of cdn downloading. counter type.
- dragonfly_supernode_cdn_download_failed_total{} - total failure times of cdn downloading. counter type.

## Dfdaemon

- dragonfly_dfdaemon_build_info{version, revision, goversion, arch, os} - build and version information of dfdaemon

## Dfget

TODO