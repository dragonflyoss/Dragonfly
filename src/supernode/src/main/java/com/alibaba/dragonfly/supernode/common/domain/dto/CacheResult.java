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
package com.alibaba.dragonfly.supernode.common.domain.dto;

import java.security.MessageDigest;

import org.apache.commons.codec.digest.DigestUtils;

public class CacheResult {

    private MessageDigest fileM5 = DigestUtils.getMd5Digest();
    private int startPieceNum;

    public MessageDigest getFileM5() {
        return fileM5;
    }

    public int getStartPieceNum() {
        return startPieceNum;
    }

    public void setFileM5(MessageDigest fileM5) {
        this.fileM5 = fileM5;
    }

    public void setStartPieceNum(int startPieceNum) {
        this.startPieceNum = startPieceNum;
    }

}
