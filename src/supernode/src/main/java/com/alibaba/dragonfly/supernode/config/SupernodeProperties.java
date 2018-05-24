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

package com.alibaba.dragonfly.supernode.config;

import javax.annotation.PostConstruct;
import java.nio.file.Files;
import java.nio.file.Paths;

import com.alibaba.dragonfly.supernode.common.Constants;

import lombok.Data;
import lombok.extern.slf4j.Slf4j;
import org.springframework.boot.context.properties.ConfigurationProperties;

/**
 * Created on 2018/05/23
 *
 * @author lowzj
 */
@ConfigurationProperties("supernode")
@Slf4j
@Data
public class SupernodeProperties {
    /**
     * working directory of supernode, default: /home/admin/supernode
     */
    private String baseHome = Constants.DEFAULT_BASE_HOME;

    /**
     * the network rate reserved for system , default: 20 MB/s
     */
    private int systemNeedRate = Constants.DEFAULT_SYSTEM_NEED_RATE;

    /**
     * the network rate that supernode can use, default: 200 MB/s
     */
    private int totalLimit = Constants.DEFAULT_TOTAL_LIMIT;

    /**
     * the core pool size of ScheduledExecutorService, default: 10
     */
    private int schedulerCorePoolSize = Constants.DEFAULT_SCHEDULER_CORE_POOL_SIZE;

    @PostConstruct
    public void init() {
        String cdnHome = baseHome + "/repo";
        Constants.DOWNLOAD_HOME = cdnHome + Constants.DOWN_SUB_PATH;
        Constants.UPLOAD_HOME = cdnHome + Constants.HTTP_SUB_PATH;

        try {
            Files.createDirectories(Paths.get(Constants.DOWNLOAD_HOME));
            Files.createDirectories(Paths.get(Constants.UPLOAD_HOME));
        } catch (Exception e) {
            log.error("create repo dir error", e);
            System.exit(1);
        }
    }
}
