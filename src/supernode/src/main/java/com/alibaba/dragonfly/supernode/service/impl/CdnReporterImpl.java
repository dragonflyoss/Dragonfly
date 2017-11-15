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

import com.alibaba.dragonfly.supernode.common.Constants;
import com.alibaba.dragonfly.supernode.common.enumeration.CdnStatus;
import com.alibaba.dragonfly.supernode.common.enumeration.PeerPieceStatus;
import com.alibaba.dragonfly.supernode.common.view.ResultInfo;
import com.alibaba.dragonfly.supernode.service.TaskService;
import com.alibaba.dragonfly.supernode.service.cdn.CdnReporter;
import com.alibaba.dragonfly.supernode.service.scheduler.ProgressService;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;

@Service("cdnReporter")
public class CdnReporterImpl implements CdnReporter {

    private static final Logger downLogger = LoggerFactory.getLogger("DownloaderLogger");

    private static final Logger logger = LoggerFactory.getLogger(CdnReporterImpl.class);

    @Autowired
    private ProgressService progressService;
    @Autowired
    private TaskService taskService;

    @Override
    public void reportPieceStatus(String taskId, int pieceNum, String md5, PeerPieceStatus status, String from) {
        if (status.isSuccess()) {
            if (!taskService.setPieceMd5(taskId, pieceNum, md5)) {
                throw new RuntimeException("report piece status fail for taskId:" + taskId);
            }
        }
        ResultInfo resultInfo =
            progressService.updateProgress(taskId, Constants.getSuperCid(taskId), null, pieceNum, status);
        if (!resultInfo.successCode()) {
            throw new RuntimeException(
                "report piece status fail for taskId:" + taskId + " reason:" + resultInfo.getMsg());
        }
        downLogger.info("taskId:{} pieceNum:{} from:{}", taskId, pieceNum, from);
    }

    @Override
    public void reportTaskStatus(String taskId, CdnStatus cdnStatus, String md5, Long fileLength, String from) {
        Integer pieceTotal = null;
        Integer pieceSize = taskService.get(taskId).getPieceSize();
        if (fileLength != null && fileLength > 0) {
            pieceTotal = (int)((fileLength + pieceSize - 1) / pieceSize);
        }
        if (!taskService.updateTaskInfo(taskId, md5, fileLength, pieceTotal,
            cdnStatus)) {
            throw new RuntimeException("report task status fail for taskId:" + taskId);
        }
        logger.info("taskId:{} fileLength:{} status:{} from:{}", taskId, fileLength, cdnStatus.name(), from);
    }

}
