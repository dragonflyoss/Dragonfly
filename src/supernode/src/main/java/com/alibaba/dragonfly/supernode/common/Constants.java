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
package com.alibaba.dragonfly.supernode.common;

import java.nio.file.Files;
import java.nio.file.Paths;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class Constants {
    private static final Logger logger = LoggerFactory.getLogger(Constants.class);

    public static String localIp;
    public static final int port = 8001;
    public static final String CDN_HOME = "/home/admin/supernode/repo";

    public static String DOWNLOAD_HOME = "";
    public static String UPLOAD_HOME = "";

    public static final String HTTP_SUB_PATH = "/qtdown/";
    public static final String DOWN_SUB_PATH = "/download/";

    static {
        DOWNLOAD_HOME = CDN_HOME + DOWN_SUB_PATH;
        UPLOAD_HOME = CDN_HOME + HTTP_SUB_PATH;
        try {
            Files.createDirectories(Paths.get(DOWNLOAD_HOME));
            Files.createDirectories(Paths.get(UPLOAD_HOME));
        } catch (Exception e) {
            logger.error("create repo dir error", e);
            System.exit(1);
        }
    }

    public static int DEFAULT_PIECE_SIZE = 4 * 1024 * 1024;
    public static int PIECE_SIZE_LIMIT = 15 * 1024 * 1024;
    /**
     * 4 bytes head and 1 byte tail
     */
    public static final int PIECE_HEAD_SIZE = 4;
    /**
     * can not change
     */
    public static final int PIECE_WRAP_SIZE = PIECE_HEAD_SIZE + 1;
    public static int PEER_UP_LIMIT = 5;
    public static int PEER_DOWN_LIMIT = 4;
    public static int ELIMINATION_LIMIT = 5;
    /**
     * unit is KB
     */
    public static volatile int LINK_LIMIT = 20 * 1024;
    /**
     * unit is MB
     */
    public static final int SYSTEM_NEED_RATE = 20;
    /**
     * unit is MB
     */
    public static final int DEFAULT_TOTAL_LIMIT = 200;

    public static String getSuperNodeCidPrefix() {
        return new StringBuilder("cdnnode:").append(localIp).append("~").toString();
    }

    public static String SUPER_NODE_CID = null;

    public static final int FAIL_COUNT_LIMIT = 5;

    public static void generateNodeCid() {
        if (SUPER_NODE_CID == null) {
            SUPER_NODE_CID = getSuperNodeCidPrefix();
        }
    }

    public static String getSuperCid(String taskId) {
        return SUPER_NODE_CID + taskId;
    }

    public volatile static boolean debugSwitcher = false;

    public static boolean isDebugEnabled() {
        return debugSwitcher;
    }
}
