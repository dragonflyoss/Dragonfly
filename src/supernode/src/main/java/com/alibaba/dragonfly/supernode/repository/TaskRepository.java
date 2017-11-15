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

import java.util.concurrent.ConcurrentHashMap;

import com.alibaba.dragonfly.supernode.common.Constants;
import com.alibaba.dragonfly.supernode.common.domain.Task;
import com.alibaba.dragonfly.supernode.common.enumeration.CdnStatus;
import com.alibaba.dragonfly.supernode.common.exception.AuthenticationRequiredException;
import com.alibaba.dragonfly.supernode.common.exception.AuthenticationWaitedException;
import com.alibaba.dragonfly.supernode.common.exception.TaskIdDuplicateException;
import com.alibaba.dragonfly.supernode.common.exception.UrlNotReachableException;
import com.alibaba.dragonfly.supernode.common.util.HttpClientUtil;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Repository;

@Repository
public class TaskRepository {

    private static final Logger logger = LoggerFactory.getLogger(TaskRepository.class);

    private final static ConcurrentHashMap<String, Task> taskMap = new ConcurrentHashMap<>();

    public Task add(Task task)
        throws TaskIdDuplicateException, UrlNotReachableException, AuthenticationRequiredException,
        AuthenticationWaitedException {
        String tmpId = task.getTaskId();
        Task existTask = taskMap.putIfAbsent(tmpId, task);
        if (existTask != null && !existTask.equals(task)) {
            throw new TaskIdDuplicateException(tmpId, "taskId conflict");
        }
        existTask = existTask != null ? existTask : task;

        if (existTask.isNotReachable()) {
            throw new UrlNotReachableException();
        }

        synchronized (existTask) {
            if (existTask.isNotReachable()) {
                throw new UrlNotReachableException();
            }
            if (existTask.getHttpFileLen() == null) {
                existTask.addAuthIps(task.getCurIp());
                if (existTask.inAuthIps(task.getCurIp())) {
                    try {
                        existTask.setHttpFileLen(
                            HttpClientUtil
                                .getContentLength(existTask.getSourceUrl(), task.getHeaders(), task.isDfdaemon()));
                        if (task.getHeaders() != null) {
                            existTask.setHeaders(task.getHeaders());
                        }
                        if (existTask.getPieceSize() == null) {
                            existTask.setPieceSize(computePieceSize(existTask.getHttpFileLen()));
                        }
                        logger.info("get file length:{} from http client about taskId:{}", existTask.getHttpFileLen(),
                            existTask.getTaskId());
                    } catch (UrlNotReachableException e) {
                        existTask.setNotReachable(true);
                        throw e;
                    } catch (AuthenticationRequiredException e) {
                        throw e;
                    } catch (Exception e) {
                        logger.error("E_add_task_getContentLength", e);
                    }
                } else {
                    throw new AuthenticationWaitedException();
                }
            }
        }
        return existTask;
    }

    private Integer computePieceSize(Long httpFileLen) {
        if (httpFileLen == null || httpFileLen <= 200 * 1024 * 1024L) {
            return Constants.DEFAULT_PIECE_SIZE;
        } else {
            long gapCount = httpFileLen / (100 * 1024 * 1024L);
            long tmpSize = (gapCount - 2) * 1024 * 1024 + Constants.DEFAULT_PIECE_SIZE;
            return tmpSize > Constants.PIECE_SIZE_LIMIT ? Constants.PIECE_SIZE_LIMIT : (int)tmpSize;
        }
    }

    public Task get(String taskId) {
        return taskId == null ? null : taskMap.get(taskId);
    }

    public boolean remove(String taskId) {
        return taskId != null && taskMap.remove(taskId) != null;
    }

    public boolean updateTaskInfo(String taskId, String md5, Long fileLength, Integer pieceTotal, CdnStatus cdnStatus) {
        Task task = get(taskId);
        if (task != null) {
            synchronized (task) {
                if (!task.isSuccess() || (cdnStatus != null && cdnStatus.equals(CdnStatus.SUCCESS))) {
                    if (cdnStatus.equals(CdnStatus.SUCCESS)) {
                        if (fileLength != null) {
                            task.setFileLength(fileLength);
                        }
                        if (md5 != null) {
                            task.setRealMd5(md5);
                        }
                        if (pieceTotal != null) {
                            task.setPieceTotal(pieceTotal);
                        }
                    }
                    task.setCdnStatus(cdnStatus);
                }
                return true;
            }
        }
        return false;
    }
}
