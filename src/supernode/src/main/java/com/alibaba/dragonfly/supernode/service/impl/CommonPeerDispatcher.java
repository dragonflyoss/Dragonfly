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

import com.alibaba.dragonfly.supernode.common.enumeration.PeerPieceStatus;
import com.alibaba.dragonfly.supernode.common.enumeration.PeerTaskStatus;
import com.alibaba.dragonfly.supernode.common.exception.ValidateException;
import com.alibaba.dragonfly.supernode.common.util.Assert;
import com.alibaba.dragonfly.supernode.common.util.RangeParseUtil;
import com.alibaba.dragonfly.supernode.common.view.ResultCode;
import com.alibaba.dragonfly.supernode.common.view.ResultInfo;
import com.alibaba.dragonfly.supernode.service.PeerDispatcherService;
import com.alibaba.dragonfly.supernode.service.PeerTaskService;
import com.alibaba.dragonfly.supernode.service.scheduler.ProgressService;
import com.alibaba.dragonfly.supernode.service.timer.DataGcService;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Component;

@Component("commonPeerDispatcher")
public class CommonPeerDispatcher extends PeerDispatcherService {

    @Autowired
    private PeerTaskService peerTaskService;
    @Autowired
    private ProgressService progressService;
    @Autowired
    private DataGcService dataGcService;

    @Override
    public ResultInfo process(String srcCid, String dstCid, String taskId, String range, String result, String status)
        throws ValidateException {

        int requestStatus = Integer.parseInt(status);
        int requestResult = Integer.parseInt(result);

        dataGcService.updateAccessTime(taskId);
        PeerTaskStatus peerTaskStatus = convertToPeerTaskStatus(requestStatus, requestResult);
        Assert.assertNotNull(peerTaskStatus, ResultCode.PARAM_ERROR, "convertToPeerTaskStatus fail");
        if (peerTaskStatus.isWait()) {
            return processTaskStart(srcCid, taskId);
        } else if (peerTaskStatus.isRunning()) {
            int pieceNum = RangeParseUtil.calculatePieceNum(range);
            Assert.assertTrue(pieceNum >= 0, ResultCode.PARAM_ERROR, "invalid range");
            PeerPieceStatus pieceStatus = convertToPeerPieceStatus(requestStatus, requestResult);
            Assert.assertNotNull(pieceStatus, ResultCode.PARAM_ERROR, "convertToPeerPieceStatus fail");
            return processRunning(srcCid, dstCid, taskId, pieceNum, pieceStatus);
        } else {
            return processTaskFinish(srcCid, peerTaskStatus, taskId);
        }
    }

    private ResultInfo processTaskStart(String srcCid, String taskId) {
        boolean result = peerTaskService.updatePeerTaskStatus(srcCid, taskId, PeerTaskStatus.RUNNING);
        if (!result) {
            return new ResultInfo(ResultCode.SYSTEM_ERROR, "updatePeerTaskStatus fail", null);
        }
        return progressService.parseAvaliablePeerTasks(taskId, srcCid);
    }

    private ResultInfo processRunning(String srcCid, String dstCid, String taskId, int pieceNum,
        PeerPieceStatus peerPieceStatus) {
        ResultInfo result = progressService.updateProgress(taskId, srcCid, dstCid, pieceNum, peerPieceStatus);
        if (!result.successCode()) {
            return result;
        }
        return progressService.parseAvaliablePeerTasks(taskId, srcCid);
    }

    private ResultInfo processTaskFinish(String srcCid, PeerTaskStatus peerTaskStatus, String taskId) {
        boolean result = peerTaskService.updatePeerTaskStatus(srcCid, taskId, peerTaskStatus);
        if (!result) {
            return new ResultInfo(ResultCode.SYSTEM_ERROR, "updatePeerTaskStatus fail", null);
        }
        return new ResultInfo(ResultCode.SUCCESS);
    }
}
