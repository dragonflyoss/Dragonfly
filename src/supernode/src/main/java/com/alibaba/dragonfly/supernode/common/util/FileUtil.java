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

import java.nio.file.Files;
import java.nio.file.Path;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class FileUtil {

    private static final Logger logger = LoggerFactory.getLogger(FileUtil.class);

    public static boolean createSymbolicLink(Path linkPath, Path targetPath) {
        try {
            Files.createDirectories(linkPath.getParent());
            Files.createDirectories(targetPath.getParent());
            try {
                if (Files.isSymbolicLink(linkPath) && Files.exists(targetPath)
                    && Files.isSameFile(linkPath, targetPath)) {
                    return true;
                }
            } catch (Exception e) {
                logger.warn("access link:{} error", linkPath.toString(), e);

            }
            if (Files.deleteIfExists(linkPath)) {
                logger.warn("linkPath:{} not match targetPath:{}", linkPath.toString(), targetPath.toString());
            }
            Files.createSymbolicLink(linkPath, targetPath);
            return true;
        } catch (Exception e) {
            logger.error("E_createSymbolicLink", e);
        }
        return false;
    }

}
