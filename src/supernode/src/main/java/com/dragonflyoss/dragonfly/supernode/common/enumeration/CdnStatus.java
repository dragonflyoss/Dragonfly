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
package com.dragonflyoss.dragonfly.supernode.common.enumeration;

public enum CdnStatus {

    WAIT(0),
    RUNNING(1),
    FAIL(2),
    SUCCESS(3),
    SOURCE_ERROR(4);
    private int status;

    CdnStatus(int status) {
        this.status = status;
    }

    public int getStatus() {
        return status;
    }
}
