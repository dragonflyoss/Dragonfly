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

import com.alibaba.dragonfly.supernode.common.domain.FileMetaData;
import com.alibaba.dragonfly.supernode.common.domain.Task;
import com.alibaba.dragonfly.supernode.common.domain.dto.CacheResult;

public interface CacheDetector {
    /**
     * check cache
     *
     * @param task
     * @return
     */
    CacheResult detectCache(Task task);

    /**
     * check same file
     *
     * @param task
     * @param fileMetaData
     * @return
     */
    boolean checkSameFile(Task task, FileMetaData fileMetaData);

}
