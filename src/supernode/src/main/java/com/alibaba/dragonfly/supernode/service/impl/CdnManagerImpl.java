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

import java.util.concurrent.ExecutorService;
import java.util.concurrent.FutureTask;
import java.util.concurrent.SynchronousQueue;
import java.util.concurrent.ThreadPoolExecutor;
import java.util.concurrent.TimeUnit;

import com.alibaba.dragonfly.supernode.common.domain.Task;
import com.alibaba.dragonfly.supernode.common.enumeration.CdnStatus;
import com.alibaba.dragonfly.supernode.common.view.ResultInfo;
import com.alibaba.dragonfly.supernode.service.PeerRegistryService;
import com.alibaba.dragonfly.supernode.service.TaskService;
import com.alibaba.dragonfly.supernode.service.cdn.CdnManager;
import com.alibaba.dragonfly.supernode.service.cdn.Downloader;
import com.alibaba.dragonfly.supernode.service.lock.LockConstants;
import com.alibaba.dragonfly.supernode.service.lock.LockService;
import com.alibaba.fastjson.JSON;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;

@Service
public class CdnManagerImpl implements CdnManager {
    private static final Logger logger = LoggerFactory.getLogger(CdnManagerImpl.class);
    @Autowired
    private TaskService taskService;
    @Autowired
    private LockService lockService;
    @Autowired
    private PeerRegistryService peerRegistryService;

    private final ExecutorService cdnExecutor =
        new ThreadPoolExecutor(20, 100, 60L, TimeUnit.SECONDS, new SynchronousQueue<Runnable>(true)); // 执行下载任务的线程池

    /**
     * @param taskId
     * @return
     */
    @Override
    public boolean triggerCdnSyncAction(String taskId) {
        String lockName = lockService.getLockName(LockConstants.CDN_TRIGGER_LOCK, taskId);
        if (lockService.isAccessWindow(lockName, 120 * 1000) && lockService.tryLock(lockName)) {
            try {
                Task task = taskService.get(taskId);
                synchronized (task) {
                    if (task.isFrozen()) {
                        if (task.isWait()) {
                            ResultInfo resultInfo = peerRegistryService.registryCdnNode(task);
                            if (!resultInfo.successCode()) {
                                logger.error("do trigger cdn fail for task:{}", JSON.toJSONString(task));
                                return false;
                            }
                        }
                        taskService.updateCdnStatus(taskId, CdnStatus.RUNNING);
                        new Thread(this.new Trigger(task)).start();
                        logger.info("do trigger cdn start for taskId:{},httpLen:{}", taskId, task.getHttpFileLen());
                    }
                }
            } finally {
                lockService.unlock(lockName);
            }
        }
        return true;
    }

    private class Trigger implements Runnable {
        private Task task;

        private Trigger(Task task) {
            this.task = task;
        }

        @Override
        public void run() {
            String taskId = task.getTaskId();
            try {
                Downloader downloader = new Downloader(task);
                FutureTask<Boolean> future = new FutureTask<Boolean>(downloader);
                downloader.setFuture(future);
                cdnExecutor.submit(future);
                logger.info("do trigger cdn success for taskId:{}", taskId);
                return;

            } catch (Exception e) {
                logger.error("E_trigger_cdn for task:{}", JSON.toJSONString(task), e);
            }
            taskService.updateCdnStatus(taskId, CdnStatus.FAIL);
        }

    }
}
