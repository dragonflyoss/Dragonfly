/*
 * Copyright 1999-2018 Alibaba Group.
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

package com.alibaba.dragonfly.supernode.rest.request;

import lombok.Data;

/**
 * Created on 2018/05/22
 *
 * @author lowzj
 */
@Data
public class RegistryRequest {
    private String cid;
    private String ip;
    private String hostName;
    private String rawUrl;
    private String taskUrl;
    private String md5;
    private String identifier;
    private String port;
    private String path;
    private String version;
    private String superNodeIp;
    /**
     * ["a:b","c:d","c:e"]
     */
    private String[] headers;
    private boolean dfdaemon;
}
