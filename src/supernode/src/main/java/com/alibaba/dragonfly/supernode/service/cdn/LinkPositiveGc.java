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
package com.alibaba.dragonfly.supernode.service.cdn;

import java.nio.file.Files;

import com.alibaba.dragonfly.supernode.service.cdn.util.PathUtil;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Component;

@Component
public class LinkPositiveGc {
    private static final Logger logger = LoggerFactory
        .getLogger("DataGcLogger");

    public boolean gc(String taskId) {
        try {
            Files.deleteIfExists(PathUtil.getUploadPath(taskId));
        } catch (Exception e) {
            logger.error("E_link_gc", e);
        }
        return true;
    }
}
