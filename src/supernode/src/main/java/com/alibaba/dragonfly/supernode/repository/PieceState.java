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

import java.util.Set;
import java.util.concurrent.ArrayBlockingQueue;
import java.util.concurrent.atomic.AtomicInteger;

import com.alibaba.dragonfly.supernode.common.Constants;
import com.alibaba.dragonfly.supernode.common.util.BeanPoolUtil;
import com.alibaba.dragonfly.supernode.common.util.PowerRateLimiter;

import org.apache.commons.lang3.StringUtils;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class PieceState {
    private static final Logger pieceHitLogger = LoggerFactory
        .getLogger("PieceHitLogger");

    private static final Logger logger = LoggerFactory
        .getLogger(PieceState.class);

    private static final ProgressRepository progressRepo = BeanPoolUtil
        .getBean(ProgressRepository.class);
    private static final PowerRateLimiter rateLimiter = BeanPoolUtil
        .getBean(PowerRateLimiter.class);
    private volatile String superCid;
    private Integer pieceSize;
    private ArrayBlockingQueue<String> pieceContainer = new ArrayBlockingQueue<>(50000);

    private volatile int distributedCount = 0;

    public PieceState(Integer pieceSize) {
        this.pieceSize = pieceSize;
    }

    public boolean offerProducer(String cid) {
        if (StringUtils.isBlank(cid)) {
            return false;
        }
        if (superCid == null && cid.startsWith(Constants.SUPER_NODE_CID)) {
            superCid = cid;
        } else {
            synchronized (pieceContainer) {
                if (!pieceContainer.contains(cid)) {
                    if (pieceContainer.offer(cid)) {
                        distributedCount++;
                    }
                } else {
                    logger.warn("cid:{} is already in queue", cid);
                }
            }
        }
        return true;
    }

    public String popProducer(String taskId, String srcCid, int pieceLen) {
        String dstCid = null;

        Set<String> blackSet = progressRepo.getClientBlackInfo(srcCid);

        AtomicInteger clientErrorCount = progressRepo
            .getClientErrorInfo(srcCid);
        if (clientErrorCount == null
            || clientErrorCount.get() <= Constants.FAIL_COUNT_LIMIT) {
            dstCid = tryGetCid(blackSet);
        } else {
            pieceHitLogger.error("srcCid:{} taskId:{} reach error limit",
                srcCid, taskId);
        }

        if (dstCid == null && superCid != null) {
            if (rateLimiter.tryAcquire(pieceLen > 0 ? pieceLen : pieceSize, distributedCount > 1)) {

                dstCid = superCid;
            }
        }

        return dstCid;
    }

    private String tryGetCid(Set<String> blackSet) {
        AtomicInteger load;
        String cid = null;
        String tmpCid;
        int times = pieceContainer.size();
        boolean needOffer;
        while (times-- > 0) {
            needOffer = true;
            tmpCid = pieceContainer.peek();

            if (tmpCid == null) {
                pieceHitLogger.info("peek element from empty queue");
                break;
            }
            if (progressRepo.getServiceDownInfo(tmpCid)) {
                needOffer = false;
            } else {
                AtomicInteger errorCount = progressRepo
                    .getServiceErrorInfo(tmpCid);
                if (errorCount != null
                    && errorCount.get() >= Constants.ELIMINATION_LIMIT) {
                    needOffer = false;
                } else {
                    if (blackSet == null || !blackSet.contains(tmpCid)) {
                        load = progressRepo.getProducerLoad(tmpCid);
                        if (load != null) {
                            if (load.incrementAndGet() <= Constants.PEER_UP_LIMIT) {
                                cid = tmpCid;
                                break;
                            } else {
                                load.decrementAndGet();
                            }
                        } else {
                            needOffer = false;
                        }
                    }
                }
            }
            synchronized (pieceContainer) {
                if (StringUtils.equals(pieceContainer.peek(), tmpCid)) {
                    if (pieceContainer.remove(tmpCid) && needOffer) {
                        pieceContainer.offer(tmpCid);
                    }
                }
            }

        }

        return cid;
    }

    public int getDistributedCount() {
        return distributedCount;
    }

}
