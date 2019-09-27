# Prometheus Metrics

This doc contains all the metrics that Dragonfly components currently support. Now we support metrics for Dfdaemon, SuperNode and Dfget. For Dfdaemon and SuperNode, the metrics path is fixed to /metrics. The following metrics are exported.

## Supernode

Name                                                   | Labels                                 | Type      | Description
:----------------------------------------------------- | :--------------------------------------|:--------- | :----------
dragonfly_supernode_build_info                         | version, revision, goversion, arch, os | gauge     | Build and version information of supernode.
dragonfly_supernode_http_requests_total                | code, handler                          | counter   | Total number of http requests.
dragonfly_supernode_http_request_duration_seconds      | handler                                | histogram | HTTP request latency in seconds.
dragonfly_supernode_http_request_size_bytes            | handler                                | histogram | HTTP request size in bytes.
dragonfly_supernode_http_response_size_bytes           | handler                                | histogram | HTTP response size in bytes.
dragonfly_supernode_peers                              | peer                                   | gauge     | Dragonfly peers, the label peer consists of the hostname and ip address of one peer.
dragonfly_supernode_tasks                              | cdnstatus                              | gauge     | Dragonfly tasks.
dragonfly_supernode_tasks_registered_total             |                                        | counter   | Total times of registering new tasks.
dragonfly_supernode_dfgettasks                         | callsystem, status                     | gauge     | Dragonfly dfget tasks.
dragonfly_supernode_dfgettasks_registered_total        | callsystem                             | counter   | Total times of registering new dfgettasks.
dragonfly_supernode_dfgettasks_failed_total            | callsystem                             | counter   | Total times of failed dfgettasks.
dragonfly_supernode_schedule_duration_milliseconds     | peer                                   | histogram | Duration for task scheduling in milliseconds.
dragonfly_supernode_cdn_trigger_total                  |                                        | counter   | Total times of triggering cdn.
dragonfly_supernode_cdn_trigger_failed_total           |                                        | counter   | Total failed times of triggering cdn.
dragonfly_supernode_cdn_cache_hit_total                |                                        | counter   | Total times of hitting cdn cache.
dragonfly_supernode_cdn_download_total                 |                                        | counter   | Total times of cdn downloading.
dragonfly_supernode_cdn_download_failed_total          |                                        | counter   | Total failure times of cdn downloading.
dragonfly_supernode_pieces_downloaded_size_bytes_total |                                        | counter   | Total size of pieces downloaded from supernode in bytes.
dragonfly_supernode_gc_peers_total                     |                                        | counter   | Total number of peers that have been garbage collected.
dragonfly_supernode_gc_tasks_total                     |                                        | counter   | Total number of tasks that have been garbage collected.
dragonfly_supernode_gc_disks_total                     |                                        | counter   | Total number of garbage collecting the task data in disks.
dragonfly_supernode_last_gc_disks_timestamp_seconds    |                                        | gauge     | Timestamp of the last disk gc.

## Dfdaemon

Name                          | Labels                                 | Type  | Description
:---------------------------- | :------------------------------------- | :---- | :----------
dragonfly_dfdaemon_build_info | version, revision, goversion, arch, os | gauge | Build and version information of dfdaemon.

## Dfget

Name                                      | Labels                   | Type      | Description
:---------------------------------------- | :----------------------- | :-------- | :----------
dragonfly_dfget_download_duration_seconds | callsystem, peer         | histogram | Dfget download duration in seconds.
dragonfly_dfget_download_size_bytes_total | callsystem, peer         | counter   | Total size of files downloaded by dfget in bytes.
dragonfly_dfget_download_total            | callsystem, peer         | counter   | Total times of dfget downloading.
dragonfly_dfget_download_failed_total     | callsystem, peer, reason | counter   | Total times of failed dfget downloading.
