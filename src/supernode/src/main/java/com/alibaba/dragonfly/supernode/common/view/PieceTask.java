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
package com.alibaba.dragonfly.supernode.common.view;

public class PieceTask {

    private String range;
    private int pieceNum;
    private int pieceSize;
    private String pieceMd5;
    private String cid;
    private String peerIp;
    private int peerPort;
    private String path;
    private int downLink;

    public int getPieceNum() {
        return pieceNum;
    }

    public void setPieceNum(int pieceNum) {
        this.pieceNum = pieceNum;
    }

    public int getPieceSize() {
        return pieceSize;
    }

    public void setPieceSize(int pieceSize) {
        this.pieceSize = pieceSize;
    }

    public String getPath() {
        return path;
    }

    public void setPath(String path) {
        this.path = path;
    }

    public String getRange() {
        return range;
    }

    public String getPeerIp() {
        return peerIp;
    }

    public int getPeerPort() {
        return peerPort;
    }

    public int getDownLink() {
        return downLink;
    }

    public void setRange(String range) {
        this.range = range;
    }

    public void setPeerIp(String peerIp) {
        this.peerIp = peerIp;
    }

    public void setPeerPort(int peerPort) {
        this.peerPort = peerPort;
    }

    public void setDownLink(int downLink) {
        this.downLink = downLink;
    }

    public String getCid() {
        return cid;
    }

    public void setCid(String cid) {
        this.cid = cid;
    }

    public String getPieceMd5() {
        return pieceMd5;
    }

    public void setPieceMd5(String pieceMd5) {
        this.pieceMd5 = pieceMd5;
    }
}
