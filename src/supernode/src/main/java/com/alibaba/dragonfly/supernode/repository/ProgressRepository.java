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

import java.util.BitSet;
import java.util.Map;
import java.util.Set;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.atomic.AtomicInteger;

import org.apache.commons.lang3.StringUtils;
import org.springframework.stereotype.Repository;

@Repository
public class ProgressRepository {

    /**
     * key:cid,value:bitSet
     */
    private static final ConcurrentHashMap<String, BitSet> clientProgress = new ConcurrentHashMap<String, BitSet>();
    /**
     * key:taskId,value:bitSet
     */
    private static final ConcurrentHashMap<String, BitSet> cdnProgress = new ConcurrentHashMap<String, BitSet>();
    /**
     * key:pieceNum@taskId
     */
    private static final ConcurrentHashMap<String, PieceState> pieceProgress
        = new ConcurrentHashMap<String, PieceState>();
    /**
     * key:cid,value:count
     */
    private static final ConcurrentHashMap<String, AtomicInteger> producerLoad
        = new ConcurrentHashMap<String, AtomicInteger>();
    /**
     * cid:pieceNum:dstCid
     */
    private static final ConcurrentHashMap<String, Map<Integer, String>> runningPiece
        = new ConcurrentHashMap<String, Map<Integer, String>>();
    /**
     * dstCid:failCount
     */
    private static final ConcurrentHashMap<String, AtomicInteger> serviceErrorInfo
        = new ConcurrentHashMap<String, AtomicInteger>();
    /**
     * srcCid:failCount
     */
    private static final ConcurrentHashMap<String, AtomicInteger> clientErrorInfo
        = new ConcurrentHashMap<String, AtomicInteger>();
    /**
     * cid:long
     */
    private static final ConcurrentHashMap<String, Long> serviceDownInfo = new ConcurrentHashMap<String, Long>();
    /**
     * srcCid:dstCids
     */
    private static final ConcurrentHashMap<String, Set<String>> clientBlackInfo
        = new ConcurrentHashMap<String, Set<String>>();

    public ConcurrentHashMap<String, AtomicInteger> getServiceErrorInfo() {
        return serviceErrorInfo;
    }

    public ConcurrentHashMap<String, Long> getServiceDownInfo() {
        return serviceDownInfo;
    }

    public ConcurrentHashMap<String, PieceState> getPieceProgress() {
        return pieceProgress;
    }

    /**
     * clientProgress
     */
    public boolean addClientProgress(String cid, BitSet bitSet) {
        if (StringUtils.isBlank(cid)) {
            return false;
        }
        clientProgress.putIfAbsent(cid, bitSet);
        return true;
    }

    public BitSet getClientProgress(String cid) {
        return cid == null ? null : clientProgress.get(cid);
    }

    public boolean removeClientProgress(String cid) {
        return cid != null && clientProgress.remove(cid) != null;
    }

    /**
     * cdnProgress
     */
    public boolean addCdnProgress(String taskId, BitSet bitSet) {
        if (StringUtils.isBlank(taskId)) {
            return false;
        }
        cdnProgress.putIfAbsent(taskId, bitSet);
        return true;
    }

    public BitSet getCdnProgress(String taskId) {
        return taskId == null ? null : cdnProgress.get(taskId);
    }

    public boolean removeCdnProgress(String taskId) {
        return taskId != null && cdnProgress.remove(taskId) != null;
    }

    /**
     * pieceProgress
     */
    private String makePieceProgressKey(String taskId, int pieceNum) {
        if (StringUtils.isBlank(taskId) || pieceNum < 0) {
            return null;
        }
        return pieceNum + "@" + taskId;
    }

    public boolean addPieceProgress(String taskId, int pieceNum,
        PieceState pieceState) {
        String key = makePieceProgressKey(taskId, pieceNum);
        if (key == null) {
            return false;
        }
        pieceProgress.putIfAbsent(key, pieceState);
        return true;
    }

    public PieceState getPieceProgress(String taskId, int pieceNum) {
        String key = makePieceProgressKey(taskId, pieceNum);
        return key == null ? null : pieceProgress.get(key);
    }

    public boolean removePieceProgress(String key) {
        return key != null && pieceProgress.remove(key) != null;
    }

    /**
     * producerLoad
     */
    public boolean addProducerLoad(String cid, AtomicInteger atomicInteger) {
        if (StringUtils.isBlank(cid)) {
            return false;
        }
        producerLoad.putIfAbsent(cid, atomicInteger);
        return true;
    }

    public AtomicInteger getProducerLoad(String cid) {
        return cid == null ? null : producerLoad.get(cid);
    }

    public boolean removeProducerLoad(String cid) {
        return cid != null && producerLoad.remove(cid) != null;
    }

    /**
     * runningPiece
     */
    public boolean addRunningPiece(String cid,
        Map<Integer, String> runningPieces) {
        if (StringUtils.isBlank(cid)) {
            return false;
        }
        runningPiece.putIfAbsent(cid, runningPieces);
        return true;
    }

    public Map<Integer, String> getRunningPiece(String cid) {
        return cid == null ? null : runningPiece.get(cid);
    }

    public boolean removeRunningPiece(String cid) {
        return cid != null && runningPiece.remove(cid) != null;
    }

    /**
     * serviceErrorInfo
     */
    public boolean addServiceErrorInfo(String cid, AtomicInteger atomicInteger) {
        if (StringUtils.isBlank(cid)) {
            return false;
        }
        serviceErrorInfo.putIfAbsent(cid, atomicInteger);
        return true;
    }

    public AtomicInteger getServiceErrorInfo(String cid) {
        return cid == null ? null : serviceErrorInfo.get(cid);
    }

    public boolean removeServiceErrorInfo(String cid) {
        return cid != null && serviceErrorInfo.remove(cid) != null;
    }

    /**
     * clientErrorInfo
     */
    public boolean addClientErrorInfo(String cid, AtomicInteger atomicInteger) {
        if (StringUtils.isBlank(cid)) {
            return false;
        }
        clientErrorInfo.putIfAbsent(cid, atomicInteger);
        return true;
    }

    public AtomicInteger getClientErrorInfo(String cid) {
        return cid == null ? null : clientErrorInfo.get(cid);
    }

    public boolean removeClientErrorInfo(String cid) {
        return cid != null && clientErrorInfo.remove(cid) != null;
    }

    /**
     * serviceDownInfo
     */
    public boolean addServiceDownInfo(String cid, long time) {
        if (StringUtils.isBlank(cid)) {
            return false;
        }
        serviceDownInfo.put(cid, time);
        return true;
    }

    public Boolean getServiceDownInfo(String cid) {
        if (cid == null) {
            return true;
        }
        Long downTime = serviceDownInfo.get(cid);
        if (downTime == null || downTime > 0) {
            return true;
        }
        return false;
    }

    public boolean removeServiceDownInfo(String cid) {
        return cid != null && serviceDownInfo.remove(cid) != null;
    }

    /**
     * clientBlackInfo
     */
    public boolean addClientBlackInfo(String cid, Set<String> blackCids) {
        if (StringUtils.isBlank(cid)) {
            return false;
        }
        clientBlackInfo.putIfAbsent(cid, blackCids);
        return true;
    }

    public Set<String> getClientBlackInfo(String cid) {
        if (cid == null) {
            return null;
        }
        return clientBlackInfo.get(cid);
    }

    public boolean removeClientBlackInfo(String cid) {
        return cid != null && clientBlackInfo.remove(cid) != null;
    }
}
