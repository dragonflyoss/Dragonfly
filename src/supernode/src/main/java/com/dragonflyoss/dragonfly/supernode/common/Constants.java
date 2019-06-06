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
package com.dragonflyoss.dragonfly.supernode.common;

/**
 * @author lowzj
 */
public final class Constants {

    public static final int PORT = 8001;
    public static final String CDN_NODE_PREFIX = "cdnnode:";

    public static String localIp;
    public static String SUPER_NODE_CID = null;

    public static final int FAIL_COUNT_LIMIT = 5;

    public static final int DEFAULT_SCHEDULER_CORE_POOL_SIZE = 10;

    //-------------------------------------------------------------------------
    // dfget

    public static final String DFGET_PATH = "/usr/local/bin/dfget";

    //-------------------------------------------------------------------------
    // directories

    public static final String DEFAULT_BASE_HOME = "/home/admin/supernode";
    public static final String HTTP_SUB_PATH = "/qtdown/";
    public static final String DOWN_SUB_PATH = "/download/";
    public static final String PREHEAT_SUB_PATH = "/preheat/";

    public static String DOWNLOAD_HOME = DEFAULT_BASE_HOME + "/repo" + DOWN_SUB_PATH;
    public static String UPLOAD_HOME = DEFAULT_BASE_HOME + "/repo" + HTTP_SUB_PATH;
    public static String PREHEAT_HOME = DEFAULT_BASE_HOME + "/repo" + PREHEAT_SUB_PATH;

    //-------------------------------------------------------------------------

    public static final int DEFAULT_PIECE_SIZE = 4 * 1024 * 1024;
    public static final int PIECE_SIZE_LIMIT = 15 * 1024 * 1024;
    /**
     * 4 bytes head and 1 byte tail
     */
    public static final int PIECE_HEAD_SIZE = 4;
    /**
     * can not change
     */
    public static final int PIECE_WRAP_SIZE = PIECE_HEAD_SIZE + 1;

    //-------------------------------------------------------------------------

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
    public static final int DEFAULT_SYSTEM_NEED_RATE = 20;
    /**
     * unit is MB
     */
    public static final int DEFAULT_TOTAL_LIMIT = 200;

    //-------------------------------------------------------------------------

    private static String getSuperNodeCidPrefix() {
        return CDN_NODE_PREFIX + localIp + "~";
    }

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
