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

public class FileMetaData {

    private String url;
    /**
     * md5 in param
     */
    private String md5;
    private String bizId;
    private String taskId;
    private Long fileLength;
    private Long httpFileLen;
    private Integer pieceSize;
    private long accessTime;

    private long lastModified;
    private boolean finish;
    private boolean success;
    /**
     * real file md5
     */
    private String realMd5;
    private long interval = 0;

    public String getUrl() {
        return url;
    }

    public String getMd5() {
        return md5;
    }

    public String getBizId() {
        return bizId;
    }

    public String getTaskId() {
        return taskId;
    }

    public long getAccessTime() {
        return accessTime;
    }

    public long getLastModified() {
        return lastModified;
    }

    public void setUrl(String url) {
        this.url = url;
    }

    public void setMd5(String md5) {
        this.md5 = md5;
    }

    public void setBizId(String bizId) {
        this.bizId = bizId;
    }

    public void setTaskId(String taskId) {
        this.taskId = taskId;
    }

    public void setAccessTime(long accessTime) {
        this.accessTime = accessTime;
    }

    public void setLastModified(long lastModified) {
        this.lastModified = lastModified;
    }

    public void setFileLength(Long fileLength) {
        this.fileLength = fileLength;
    }

    public boolean isSuccess() {
        return success;
    }

    public void setSuccess(boolean success) {
        this.success = success;
    }

    public boolean isFinish() {
        return finish;
    }

    public void setFinish(boolean finish) {
        this.finish = finish;
    }

    public Long getFileLength() {
        return fileLength;
    }

    public String getRealMd5() {
        return realMd5;
    }

    public void setRealMd5(String realMd5) {
        this.realMd5 = realMd5;
    }

    public long getInterval() {
        return interval;
    }

    public void setInterval(long interval) {
        this.interval = interval;
    }

    public Long getHttpFileLen() {
        return httpFileLen;
    }

    public void setHttpFileLen(Long httpFileLen) {
        this.httpFileLen = httpFileLen;
    }

    public Integer getPieceSize() {
        return pieceSize;
    }

    public void setPieceSize(Integer pieceSize) {
        this.pieceSize = pieceSize;
    }
}
