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
package com.alibaba.dragonfly.supernode.service.scheduler;

import com.alibaba.dragonfly.supernode.common.domain.gc.Recyclable;
import com.alibaba.dragonfly.supernode.common.enumeration.PeerPieceStatus;
import com.alibaba.dragonfly.supernode.common.view.ResultInfo;

public interface ProgressService extends Recyclable {

    /**
     * taskId progress(bitset(3)) taskId cid progress(bitset(3))
     *
     * @param taskId
     * @param cid
     * @return
     */
    ResultInfo initProgress(String taskId, String cid);

    /**
     * @param taskId
     * @param pieceNum
     * @return
     */
    ResultInfo updateProgress(String taskId, String srcCid, String dstCid, int pieceNum,
        PeerPieceStatus peerPieceStatus);

    /**
     * @param taskId
     * @param cid
     * @return
     */
    ResultInfo parseAvaliablePeerTasks(String taskId, String cid);

    /**
     * @param cid
     */
    void updateDownInfo(String cid);

    /**
     * @param taskId
     * @param srcCid
     * @param pieceNum
     * @return
     */
    ResultInfo processPieceSuc(String taskId, String srcCid, int pieceNum);

}
