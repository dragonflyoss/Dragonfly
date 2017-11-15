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
package com.alibaba.dragonfly.supernode.common.domain.gc;

import java.util.List;

public class GcMeta {
    private String taskId;
    private List<String> cids;
    private boolean isAll;

    public GcMeta(String taskId, List<String> cids, boolean isAll) {
        this.taskId = taskId;
        this.cids = cids;
        this.isAll = isAll;
    }

    public String getTaskId() {
        return taskId;
    }

    public List<String> getCids() {
        return cids;
    }

    public void setTaskId(String taskId) {
        this.taskId = taskId;
    }

    public void setCids(List<String> cids) {
        this.cids = cids;
    }

    public boolean isAll() {
        return isAll;
    }

    public void setAll(boolean isAll) {
        this.isAll = isAll;
    }

}
