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

import java.util.ArrayList;
import java.util.List;

import com.alibaba.dragonfly.supernode.common.domain.Task;
import com.alibaba.dragonfly.supernode.common.domain.gc.GcMeta;
import com.alibaba.dragonfly.supernode.common.enumeration.CdnStatus;
import com.alibaba.dragonfly.supernode.common.exception.AuthenticationRequiredException;
import com.alibaba.dragonfly.supernode.common.exception.AuthenticationWaitedException;
import com.alibaba.dragonfly.supernode.common.exception.DataNotFoundException;
import com.alibaba.dragonfly.supernode.common.exception.TaskIdDuplicateException;
import com.alibaba.dragonfly.supernode.common.exception.UrlNotReachableException;
import com.alibaba.dragonfly.supernode.common.util.DigestUtil;
import com.alibaba.dragonfly.supernode.repository.TaskRepository;
import com.alibaba.dragonfly.supernode.service.TaskService;

import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;

@Service("taskService")
public class TaskServiceImpl implements TaskService {
    private static final String key = ">I$pg-~AS~sP'rqu_`Oh&lz#9]\"=;nE%";

    @Autowired
    private TaskRepository taskRepo;

    @Override
    public Task add(Task task)
        throws TaskIdDuplicateException, UrlNotReachableException, AuthenticationRequiredException,
        AuthenticationWaitedException {
        return taskRepo.add(task);
    }

    @Override
    public Task get(String taskId) {
        Task task = taskRepo.get(taskId);
        if (task == null) {
            throw new DataNotFoundException("Task", taskId, "task not found");
        }
        return task;
    }

    @Override
    public String getPieceMd5(String taskId, int pieceNum) {
        Task task = get(taskId);
        return task.getPieceMd5(pieceNum);
    }

    @Override
    public boolean setPieceMd5(String taskId, int pieceNum, String md5) {
        Task task = get(taskId);
        return task.setPieceMd5(pieceNum, md5);
    }

    @Override
    public boolean gc(GcMeta gcMeta) {
        boolean result = false;
        if (gcMeta != null) {
            if (gcMeta.isAll()) {
                taskRepo.remove(gcMeta.getTaskId());
            }
            result = true;
        }
        return result;
    }

    @Override
    public boolean updateCdnStatus(String taskId, CdnStatus cdnStatus) {
        return taskRepo.updateTaskInfo(taskId, null, null, null, cdnStatus);
    }

    @Override
    public boolean updateTaskInfo(String taskId, String md5, Long fileLength, Integer pieceTotal, CdnStatus cdnStatus) {
        return taskRepo.updateTaskInfo(taskId, md5, fileLength, pieceTotal, cdnStatus);
    }

    @Override
    public String createTaskId(String taskUrl, String md5, String bizId) {
        String sign = "";
        if (StringUtils.isNotBlank(md5)) {
            sign = md5;
        } else if (StringUtils.isNotBlank(bizId)) {
            sign = bizId;
        }
        StringBuilder sb = new StringBuilder(key);
        sb.append(taskUrl).append(sign).append(key);
        return DigestUtil.sha256(sb.toString());
    }

    @Override
    public List<String> getFullPieceMd5sByTask(Task task) {
        List<String> pieceMd5s = new ArrayList<>();
        if (task.isSuccess()) {
            Integer pieceTotal = task.getPieceTotal();
            if (pieceTotal != null) {
                for (int i = 0; i < pieceTotal; i++) {
                    pieceMd5s.add(task.getPieceMd5(i));
                }
            }

        }
        return pieceMd5s;
    }

}
