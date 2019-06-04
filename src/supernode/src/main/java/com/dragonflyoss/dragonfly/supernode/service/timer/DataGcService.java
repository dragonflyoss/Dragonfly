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
package com.dragonflyoss.dragonfly.supernode.service.timer;

import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.ConcurrentHashMap;

import com.dragonflyoss.dragonfly.supernode.common.domain.gc.GcMeta;
import com.dragonflyoss.dragonfly.supernode.common.domain.gc.Recyclable;
import com.dragonflyoss.dragonfly.supernode.common.util.BeanPoolUtil;
import com.dragonflyoss.dragonfly.supernode.repository.ProgressRepository;
import com.dragonflyoss.dragonfly.supernode.service.PeerTaskService;
import com.dragonflyoss.dragonfly.supernode.service.cdn.LinkPositiveGc;
import com.dragonflyoss.dragonfly.supernode.service.cdn.util.PathUtil;
import com.dragonflyoss.dragonfly.supernode.service.lock.LockService;

import org.apache.commons.collections.CollectionUtils;
import org.apache.commons.lang3.StringUtils;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Service;

@Service
public class DataGcService {

    private static final Logger logger = LoggerFactory.getLogger("DataGcLogger");
    @Autowired
    private LockService lockService;
    @Autowired
    private PeerTaskService peerTaskService;
    @Autowired
    private LinkPositiveGc linkPositiveGc;
    @Autowired
    private ProgressRepository progressRepo;

    private static final ConcurrentHashMap<String, Long> lruInfoMap = new ConcurrentHashMap<String, Long>();
    private static final int TASK_EXPIRE_TIME = 3 * 60 * 1000;
    private static final int LOCK_EXPIRE_TIME = 3 * 60 * 1000;
    private static int gc_part_count = 0;
    private static int total_client_count = 0;
    private static volatile List<Recyclable> recyclableBeans;

    public void updateAccessTime(String taskId) {
        if (StringUtils.isNotBlank(taskId)) {
            lruInfoMap.put(taskId, System.currentTimeMillis());
        }
    }

    public void updateAccessTimeIfAbsent(String taskId) {
        if (StringUtils.isNotBlank(taskId)) {
            lruInfoMap.putIfAbsent(taskId, System.currentTimeMillis());
        }
    }

    @Scheduled(initialDelay = 6000, fixedDelay = 120000)
    public void dataGc() {
        try {
            logger.info("=================data gc=================");
            if (recyclableBeans == null) {
                recyclableBeans = BeanPoolUtil.getBeans(Recyclable.class);
            }
            int removedTaskCount = 0;
            List<String> taskIds = new ArrayList<String>(lruInfoMap.keySet());
            for (String taskId : taskIds) {
                lockService.lockTaskOnWrite(taskId);
                try {
                    long accessTime = lruInfoMap.get(taskId);
                    if (System.currentTimeMillis() - accessTime > TASK_EXPIRE_TIME) {
                        if (gcTaskData(taskId, true)) {
                            lruInfoMap.remove(taskId);
                            linkPositiveGc.gc(taskId);
                            removedTaskCount++;
                        }
                    }
                } finally {
                    lockService.unlockTaskOnWrite(taskId);
                }
            }
            logger.info("full gc task count:{},remainder count:{}", removedTaskCount, lruInfoMap.size());

            gc_part_count = 0;
            total_client_count = 0;
            taskIds = new ArrayList<>(lruInfoMap.keySet());
            for (String taskId : taskIds) {
                gcTaskData(taskId, false);
            }
            logger.info("gc part client count:{}/{}", gc_part_count, total_client_count);

            lockService.gc(LOCK_EXPIRE_TIME);
        } catch (Exception e) {
            logger.error("E_dataGc", e);
        }
    }

    public boolean gcOneTask(String taskId) {
        logger.info("gc one taskId:{}", taskId);
        lockService.lockTaskOnWrite(taskId);
        if (recyclableBeans == null) {
            recyclableBeans = BeanPoolUtil.getBeans(Recyclable.class);
        }
        try {
            if (gcTaskData(taskId, true)) {
                lruInfoMap.remove(taskId);
                linkPositiveGc.gc(taskId);
                lockService.gcCdnLock(taskId);
            }
            PathUtil.deleteTaskFiles(taskId, true);
            return true;
        } catch (Exception e) {
            logger.error("gc taskId:{} fail", taskId, e);
        } finally {
            lockService.unlockTaskOnWrite(taskId);
        }
        return false;
    }

    private boolean gcTaskData(String taskId, boolean isAll) {
        boolean result = true;
        List<String> cids = peerTaskService.getCidsByTaskId(taskId);
        if (!isAll && CollectionUtils.isNotEmpty(cids)) {
            List<String> gcCids = new ArrayList<String>();
            ConcurrentHashMap<String, Long> serviceDownInfo = progressRepo.getServiceDownInfo();
            long curTime = System.currentTimeMillis();
            for (String cid : cids) {
                Long downTime = serviceDownInfo.get(cid);
                if (downTime != null && downTime > 0 && curTime - downTime > 120000) {
                    gcCids.add(cid);
                }
            }
            total_client_count += cids.size();
            gc_part_count += gcCids.size();
            cids = gcCids;

        }
        GcMeta gcMeta = new GcMeta(taskId, cids, isAll);
        for (Recyclable bean : recyclableBeans) {
            if (!bean.gc(gcMeta)) {
                result = false;
            }
        }
        return result;
    }
}
