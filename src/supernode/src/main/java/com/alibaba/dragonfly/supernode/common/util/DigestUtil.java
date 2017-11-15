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
package com.alibaba.dragonfly.supernode.common.util;

import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.util.List;

import org.apache.commons.codec.binary.Hex;
import org.apache.commons.codec.digest.DigestUtils;

public class DigestUtil {

    /**
     * @param target
     * @return
     */
    public static String sha256(String target) {
        return DigestUtils.sha256Hex(target);
    }

    public static String sha1(List<String> contents) {
        MessageDigest md = DigestUtils.getSha1Digest();
        for (String content : contents) {
            md.update(content.getBytes(StandardCharsets.UTF_8));
        }
        return Hex.encodeHexString(md.digest());
    }
}
