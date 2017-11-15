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
package com.alibaba.dragonfly.supernode.service.impl;

import java.util.List;

import com.alibaba.dragonfly.supernode.common.domain.PeerTask;
import com.alibaba.dragonfly.supernode.common.domain.gc.GcMeta;
import com.alibaba.dragonfly.supernode.common.enumeration.PeerTaskStatus;
import com.alibaba.dragonfly.supernode.common.exception.DataNotFoundException;
import com.alibaba.dragonfly.supernode.repository.PeerTaskRepository;
import com.alibaba.dragonfly.supernode.service.PeerTaskService;

import org.apache.commons.collections.CollectionUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;

@Service("peerTaskService")
public class PeerTaskServiceImpl implements PeerTaskService {
    @Autowired
    private PeerTaskRepository peerTaskRepo;

    @Override
    public void add(PeerTask peerTask) {
        peerTaskRepo.add(peerTask);
    }

    @Override
    public PeerTask get(String cid, String taskId) {
        PeerTask peerTask = peerTaskRepo.get(cid, taskId);
        if (peerTask == null) {
            throw new DataNotFoundException("PeerTask", cid + "@" + taskId,
                "peerTask not found");
        }
        return peerTask;
    }

    @Override
    public boolean updatePeerTaskStatus(String cid, String taskId,
        PeerTaskStatus status) {
        return peerTaskRepo.updateStatus(taskId, cid, status);
    }

    @Override
    public List<String> getCidsByTaskId(String taskId) {
        return peerTaskRepo.getCidsByTaskId(taskId);
    }

    @Override
    public boolean gc(GcMeta gcMeta) {
        boolean result = false;
        if (gcMeta != null) {
            List<String> cids = gcMeta.getCids();
            String taskId = gcMeta.getTaskId();
            if (CollectionUtils.isNotEmpty(cids) && taskId != null) {
                for (String cid : cids) {
                    peerTaskRepo.remove(taskId, cid);
                }
            }
            result = true;
        }
        return result;
    }
}
