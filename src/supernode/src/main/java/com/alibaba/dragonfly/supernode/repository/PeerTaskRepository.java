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
package com.alibaba.dragonfly.supernode.repository;

import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.ConcurrentHashMap;

import com.alibaba.dragonfly.supernode.common.domain.PeerTask;
import com.alibaba.dragonfly.supernode.common.enumeration.PeerTaskStatus;

import org.apache.commons.lang3.StringUtils;
import org.springframework.stereotype.Repository;

@Repository
public class PeerTaskRepository {

    private static final ConcurrentHashMap<String, PeerTask> peerTaskMap = new ConcurrentHashMap<String, PeerTask>();

    public boolean add(PeerTask peerTask) {
        String key = makeKey(peerTask.getCid(), peerTask.getTaskId());
        if (key == null) {
            return false;
        }
        peerTaskMap.putIfAbsent(key, peerTask);
        return true;
    }

    public PeerTask get(String cid, String taskId) {
        String key = makeKey(cid, taskId);
        return key == null ? null : peerTaskMap.get(key);
    }

    public List<String> getCidsByTaskId(String taskId) {
        String suffix = "@" + taskId;
        List<String> cids = new ArrayList<String>();
        for (String key : peerTaskMap.keySet()) {
            if (key.endsWith(suffix)) {
                cids.add(key.substring(0, key.lastIndexOf(suffix)));
            }
        }
        return cids;
    }

    public boolean remove(String taskId, String cid) {
        String key = makeKey(cid, taskId);
        return key != null && peerTaskMap.remove(key) != null;
    }

    public boolean updateStatus(String taskId, String cid, PeerTaskStatus status) {
        PeerTask peerTask = get(cid, taskId);
        if (peerTask != null) {
            synchronized (peerTask) {
                if (!peerTask.isSuccess()) {
                    peerTask.setStatus(status);
                    return true;
                }
            }
        }
        return false;
    }

    private String makeKey(String cid, String taskId) {
        if (StringUtils.isBlank(cid) || StringUtils.isBlank(taskId)) {
            return null;
        }
        StringBuilder sb = new StringBuilder();
        sb.append(cid).append("@").append(taskId);
        return sb.toString();
    }

}
