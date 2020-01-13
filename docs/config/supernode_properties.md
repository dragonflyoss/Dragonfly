# Customize supernode properties

This topic explains how to customize the dragonfly supernode startup parameters.

## Parameter instructions

### The parameters we can configure in supernode are as follows

The following startup parameters are supported for `supernode`

| Parameter | Default | Description |
| ------------- | ------------- | ------------- |
| listenPort | 8002 | listenPort is the port that supernode server listens on |
| downloadPort | 8001 | downloadPort is the port for download files from supernode |
| homeDir | /home/admin/supernode | homeDir is the working directory of supernode |
| schedulerCorePoolSize | 10 | pool size is the core pool size of ScheduledExecutorService(the parameter is aborted) |
| peerUpLimit | 5 | upload limit for a peer to serve download tasks |
| peerDownLimit | 4 |the task upload limit of a peer when dfget starts to play a role of peer |
| eliminationLimit | 5 | if a dfget fails to provide service for other peers up to eliminationLimit, it will be isolated |
| failureCountLimit | 5 | when dfget client fails to finish distribution task up to failureCountLimit, supernode will add it to blacklist|
| systemReservedBandwidth | 20M |  network rate reserved for system |
| maxBandwidth | 200M | network rate that supernode can use |
| enableProfiler | false | profiler sets whether supernode HTTP server setups profiler |
| debug | false | switch daemon log level to DEBUG mode |
| failAccessInterval | 3m0s | fail access interval is the interval time after failed to access the URL |
| gcInitialDelay | 6s | gc initial delay is the delay time from the start to the first GC execution |
| gcMetaInterval | 2m0s | gc meta interval is the interval time to execute the GC meta |
| taskExpireTime | 3m0s | task expire time is the time that a task is treated expired if the task is not accessed within the time |
| peerGCDelay | 3m0s | peer gc delay is the delay time to execute the GC after the peer has reported the offline |
| gcDiskInterval | 15s | GCDiskInterval is the interval time to execute GC disk |
| youngGCThreshold | 100GB | if the available disk space is more than YoungGCThreshold and there is no need to GC disk |
| fullGCThreshold | 5GB | if the available disk space is less than FullGCThreshold and the supernode should gc all task files which are not being used |
| IntervalThreshold | 2h0m0s | IntervalThreshold is the threshold of the interval at which the task file is accessed |

### Some common configurations

We use `--config` to specify the configuration file directory, the default value is `/etc/dragonfly/supernode.yml`
In Dragonfly, supernode provides `listenPort` for dfgets to connect, and dfget downloads document from `downloadPort` instead of `listenPort`.
We can also configure the supernode's IP via `advertiseIP`.

### About gc parameters

In supernode, gc will begin `gcInitialDelay` time after supernode works.
Then supernode will run peer-gc goroutine and task-gc goroutine every  `gcMetaInterval` time.
If a task isn't accessed by dfgets in `taskExpireTime` time, task-gc goroutine will gc this task.
If a peer reports that it's offline and can't provide download service to other peers, peer-gc goroutine will gc this peer after `peerGCDelay` time.

## Examples

To make it easier for you, you can copy the [template](supernode_config_template.yml) and modify it according to your requirement.

When deploying in your physical machine, you can use `--config` to configure where is the configuration file.

```ssh
supernode --config /etc/dragonfly/supernode.yml
```