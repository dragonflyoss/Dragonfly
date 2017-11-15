# Architecture Introduction

## Overview
This guide introduces the system architecture of dragonfly.
<br/>
<br/>
The process distributing general file is as follows:

<div align="center">
<img src="https://github.com/alibaba/Dragonfly/raw/master/docs/images/dfget.png"/>
</div>
<br/>
The cluster manager is also called supernode, which is responsible for CDN and scheduling all peers to transfer blocks between them. dfget is the client of P2P, the so-called 'peer',
which is mainly used to download and share blocks. cluster manager will determine whether the corresponding file exists in the local disk, if not, 
it will be downloaded into cluster manager from file server.<br/>
The process distributing container image is as follows:<br/>

<div align="center">
<img src="https://github.com/alibaba/Dragonfly/raw/master/docs/images/dfget-combine-container.png"/>
</div>
<br/>
Registry is similar to the file server above. dfget proxy is also called df-daemon, which intercepts http-requests from docker pull or docker push,
and determines which requests use dfget to handle.