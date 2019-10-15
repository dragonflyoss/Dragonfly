# Customize dfdaemon properties

This topic explains how to customize the dragonfly dfdaemon startup parameters.

## Parameter instructions

The following startup parameters are supported for `dfdaemon`

| Parameter | Description |
| ------------- | ------------- |
| dfget_flags |	dfget properties |
| dfpath | dfget path |
| hijack_https | HijackHTTPS is the list of hosts whose https requests should be hijacked by dfdaemon. The first matched rule will be used |
| localrepo | Temp output dir of dfdaemon, by default `$HOME/.small-dragonfly/dfdaemon/data/` |
| maxprocs| The maximum number of CPUs that the dfdaemon can use |
| proxies | Proxies is the list of rules for the transparent proxy |
| ratelimit | Net speed limit,format:xxxM/K |
| registry_mirror | Registry mirror settings |
| supernodes | Specify the addresses(host:port) of supernodes, it is just to be compatible with previous versions |
| verbose | Open detail info switch |

## Examples

Parameters are configured in `/etc/dragonfly/dfdaemon.yml`.
To make it easier for you, you can copy the [template](dfdaemon_config_template.yml) and modify it according to your requirement.

Properties holds all configurable properties of dfdaemon including `dfget` properties. By default, dragonfly configuration files locate at `/etc/dragonfly`. You can create `dfdaemon.yml` for configuring dfdaemon startup params.