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
package com.alibaba.dragonfly.supernode.service;

import java.util.List;

import com.alibaba.dragonfly.supernode.common.domain.PeerTask;
import com.alibaba.dragonfly.supernode.common.domain.gc.Recyclable;
import com.alibaba.dragonfly.supernode.common.enumeration.PeerTaskStatus;

public interface PeerTaskService extends Recyclable {

    void add(PeerTask peerTask);

    PeerTask get(String cid, String taskId);

    boolean updatePeerTaskStatus(String cid, String taskId,
        PeerTaskStatus status);

    List<String> getCidsByTaskId(String taskId);
}
