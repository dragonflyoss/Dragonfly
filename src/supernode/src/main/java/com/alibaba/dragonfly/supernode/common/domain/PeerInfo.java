/*
 * Copyright 1999-2017 Alibaba Group.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package com.alibaba.dragonfly.supernode.common.domain;

import java.util.Map;

public class PeerInfo {

    private String cid;
    private String ip;
    private String hostName;

    public static PeerInfo newInstance(Map<String, String> params) {
        PeerInfo peerInfo = new PeerInfo();
        peerInfo.setCid(params.get("cid"));
        peerInfo.setIp(params.get("ip"));
        peerInfo.setHostName(params.get("hostName"));

        return peerInfo;
    }

    public static PeerInfo newInstance(String cid, String ip, String hostName) {
        PeerInfo peerInfo = new PeerInfo();
        peerInfo.setCid(cid);
        peerInfo.setIp(ip);
        peerInfo.setHostName(hostName);
        return peerInfo;
    }

    public String getCid() {
        return cid;
    }

    public void setCid(String cid) {
        this.cid = cid;
    }

    public String getIp() {
        return ip;
    }

    public void setIp(String ip) {
        this.ip = ip;
    }

    public String getHostName() {
        return hostName;
    }

    public void setHostName(String hostName) {
        this.hostName = hostName;
    }
}
