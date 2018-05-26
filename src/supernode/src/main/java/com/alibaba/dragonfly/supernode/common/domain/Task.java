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

import java.util.HashSet;
import java.util.Set;
import java.util.concurrent.ConcurrentHashMap;

import com.alibaba.dragonfly.supernode.common.enumeration.CdnStatus;

import org.apache.commons.lang3.StringUtils;

public class Task {

    private String taskId;
    private String sourceUrl;
    private String taskUrl;
    private String bizId;
    private String md5;

    private Long fileLength;
    private volatile Long httpFileLen;

    private volatile String realMd5;
    private volatile Integer pieceTotal = -1;
    private volatile CdnStatus cdnStatus = CdnStatus.WAIT;
    private ConcurrentHashMap<Integer, String> pieceMd5Map = new ConcurrentHashMap<>();

    private Integer pieceSize;

    private volatile boolean notReachable;

    /**
     * support headers
     */
    private volatile String[] headers;

    /**
     * df-daemon field
     */
    private boolean dfdaemon;
    private Set<String> authIps;
    private String curIp;

    private static int AUTH_IP_LIMIT = 5;

    public Task(String sourceUrl, String taskUrl, String md5, String bizId, String[] headers, boolean dfdaemon,
        String curIp) {
        this.sourceUrl = sourceUrl;
        this.taskUrl = taskUrl;
        this.md5 = md5;
        this.bizId = bizId;
        this.headers = headers;
        this.dfdaemon = dfdaemon;
        this.curIp = curIp;
    }

    public Task(String taskUrl, String md5, String bizId) {
        this.taskUrl = taskUrl;
        this.md5 = md5;
        this.bizId = bizId;
    }

    public Task() {
        super();
    }

    public boolean isFrozen() {
        return CdnStatus.WAIT.equals(cdnStatus) || CdnStatus.FAIL.equals(cdnStatus)
            || CdnStatus.SOURCE_ERROR.equals(cdnStatus);
    }

    public boolean isWait() {
        return CdnStatus.WAIT.equals(cdnStatus);
    }

    public boolean isSuccess() {
        return CdnStatus.SUCCESS.equals(cdnStatus);
    }

    public boolean isFail() {
        return CdnStatus.FAIL.equals(cdnStatus);
    }

    public boolean setPieceMd5(int pieceNum, String md5) {
        if (pieceNum >= 0 && StringUtils.isNotBlank(md5)) {
            pieceMd5Map.putIfAbsent(pieceNum, md5);
            return true;
        }
        return false;
    }

    public String getPieceMd5(int pieceNum) {
        String md5 = pieceMd5Map.get(pieceNum);
        return md5 == null ? "" : md5;
    }

    public String getTaskId() {
        return taskId;
    }

    public void setTaskId(String taskId) {
        this.taskId = taskId;
    }

    public String getSourceUrl() {
        return sourceUrl;
    }

    public void setSourceUrl(String sourceUrl) {
        this.sourceUrl = sourceUrl;
    }

    public String getMd5() {
        return md5;
    }

    public String getBizId() {
        return bizId;
    }

    public void setBizId(String bizId) {
        this.bizId = bizId;
    }

    public Integer getPieceTotal() {
        return pieceTotal;
    }

    public void setPieceTotal(Integer pieceTotal) {
        this.pieceTotal = pieceTotal;
    }

    public Long getFileLength() {
        return fileLength;
    }

    public void setFileLength(Long fileLength) {
        this.fileLength = fileLength;
    }

    public CdnStatus getCdnStatus() {
        return cdnStatus;
    }

    public void setCdnStatus(CdnStatus cdnStatus) {
        this.cdnStatus = cdnStatus;
    }

    public String getTaskUrl() {
        return taskUrl;
    }

    public void setTaskUrl(String taskUrl) {
        this.taskUrl = taskUrl;
    }

    @Override
    public boolean equals(Object obj) {
        if (this == obj) {
            return true;
        }
        if (obj instanceof Task) {
            Task task = (Task)obj;
            if (StringUtils.equals(task.getTaskUrl(), taskUrl)) {
                if (StringUtils.isNotBlank(md5)) {
                    return StringUtils.equals(md5, task.getMd5());
                } else if (StringUtils.isBlank(task.getMd5())) {
                    return StringUtils.equals(task.getBizId() == null ? "" : task.getBizId(),
                        bizId == null ? "" : bizId);
                }
            }
        }
        return false;
    }

    @Override
    public int hashCode() {
        int result = 17;
        if (taskUrl != null) {
            result = result * 31 + taskUrl.hashCode();
        }
        if (StringUtils.isNotBlank(md5)) {
            result = result * 31 + md5.hashCode();
        } else if (StringUtils.isNotBlank(bizId)) {
            result = result * 31 + bizId.hashCode();
        }
        return result;

    }

    public String getRealMd5() {
        return realMd5;
    }

    public void setRealMd5(String realMd5) {
        this.realMd5 = realMd5;
    }

    public ConcurrentHashMap<Integer, String> getPieceMd5Map() {
        return pieceMd5Map;
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

    public boolean isNotReachable() {
        return notReachable;
    }

    public void setNotReachable(boolean notReachable) {
        this.notReachable = notReachable;
    }

    public String[] getHeaders() {
        return headers;
    }

    public boolean isDfdaemon() {
        return dfdaemon;
    }

    public boolean inAuthIps(String ip) {
        return authIps.contains(ip);
    }

    public void addAuthIps(String ip) {
        if (authIps == null) {
            authIps = new HashSet<>();
        }
        if (authIps.size() >= AUTH_IP_LIMIT) {
            return;
        }
        authIps.add(ip);
    }

    public String getCurIp() {
        return curIp;
    }

    public void setCurIp(String curIp) {
        this.curIp = curIp;
    }

    public void setHeaders(String[] headers) {
        this.headers = headers;
    }
}
