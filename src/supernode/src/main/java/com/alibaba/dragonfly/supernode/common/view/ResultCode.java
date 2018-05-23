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

import java.util.HashMap;
import java.util.Map;

public class ResultCode {
    public static final int SUCCESS = 200;

    public static final int NOT_FOUND = 404;

    public static final int SYSTEM_ERROR = 500;
    public static final int PARAM_ERROR = 501;
    public static final int TARGET_NOT_FOUND = 502;

    public static final int PEER_FINISH = 600;
    public static final int PEER_CONTINUE = 601;
    public static final int PEER_WAIT = 602;
    public static final int PEER_LIMIT = 603;
    public static final int SUPER_FAIL = 604;
    public static final int UNKNOWN_ERROR = 605;
    public static final int TASK_CONFLICT = 606;
    public static final int URL_NOT_REACH = 607;
    public static final int NEED_AUTH = 608;
    public static final int WAIT_AUTH = 609;

    private static final Map<Integer, String> descMap = new HashMap<Integer, String>();

    static {
        descMap.put(SUCCESS, "success");
        descMap.put(SYSTEM_ERROR, "system error");
        descMap.put(PEER_FINISH, "peer task end");
        descMap.put(PEER_CONTINUE, "peer task go on");
        descMap.put(PEER_WAIT, "peer task wait");
        descMap.put(SUPER_FAIL, "super node sync source fail");
        descMap.put(PARAM_ERROR, "param is illegal");
        descMap.put(TARGET_NOT_FOUND, "target not found");
        descMap.put(PEER_LIMIT, "peer down limit");
        descMap.put(UNKNOWN_ERROR, "unknown error");
        descMap.put(TASK_CONFLICT, "task conflict");
        descMap.put(URL_NOT_REACH, "url is not reachable");
        descMap.put(NEED_AUTH, "need auth");
        descMap.put(WAIT_AUTH, "wait auth");
    }

    public static String getDesc(int code) {
        return descMap.get(code);
    }

}
