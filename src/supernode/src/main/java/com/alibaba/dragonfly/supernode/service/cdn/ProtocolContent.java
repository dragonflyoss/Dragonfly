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
package com.alibaba.dragonfly.supernode.service.cdn;

import java.nio.ByteBuffer;

import com.alibaba.dragonfly.supernode.common.domain.DownloadMetaData;

public class ProtocolContent {
    private static final String TYPE_PIECE = "piece";
    private static final String TYPE_TASK = "task";

    private String type;

    // piece
    private ByteBuffer content;
    private Integer pieceNum;

    // whether task is finish
    private boolean finish;
    private boolean success;
    private DownloadMetaData downloadMetaData;

    public static ProtocolContent buildPieceContent(ByteBuffer content, Integer pieceNum) {
        ProtocolContent protocolContent = new ProtocolContent();
        protocolContent.setType(TYPE_PIECE);
        protocolContent.setContent(content);
        protocolContent.setPieceNum(pieceNum);
        return protocolContent;
    }

    public static ProtocolContent buildTaskResult(boolean finish, boolean success, DownloadMetaData downloadMetaData) {
        ProtocolContent protocolContent = new ProtocolContent();
        protocolContent.setType(TYPE_TASK);
        protocolContent.setFinish(finish);
        protocolContent.setSuccess(success);
        protocolContent.setDownloadMetaData(downloadMetaData);
        return protocolContent;
    }

    public boolean isPieceType() {
        return TYPE_PIECE.equals(type);
    }

    public boolean isTaskType() {
        return TYPE_TASK.equals(type);
    }

    public ByteBuffer getContent() {
        return content;
    }

    public Integer getPieceNum() {
        return pieceNum;
    }

    public void setContent(ByteBuffer content) {
        this.content = content;
    }

    public void setPieceNum(Integer pieceNum) {
        this.pieceNum = pieceNum;
    }

    public String getType() {
        return type;
    }

    public boolean isFinish() {
        return finish;
    }

    public void setType(String type) {
        this.type = type;
    }

    public void setFinish(boolean finish) {
        this.finish = finish;
    }

    public boolean isSuccess() {
        return success;
    }

    public void setSuccess(boolean success) {
        this.success = success;
    }

    public DownloadMetaData getDownloadMetaData() {
        return downloadMetaData;
    }

    public void setDownloadMetaData(DownloadMetaData downloadMetaData) {
        this.downloadMetaData = downloadMetaData;
    }

}
