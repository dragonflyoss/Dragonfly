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
import java.util.BitSet;
import java.util.Collections;
import java.util.HashMap;
import java.util.HashSet;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.TreeMap;
import java.util.concurrent.atomic.AtomicInteger;

import com.alibaba.dragonfly.supernode.common.Constants;
import com.alibaba.dragonfly.supernode.common.domain.PeerInfo;
import com.alibaba.dragonfly.supernode.common.domain.PeerTask;
import com.alibaba.dragonfly.supernode.common.domain.Task;
import com.alibaba.dragonfly.supernode.common.domain.gc.GcMeta;
import com.alibaba.dragonfly.supernode.common.enumeration.PeerPieceStatus;
import com.alibaba.dragonfly.supernode.common.util.RangeParseUtil;
import com.alibaba.dragonfly.supernode.common.view.PieceTask;
import com.alibaba.dragonfly.supernode.common.view.ResultCode;
import com.alibaba.dragonfly.supernode.common.view.ResultInfo;
import com.alibaba.dragonfly.supernode.repository.PieceState;
import com.alibaba.dragonfly.supernode.repository.ProgressRepository;
import com.alibaba.dragonfly.supernode.service.PeerService;
import com.alibaba.dragonfly.supernode.service.PeerTaskService;
import com.alibaba.dragonfly.supernode.service.TaskService;
import com.alibaba.dragonfly.supernode.service.scheduler.ProgressService;

import org.apache.commons.collections.CollectionUtils;
import org.apache.commons.lang3.StringUtils;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;

@Service("progressService")
public class ProgressServiceImpl implements ProgressService {
    private static final Logger logger = LoggerFactory.getLogger(ProgressServiceImpl.class);

    @Autowired
    private ProgressRepository progressRepo;
    @Autowired
    private TaskService taskService;
    @Autowired
    private PeerService peerService;
    @Autowired
    private PeerTaskService peerTaskService;

    @Override
    public ResultInfo initProgress(String taskId, String cid) {
        if (StringUtils.isBlank(taskId) || StringUtils.isBlank(cid)) {
            String msg = new StringBuilder().append("init progress fail for taskId:").append(taskId).append(",cid:")
                .append(cid).toString();
            logger.error(msg);
            return new ResultInfo(ResultCode.PARAM_ERROR, msg, null);
        }
        boolean isSuperNode = cid.startsWith(Constants.SUPER_NODE_CID);
        if (!isSuperNode) {
            progressRepo.addClientProgress(cid, new BitSet());
            progressRepo.addRunningPiece(cid, new HashMap<Integer, String>());
            progressRepo.addProducerLoad(cid, new AtomicInteger(0));
            progressRepo.addServiceDownInfo(cid, 0L);
        } else {
            progressRepo.addCdnProgress(taskId, new BitSet());
        }
        return new ResultInfo(ResultCode.SUCCESS);
    }

    @Override
    public ResultInfo updateProgress(String taskId, String srcCid, String dstCid, int pieceNum,
        PeerPieceStatus pieceStatus) {
        String msg = null;
        if (StringUtils.isBlank(srcCid)) {
            msg = new StringBuilder().append("src cid is empty for taskId:").append(taskId).toString();
            logger.error(msg);
            return new ResultInfo(ResultCode.UNKNOWN_ERROR, msg, null);
        }

        BitSet pieceBitSet = getBitSetByCid(taskId, srcCid);
        if (pieceBitSet == null) {
            msg = new StringBuilder().append("peer progress not found for taskId:").append(taskId).append(",cid:")
                .append(srcCid).toString();
            logger.error(msg);
            return new ResultInfo(ResultCode.UNKNOWN_ERROR, msg, null);
        }
        if (PeerPieceStatus.SUCCESS.equals(pieceStatus)) {
            ResultInfo tmpResult = processPieceSuc(taskId, srcCid, pieceNum);
            if (!tmpResult.successCode()) {
                return tmpResult;
            }
        }
        if (PeerPieceStatus.SEMISUC.equals(pieceStatus)) {
            pieceStatus = PeerPieceStatus.SUCCESS;
        }

        synchronized (pieceBitSet) {
            if (pieceBitSet.get(pieceNum * 8 + pieceStatus.getStatus())) {
                return new ResultInfo(ResultCode.SUCCESS);
            }

            boolean result = updateProgressBitSet(srcCid, dstCid, pieceBitSet, pieceNum, pieceStatus);
            if (!result) {
                return new ResultInfo(ResultCode.SUCCESS);
            }
            if (PeerPieceStatus.SUCCESS.equals(pieceStatus)) {
                processPeerSucInfo(srcCid, dstCid);
            } else if (PeerPieceStatus.FAIL.equals(pieceStatus)) {
                processPeerFailInfo(srcCid, dstCid);
            }
        }
        if (StringUtils.isNotBlank(dstCid)) {
            AtomicInteger load = progressRepo.getProducerLoad(dstCid);
            if (load != null) {
                int loadValue = load.decrementAndGet();
                if (loadValue < 0) {
                    logger.warn("client load maybe illegal,taskId:{},cid:{},pieceNum:{},load:{}",
                        taskId, dstCid, pieceNum, loadValue);
                    load.incrementAndGet();
                }
            }
        }
        return new ResultInfo(ResultCode.SUCCESS);
    }

    private boolean needDoPeerInfo(String srcCid, String dstCid) {
        if (StringUtils.isBlank(srcCid) || srcCid.startsWith(Constants.SUPER_NODE_CID)) {
            return false;
        }
        if (StringUtils.isBlank(dstCid) || dstCid.startsWith(Constants.SUPER_NODE_CID)) {
            return false;
        }
        return true;
    }

    private void processPeerSucInfo(String srcCid, String dstCid) {
        if (!needDoPeerInfo(srcCid, dstCid)) {
            return;
        }
        AtomicInteger clientErrorCount = progressRepo.getClientErrorInfo(srcCid);
        if (clientErrorCount != null) {
            clientErrorCount.set(0);
        }
        AtomicInteger serviceErrorCount = progressRepo.getServiceErrorInfo(dstCid);
        if (serviceErrorCount != null) {
            serviceErrorCount.set(0);
        }
    }

    @Override
    public ResultInfo processPieceSuc(String taskId, String srcCid, int pieceNum) {
        PeerTask peerTask = peerTaskService.get(srcCid, taskId);
        if (peerTask.getPort() > 0) {
            PieceState existedPieceState = progressRepo.getPieceProgress(taskId, pieceNum);
            if (existedPieceState == null) {
                progressRepo.addPieceProgress(taskId, pieceNum, new PieceState(peerTask.getPieceSize()));
                existedPieceState = progressRepo.getPieceProgress(taskId, pieceNum);
            }
            if (!existedPieceState.offerProducer(srcCid)) {
                String msg =
                    new StringBuilder().append("offer producer fail for srcCid:").append(srcCid).toString();
                logger.error(msg);
                return new ResultInfo(ResultCode.SYSTEM_ERROR, msg, null);
            }
        }
        return new ResultInfo(ResultCode.SUCCESS);
    }

    private void processPeerFailInfo(String srcCid, String dstCid) {
        if (!needDoPeerInfo(srcCid, dstCid)) {
            return;
        }
        Set<String> blackList = progressRepo.getClientBlackInfo(srcCid);
        if (blackList == null) {
            blackList = new HashSet<String>();
            progressRepo.addClientBlackInfo(srcCid, blackList);
        }
        if (!blackList.contains(dstCid)) {
            blackList.add(dstCid);
            AtomicInteger clientErrorCount = progressRepo.getClientErrorInfo(srcCid);
            if (clientErrorCount == null) {
                progressRepo.addClientErrorInfo(srcCid, new AtomicInteger(0));
                clientErrorCount = progressRepo.getClientErrorInfo(srcCid);
            }
            clientErrorCount.incrementAndGet();
            AtomicInteger errorCount = progressRepo.getServiceErrorInfo(dstCid);
            if (errorCount == null) {
                progressRepo.addServiceErrorInfo(dstCid, new AtomicInteger(0));
                errorCount = progressRepo.getServiceErrorInfo(dstCid);
            }
            errorCount.incrementAndGet();
        }
    }

    private BitSet getBitSetByCid(String taskId, String srcCid) {
        BitSet pieceBitSet = null;
        boolean isSuperNode = srcCid.startsWith(Constants.SUPER_NODE_CID);
        if (!isSuperNode) {
            pieceBitSet = progressRepo.getClientProgress(srcCid);
        } else {
            pieceBitSet = progressRepo.getCdnProgress(taskId);
        }
        return pieceBitSet;
    }

    /**
     * @param taskId
     * @param cid
     * @return
     */
    @Override
    public ResultInfo parseAvaliablePeerTasks(String taskId, String cid) {
        String msg = null;
        if (StringUtils.isBlank(cid) || StringUtils.isBlank(taskId)) {
            msg = new StringBuilder().append("param is illegal for taskId:").append(taskId).append(",cid:").append(cid)
                .toString();
            logger.error(msg);
            return new ResultInfo(ResultCode.UNKNOWN_ERROR, msg, null);
        }
        Task task = taskService.get(taskId);

        if (task.isFail()) {
            return new ResultInfo(ResultCode.SUPER_FAIL, "cdn status is fail", null);
        } else if (task.isWait()) {
            return new ResultInfo(ResultCode.PEER_WAIT, "cdn status is wait", null);
        }

        BitSet clientProgressValue = progressRepo.getClientProgress(cid);
        if (clientProgressValue == null) {
            msg = new StringBuilder().append("cid progress not found for taskId:").append(taskId).append(",cid:")
                .append(cid).toString();
            logger.error(msg);
            return new ResultInfo(ResultCode.UNKNOWN_ERROR, msg, null);
        }
        synchronized (clientProgressValue) {
            BitSet clonedClientProgressValue = (BitSet)clientProgressValue.clone();

            BitSet cdnProgressValue = progressRepo.getCdnProgress(taskId);
            if (cdnProgressValue == null) {
                msg = new StringBuilder().append("cdn progress not found for taskId:").append(taskId).toString();
                logger.error(msg);
                return new ResultInfo(ResultCode.UNKNOWN_ERROR, msg, null);
            }
            BitSet clonedCdnProgressValue = null;
            synchronized (cdnProgressValue) {
                clonedCdnProgressValue = (BitSet)progressRepo.getCdnProgress(taskId).clone();
            }

            clonedClientProgressValue.and(clonedCdnProgressValue);

            int pieceTotal = task.getPieceTotal();
            boolean cdnSuccess = task.isSuccess();
            int clientSucCount = clonedClientProgressValue.cardinality();
            if (cdnSuccess && clientSucCount == pieceTotal) {
                Map<String, Object> finishInfo = new HashMap<String, Object>();
                finishInfo.put("md5", task.getRealMd5());
                finishInfo.put("fileLength", task.getFileLength());
                return new ResultInfo(ResultCode.PEER_FINISH, finishInfo);
            }
            clonedCdnProgressValue.andNot(clonedClientProgressValue);

            List<Integer> availablePieces = new ArrayList<Integer>();
            for (int i = clonedCdnProgressValue.nextSetBit(0); i >= 0; i = clonedCdnProgressValue.nextSetBit(i + 1)) {
                int statusNum = i % 8;
                if (statusNum == PeerPieceStatus.SUCCESS.getStatus()) {
                    availablePieces.add(i / 8);
                } else if (statusNum == PeerPieceStatus.FAIL.getStatus()) {
                    logger.error("taskId:{} cdn piece fail for num:{}", taskId, i / 8);
                    return new ResultInfo(ResultCode.SUPER_FAIL, "cdn piece fail", null);
                }
            }

            if (availablePieces.isEmpty()) {
                StringBuilder sb = new StringBuilder();
                sb.append("client sucCount:").append(clientSucCount).append(",cdn status:")
                    .append(task.getCdnStatus().toString()).append(",cdn sucCount:")
                    .append(cdnProgressValue.cardinality());
                return new ResultInfo(ResultCode.PEER_WAIT, sb.toString(), null);
            }
            List<PieceTask> pieceTasks = new ArrayList<PieceTask>();
            Map<Integer, String> dstCidMap = progressRepo.getRunningPiece(cid);
            if (dstCidMap == null) {
                msg = new StringBuilder().append("running piece not found for taskId:").append(taskId).append(",cid:")
                    .append(cid).toString();
                logger.error(msg);
                return new ResultInfo(ResultCode.UNKNOWN_ERROR, msg, null);
            }
            List<Integer> invalidRunningPieces = new ArrayList<Integer>();
            for (Integer pieceNum : dstCidMap.keySet()) {
                if (!fillPieceTasks(pieceTasks, task, dstCidMap.get(pieceNum), pieceNum)) {
                    invalidRunningPieces.add(pieceNum);
                }
            }
            if (CollectionUtils.isNotEmpty(invalidRunningPieces)) {
                for (Integer tmpPieceNum : invalidRunningPieces) {
                    dstCidMap.remove(tmpPieceNum);
                }
            }
            int runningCount = pieceTasks.size();
            Map<Integer, List<Integer>> finishedCountMap = new TreeMap<Integer, List<Integer>>();
            if (runningCount < Constants.PEER_DOWN_LIMIT) {
                Set<Integer> runningPieces = dstCidMap.keySet();
                availablePieces.removeAll(runningPieces);

                for (Integer pieceNum : availablePieces) {
                    PieceState pieceState = progressRepo.getPieceProgress(taskId, pieceNum);
                    if (pieceState == null) {
                        msg = new StringBuilder().append("pieceState not found for taskId:").append(taskId)
                            .append(",pieceNum:").append(pieceNum).toString();
                        logger.error(msg);
                        return new ResultInfo(ResultCode.UNKNOWN_ERROR, msg, null);
                    }
                    Integer distributedCount = pieceState.getDistributedCount();
                    List<Integer> pieceNumList = finishedCountMap.get(distributedCount);
                    if (pieceNumList == null) {
                        pieceNumList = new ArrayList<Integer>();
                        finishedCountMap.put(distributedCount, pieceNumList);
                    }
                    pieceNumList.add(pieceNum);
                }
                if (!finishedCountMap.isEmpty()) {
                    for (List<Integer> pieceNums : finishedCountMap.values()) {
                        parseNearNums(pieceNums, runningPieces);
                        for (Integer tmpPieceNum : pieceNums) {
                            PieceState pieceState = progressRepo.getPieceProgress(taskId, tmpPieceNum);
                            if (pieceState == null) {
                                return new ResultInfo(ResultCode.UNKNOWN_ERROR,
                                    "piece state not found for pieceNum:" + tmpPieceNum, null);
                            }
                            String dstCid = pieceState.popProducer(taskId, cid, parsePieceLen(taskId, tmpPieceNum));

                            if (dstCid == null) {
                                continue;
                            }

                            if (!fillPieceTasks(pieceTasks, task, dstCid, tmpPieceNum)) {
                                continue;
                            }

                            updateProgressBitSet(cid, dstCid, clientProgressValue, tmpPieceNum,
                                PeerPieceStatus.RUNNING);
                            if (++runningCount >= Constants.PEER_DOWN_LIMIT) {
                                break;
                            }
                        }
                        if (runningCount >= Constants.PEER_DOWN_LIMIT) {
                            break;
                        }
                    }
                }
            }

            if (runningCount > 0) {
                ResultInfo result = new ResultInfo(ResultCode.PEER_CONTINUE, pieceTasks);
                return result;
            }
        }
        return new ResultInfo(ResultCode.PEER_WAIT, "piece resource lack", null);
    }

    private boolean updateProgressBitSet(String cid, String dstCid, BitSet pieceBitSet, int pieceNum,
        PeerPieceStatus peerStatus) {
        Map<Integer, String> dstCidMap = progressRepo.getRunningPiece(cid);
        if (dstCidMap != null) {
            if (!PeerPieceStatus.RUNNING.equals(peerStatus)) {
                if (dstCidMap.containsKey(pieceNum)) {
                    dstCidMap.remove(pieceNum);
                }
            } else if (StringUtils.isNotBlank(dstCid)) {
                dstCidMap.put(pieceNum, dstCid);
            }
        }
        if (pieceBitSet.get(pieceNum * 8 + PeerPieceStatus.SUCCESS.getStatus())) {
            return false;
        }
        pieceBitSet.set(pieceNum * 8, (pieceNum + 1) * 8, false);// 先清空
        if (!PeerPieceStatus.WAIT.equals(peerStatus)) {
            pieceBitSet.set(peerStatus.getStatus() + pieceNum * 8);
        }

        return true;
    }

    private boolean fillPieceTasks(List<PieceTask> pieceTasks, Task task, String dstCid, Integer tmpPieceNum) {
        String taskId = task.getTaskId();
        try {
            PeerTask peerTask = peerTaskService.get(dstCid, taskId);

            PieceTask pieceTask = new PieceTask();
            pieceTask.setCid(dstCid);
            String pieceM5 = taskService.getPieceMd5(taskId, tmpPieceNum);

            pieceTask.setPieceMd5(pieceM5);
            if (StringUtils.isBlank(pieceTask.getPeerIp())) {
                PeerInfo peerInfo = peerService.get(dstCid);
                pieceTask.setPeerIp(peerInfo.getIp());
            }

            pieceTask.setPath(peerTask.getPath());
            pieceTask.setPeerPort(peerTask.getPort());

            pieceTask.setDownLink(Constants.LINK_LIMIT);
            pieceTask.setRange(RangeParseUtil.calculatePieceRange(tmpPieceNum, task.getPieceSize()));
            pieceTask.setPieceNum(tmpPieceNum);
            pieceTask.setPieceSize(task.getPieceSize());
            pieceTasks.add(pieceTask);
            return true;
        } catch (Exception e) {
            logger.error("taskId:{} dstCid:{} num:{}", taskId, dstCid, tmpPieceNum, e);
        }
        return false;
    }

    private void gcProgress(String taskId, List<String> cids, boolean isAll) {
        for (String cid : cids) {
            progressRepo.removeClientProgress(cid);
            progressRepo.removeProducerLoad(cid);
            progressRepo.removeRunningPiece(cid);
            progressRepo.removeServiceErrorInfo(cid);
            progressRepo.removeClientErrorInfo(cid);
            progressRepo.removeServiceDownInfo(cid);
            progressRepo.removeClientBlackInfo(cid);
        }
        if (isAll) {
            progressRepo.removeCdnProgress(taskId);
            String suffix = "@" + taskId;
            List<String> removedPiece = new ArrayList<String>();
            for (String key : progressRepo.getPieceProgress().keySet()) {
                if (key.endsWith(suffix)) {
                    removedPiece.add(key);
                }
            }
            for (String key : removedPiece) {
                progressRepo.removePieceProgress(key);
            }
        }
    }

    @Override
    public boolean gc(GcMeta gcMeta) {
        boolean result = false;
        if (gcMeta != null) {
            List<String> cids = gcMeta.getCids();
            String taskId = gcMeta.getTaskId();
            if (taskId != null && CollectionUtils.isNotEmpty(cids)) {
                gcProgress(taskId, cids, gcMeta.isAll());
            }
            result = true;
        }
        return result;
    }

    @Override
    public void updateDownInfo(String cid) {
        if (StringUtils.isNotBlank(cid) && progressRepo.getProducerLoad(cid) != null) {
            progressRepo.addServiceDownInfo(cid, System.currentTimeMillis());
        }
    }

    /**
     * @param pieceNums
     * @param runningPieces
     */
    private void parseNearNums(List<Integer> pieceNums, Set<Integer> runningPieces) {
        if (CollectionUtils.isNotEmpty(pieceNums)) {
            int centerNum = 0;
            if (CollectionUtils.isNotEmpty(runningPieces)) {
                int totalDistance = 0;
                for (Integer num : runningPieces) {
                    totalDistance += num;
                }
                centerNum = totalDistance / runningPieces.size();
            }

            Integer priorValue = null;
            Map<Integer, List<Integer>> priorMap = new TreeMap<Integer, List<Integer>>();
            for (Integer tmpNum : pieceNums) {
                priorValue = Math.abs(tmpNum - centerNum);
                List<Integer> samePrior = priorMap.get(priorValue);
                if (samePrior == null) {
                    samePrior = new ArrayList<Integer>();
                    priorMap.put(priorValue, samePrior);
                }
                samePrior.add(tmpNum);
            }
            pieceNums.clear();
            for (List<Integer> tmpPrior : priorMap.values()) {
                if (tmpPrior.size() > 1) {
                    Collections.shuffle(tmpPrior);
                }
                pieceNums.addAll(tmpPrior);
            }
        }
    }

    private int parsePieceLen(String taskId, int pieceNum) {
        try {
            String pieceM5 = taskService.getPieceMd5(taskId, pieceNum);
            int colonIndex = pieceM5.lastIndexOf(":");
            if (colonIndex > 0) {
                return Integer.parseInt(pieceM5.substring(colonIndex + 1));
            }
        } catch (Exception e) {
            logger.error("parse piece len error for taskId:{}", taskId, e);
        }
        return -1;
    }
}
