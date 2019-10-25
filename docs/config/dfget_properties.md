# Customize dfget properties

This topic explains how to customize the dragonfly dfget startup parameters.

## Parameter instructions

The following startup parameters are supported for `dfget`

| Parameter | Description |
| ------------- | ------------- |
| nodes	| Nodes specify supernodes with format host:port=weight where the host is necessary, the port(default: 8002) and the weight(default:1) are optional. |
| localLimit | LocalLimit rate limit about a single download task,format: G(B)/g/M(B)/m/K(B)/k/B. |
| minRate | Minimal rate about a single download task,format: G(B)/g/M(B)/m/K(B)/k/B. |
| totalLimit | TotalLimit rate limit about the whole host includes download and upload, format: G(B)/g/M(B)/m/K(B)/k/B |
| clientQueueSize | ClientQueueSize is the size of client queue, which controls the number of pieces that can be processed simultaneously. It is only useful when the Pattern equals "source". The default value is 6 |

## Examples

Parameters are configured in `/etc/dragonfly/dfget.yml`.
To make it easier for you, you can copy the [template](dfget_config_template.yml) and modify it according to your requirement.

By default, dragonfly config files locate at `/etc/dragonfly`. You can create `dfget.yml` in the path if you want to install dfget in physical machine.
