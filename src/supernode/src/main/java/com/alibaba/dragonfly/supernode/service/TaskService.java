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

import com.alibaba.dragonfly.supernode.common.domain.Task;
import com.alibaba.dragonfly.supernode.common.domain.gc.Recyclable;
import com.alibaba.dragonfly.supernode.common.enumeration.CdnStatus;
import com.alibaba.dragonfly.supernode.common.exception.AuthenticationRequiredException;
import com.alibaba.dragonfly.supernode.common.exception.AuthenticationWaitedException;
import com.alibaba.dragonfly.supernode.common.exception.TaskIdDuplicateException;
import com.alibaba.dragonfly.supernode.common.exception.UrlNotReachableException;

public interface TaskService extends Recyclable {

    /**
     * @param task
     */
    Task add(Task task) throws TaskIdDuplicateException, UrlNotReachableException, AuthenticationRequiredException,
        AuthenticationWaitedException;

    /**
     * @param taskId
     * @return
     */
    Task get(String taskId);

    /**
     * @param taskId
     * @param pieceNum
     * @return
     */
    String getPieceMd5(String taskId, int pieceNum);

    /**
     * @param taskId
     * @param pieceNum
     * @param md5
     */
    boolean setPieceMd5(String taskId, int pieceNum, String md5);

    /**
     * @param taskId
     * @param pieceTotal
     * @return
     */
    boolean updateTaskInfo(String taskId, String md5, Long fileLength, Integer pieceTotal, CdnStatus cdnStatus);

    /**
     * @param taskId
     * @param cdnStatus
     * @return
     */
    boolean updateCdnStatus(String taskId, CdnStatus cdnStatus);

    /**
     * @param taskUrl
     * @param md5
     * @param bizId
     * @return
     */
    String createTaskId(String taskUrl, String md5, String bizId);

    /**
     * @param task
     * @return
     */
    List<String> getFullPieceMd5sByTask(Task task);

}
