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
package com.alibaba.dragonfly.supernode.common.domain;

import com.alibaba.dragonfly.supernode.common.enumeration.PeerTaskStatus;

public class PeerTask {

    private String taskId;
    private Integer pieceSize;
    private String cid;
    private int port;
    private String path;
    private volatile PeerTaskStatus status = PeerTaskStatus.WAIT;

    public PeerTask(String cid, String taskId, int port, String path, Integer pieceSize) {
        this.cid = cid;
        this.taskId = taskId;
        this.port = port;
        this.path = path;
        this.pieceSize = pieceSize;
    }

    public PeerTask() {
    }

    public boolean isSuccess() {
        return status != null && status.isSuccess();
    }

    public String getTaskId() {
        return taskId;
    }

    public void setTaskId(String taskId) {
        this.taskId = taskId;
    }

    public String getCid() {
        return cid;
    }

    public void setCid(String cid) {
        this.cid = cid;
    }

    public PeerTaskStatus getStatus() {
        return status;
    }

    public void setStatus(PeerTaskStatus status) {
        this.status = status;
    }

    public String getPath() {
        return path;
    }

    public void setPath(String path) {
        this.path = path;
    }

    public int getPort() {
        return port;
    }

    public void setPort(int port) {
        this.port = port;
    }

    public Integer getPieceSize() {
        return pieceSize;
    }

    public void setPieceSize(Integer pieceSize) {
        this.pieceSize = pieceSize;
    }
}
