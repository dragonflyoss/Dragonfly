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

public class DownloadMetaData {

    private String md5;
    private long fileLength;
    private long startTime;
    private long readCost;

    public DownloadMetaData(String md5, long fileLength, long startTime,
        long readCost) {
        this.md5 = md5;
        this.fileLength = fileLength;
        this.startTime = startTime;
        this.readCost = readCost;
    }

    public String getMd5() {
        return md5;
    }

    public long getFileLength() {
        return fileLength;
    }

    public void setMd5(String md5) {
        this.md5 = md5;
    }

    public void setFileLength(long fileLength) {
        this.fileLength = fileLength;
    }

    public long getStartTime() {
        return startTime;
    }

    public long getReadCost() {
        return readCost;
    }

    public void setStartTime(long startTime) {
        this.startTime = startTime;
    }

    public void setReadCost(long readCost) {
        this.readCost = readCost;
    }

}
