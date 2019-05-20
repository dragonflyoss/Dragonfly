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
package com.dragonflyoss.dragonfly.supernode.service;

import com.dragonflyoss.dragonfly.supernode.common.domain.PeerInfo;
import com.dragonflyoss.dragonfly.supernode.common.domain.Task;
import com.dragonflyoss.dragonfly.supernode.common.exception.ValidateException;
import com.dragonflyoss.dragonfly.supernode.common.view.ResultInfo;

public interface PeerRegistryService {

    /**
     * register client node
     *
     * @param sourceUrl
     * @param taskUrl
     * @param md5
     * @param bizId
     * @param port
     * @param peerInfo
     * @param path
     * @param version
     * @return
     * @throws ValidateException
     */
    ResultInfo registryTask(String sourceUrl, String taskUrl, String md5, String bizId, String port, PeerInfo peerInfo,
        String path, String version, String superNodeIp, String[] headers, boolean dfdaemon) throws ValidateException;

    /**
     * register cdn node
     *
     * @param task
     * @return
     */
    ResultInfo registryCdnNode(Task task);
}
