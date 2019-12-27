# Customize dfdaemon properties

This topic explains how to customize the dragonfly dfdaemon startup parameters.

## Parameter instructions

The following startup parameters are supported for `dfdaemon`

| Parameter | Description |
| ------------- | ------------- |
| dfget_flags |	dfget properties |
| dfpath | dfget bin path |
| logConfig | Logging properties |
| hijack_https | HijackHTTPS is the list of hosts whose https requests should be hijacked by dfdaemon. The first matched rule will be used |
| localrepo | Temp output dir of dfdaemon, by default `$HOME/.small-dragonfly/dfdaemon/data/` |
| proxies | Proxies is the list of rules for the transparent proxy |
| registry_mirror | Registry mirror settings |
| verbose | Verbose mode. If true, set log level to 'debug'. |

## Examples

Parameters are configured in `/etc/dragonfly/dfdaemon.yml`.
To make it easier for you, you can copy the [template](dfdaemon_config_template.yml) and modify it according to your requirement.

Properties holds all configurable properties of dfdaemon including `dfget` properties. By default, dragonfly configuration files locate at `/etc/dragonfly`. You can create `dfdaemon.yml` for configuring dfdaemon startup params.