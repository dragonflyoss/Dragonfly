# Prometheus Metrics

This doc contains all the metrics that Dragonfly components currently support. Now we only support metrics for Dfdaemon and SuperNode. And we will support dfget metrics in the future. For Dfdaemon and SuperNode, the metrics path is fixed to /metrics. The following metrics are exported.

## Supernode

- dragonfly_supernode_build_info{version, revision, goversion, arch, os} - build and version information of supernode
- dragonfly_supernode_http_requests_total{code, handler, method} - total number of http requests
- dragonfly_supernode_http_request_duration_seconds{code, handler, method} - http request latency in seconds
- dragonfly_supernode_http_request_size_bytes{code, handler, method} - http request size in bytes
- dragonfly_supernode_http_response_size_bytes{code, handler, method} - http response size in bytes

## Dfdaemon

- dragonfly_dfdaemon_build_info{version, revision, goversion, arch, os} - build and version information of dfdaemon

## Dfget

TODO