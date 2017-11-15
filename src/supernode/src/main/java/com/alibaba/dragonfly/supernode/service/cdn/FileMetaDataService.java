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

import java.io.IOException;
import java.util.List;

import com.alibaba.dragonfly.supernode.common.domain.FileMetaData;
import com.alibaba.dragonfly.supernode.common.domain.Task;

public interface FileMetaDataService {

    FileMetaData readFileMetaData(String taskId);

    FileMetaData createMetaData(Task task) throws IOException;

    boolean updateAccessTime(String taskId, long accessTime);

    boolean updateLastModified(String taskId, long lastModified);

    boolean updateStatusAndResult(String taskId, boolean finish, boolean success, String realMd5, Long fileLength);

    /**
     * @param taskId
     * @param fileMd5
     * @param pieceMd5s
     */
    void writePieceMd5(String taskId, String fileMd5, List<String> pieceMd5s);

    /**
     * @param taskId
     * @param fileMd5
     * @return
     */
    List<String> readPieceMd5(String taskId, String fileMd5);

}
