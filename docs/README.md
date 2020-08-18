# Dragonfly Document

Dragonfly Document is written, drawn, memorialized representation of all things about Dragonfly. For those who are generally interested in Dragonfly, README.md of this repo is sufficient. While for end users, all details contained in `/docs` is the best guide all should have. And for developers, contents in Developer Guide part is that all need.

Organization of document is as following:

* [Quick Start](#Quick-Start)
* [User Guide](#User-Guide)
    * [Container Runtimes](#Container-Runtimes)
      * [CRIO Mirror](./user_guide/container_runtimes/crio_mirror.md)
      * [Docker Proxy](./user_guide/container_runtimes/docker_proxy.md)
* [CLI Reference](#CLI-Reference)
    * [dfget](./cli_reference/dfget.md)
    * [dfdaemon](./cli_reference/dfdaemon.md)
    * [supernode](./cli_reference/supernode.md)
* [API Reference](#API-Reference)
* [Ecosystem](#Ecosystem)
    * [Kubernetes Integration](./ecosystem/Kubernetes-with-Dragonfly.md)
    * [Harbor Integration](./ecosystem/Harbor-with-Dragonfly.md)
* [Developer Guide](#Developer-Guide)
    * [Design Doc](#Design-Doc)
    * [Test Guide](#Test-Guide)

## Quick Start

[Quick Started](./quick_start) is exactly what you need if you would give Dragonfly a try. This document includes what are the prerequisites, how to install Dragonfly and how to experience Dragonfly's usage.

## User Guide

[User Guide](./user_guide) helps all kinds of guidance end users need to experience Dragonfly. Not only the very brief [Quick Start](./quick_start), but the detailed binary installation and configure illustration. In addition, any concept and function which help users understand Dragonfly better would be included as well.

### Container Runtimes

[Container Runtimes](./user_guide/container_runtimes) contains integration guides for container runtimes, such as docker, cri-o.

## CLI Reference

For almost all users, commandline is the first reference you may need. Document in directory [CLI Reference](./cli_reference) is about command detailed usage of Dragonfly CLI including `dfget`, `dfdaemon` and `supernode`. You can get introductions, synopsis, examples, options about command. Last but not least, Dragonfly can guarantee commandline docs is strongly consistent with Dragonfly CLI's source code. What's more, all commandline docs are auto generated via source code.

## API Reference

Commandline is the easiest way to experience Dragonfly's ability. And API extension will bring more further experience of Dragonfly. Commandline is just one kind of combination usage of API, if you wish to hack or take more advantages of Dragonfly, please see [API Reference](./api_reference). Like command line document, all API docs are auto generated via source code.

## Ecosystem

Ecosystem documents show connections between Dragonfly and popular tool or system in cloud native ecosystem. They guide end users how to experience cloud native systems with Dragonfly, such as other CNCF project Kubernetes and Harbor.

## Developer Guide

Develop Guide helps (potential) developers/contributors to understand the theory inside Dragonfly rather than the interface it exposes. With better understanding of how Dragonfly is designed, developer could learn source code of Dragonfly much easier and know how to debug, test and hack.

### Design Doc

[Design Doc](./design) is content all about design of Dragonfly. It includes all things taken into consideration at the very beginning, the architecture designed for all components in Dragonfly, the interactive workflow between components, all APIs in Dragonfly and some technical things else.

### Test Guide

[Test Guide](./test_guide) is the best reference helping contributors get aware of how to setup testing environment and do it. Currently we can divide test of Dragonfly into four dimensions:

* unit test;
* API integration test;
* CLI integration test;
* node e2e test.

For more details, please refer to [test](./test_guide).

## Conclusion

The folder `/docs` does not contain all the document about Dragonfly. There are still other really helpful documents in other path of this repo, like:

* [FAQ.md](../FAQ.md)
* [CHANGELOG.md](../CHANGELOG.md)
* [ROADMAP.md](../ROADMAP.md)
* others.

If you are searching some document and find no one, please do not hesitate to [file an ISSUE](https://github.com/dragonflyoss/Dragonfly/issues/new/choose) for help. In addition, if you found that there are some incorrect places or typos in document, please help submit a pull request to correct that.
