What Is Dragonfly
---

**Dragonfly is an intelligent P2P based image and file distribution system.**

It aims to resolve issues related to low-efficiency, low-success rate and waste of network bandwidth in file transferring process. Especially in large-scale file distribution scenarios such as application distribution, cache distribution, log distribution, image distribution, etc.

In Alibaba, Dragonfly is invoked 2 Billion times and the data distributed is 3.4PB every month. Dragonfly has become one of the most important pieces of infrastructure at Alibaba.

While container technologies makes devops life easier most of the time, it sure brings a some challenges: the efficiency of image distribution, especially when you have to replicate image distribution on several hosts. Dragonfly works extremely well with both Docker and [PouchContainer](https://github.com/alibaba/pouch) for this scenario. It also is compatible with any other container formats.

It delivers up to 57 times the throughput of native docker and saves up to 99.5% the out bandwidth of registry(*2).

Dragonfly makes it simple and cost-effective to set up, operate, and scale any kind of files/images/data distribution.
