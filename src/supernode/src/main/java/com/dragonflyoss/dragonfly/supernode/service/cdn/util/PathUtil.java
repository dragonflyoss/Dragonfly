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
package com.dragonflyoss.dragonfly.supernode.service.cdn.util;

import java.io.File;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;

import com.dragonflyoss.dragonfly.supernode.common.Constants;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

/**
 * @author zj
 */
public class PathUtil {
    private static final Logger logger = LoggerFactory.getLogger(PathUtil.class);


    public static String getDownloadPathStr(String taskId) {
        return Constants.DOWNLOAD_HOME + taskId.substring(0, 3) + File.separator + taskId;
    }

    public static Path getDownloadPath(String taskId) {
        String pathStr = getDownloadPathStr(taskId);
        return Paths.get(pathStr);
    }

    public static Path getMetaDataPath(String taskId) {
        String pathStr = Constants.DOWNLOAD_HOME + taskId.substring(0, 3) + File.separator + taskId + ".meta";
        return Paths.get(pathStr);
    }

    public static Path getMd5DataPath(String taskId) {
        String pathStr = Constants.DOWNLOAD_HOME + taskId.substring(0, 3) + File.separator + taskId + ".md5";
        return Paths.get(pathStr);
    }

    public static Path getPreheatMetaPath(String preheatTaskId) {
        String pathStr = getDownloadPathStr(preheatTaskId) + ".preheat";
        return Paths.get(pathStr);
    }

    public static Path getPreheatIndexPath() {
        String pathStr = Constants.PREHEAT_HOME + "preheat.index";
        return Paths.get(pathStr);
    }

    public static Path getUploadPath(String taskId) {
        String pathStr = Constants.UPLOAD_HOME + taskId.substring(0, 3) + File.separator + taskId;
        return Paths.get(pathStr);
    }

    public static String getHttpPath(String taskId) {
        return Constants.HTTP_SUB_PATH + taskId.substring(0, 3) + "/" + taskId;
    }

    public static void deleteTaskFiles(String taskId, boolean deleteUploadPath) {
        try {
            Path downPath = getDownloadPath(taskId);
            Files.deleteIfExists(downPath);

            if (deleteUploadPath) {
                Files.deleteIfExists(getUploadPath(taskId));
            }

            Files.deleteIfExists(getMetaDataPath(taskId));
            Files.deleteIfExists(getMd5DataPath(taskId));
        } catch (Exception e) {
            logger.error("delete files error for taskId:{}", taskId, e);
        }
    }

}
