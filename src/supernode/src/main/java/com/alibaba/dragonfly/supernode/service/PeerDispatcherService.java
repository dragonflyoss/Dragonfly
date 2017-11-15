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

import com.alibaba.dragonfly.supernode.common.enumeration.PeerPieceStatus;
import com.alibaba.dragonfly.supernode.common.enumeration.PeerTaskRequestResult;
import com.alibaba.dragonfly.supernode.common.enumeration.PeerTaskRequestStatus;
import com.alibaba.dragonfly.supernode.common.enumeration.PeerTaskStatus;
import com.alibaba.dragonfly.supernode.common.exception.ValidateException;
import com.alibaba.dragonfly.supernode.common.view.ResultInfo;

public abstract class PeerDispatcherService {
    /**
     * @param taskId
     * @param range
     * @param result
     * @param status
     * @return
     */
    public abstract ResultInfo process(String srcCid, String dstCid, String taskId, String range, String result,
        String status) throws ValidateException;

    /**
     * @param status
     * @param result
     * @return
     */
    protected PeerPieceStatus convertToPeerPieceStatus(int status, int result) {
        if (status == PeerTaskRequestStatus.RUNNING.getStatus()) {
            if (result == PeerTaskRequestResult.SUCCESS.getResult()) {
                return PeerPieceStatus.SUCCESS;
            } else if (result == PeerTaskRequestResult.FAIL.getResult()) {
                return PeerPieceStatus.FAIL;
            } else if (result == PeerTaskRequestResult.SEMISUC.getResult()) {
                return PeerPieceStatus.SEMISUC;
            }
        } else if (status == PeerTaskRequestStatus.START.getStatus()) {
            return PeerPieceStatus.WAIT;
        }
        return null;
    }

    /**
     * @param status
     * @param result
     * @return
     */
    protected PeerTaskStatus convertToPeerTaskStatus(int status, int result) {
        if (status == PeerTaskRequestStatus.RUNNING.getStatus()) {
            return PeerTaskStatus.RUNNING;
        }
        if (status == PeerTaskRequestStatus.FINISH.getStatus()) {
            if (result == PeerTaskRequestResult.SUCCESS.getResult()) {
                return PeerTaskStatus.SUCCESS;
            } else {
                return PeerTaskStatus.FAIL;
            }
        } else if (status == PeerTaskRequestStatus.START.getStatus()) {
            return PeerTaskStatus.WAIT;
        }
        return null;
    }
}
