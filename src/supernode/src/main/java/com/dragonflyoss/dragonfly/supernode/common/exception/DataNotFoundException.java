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
package com.dragonflyoss.dragonfly.supernode.common.exception;

public class DataNotFoundException extends RuntimeException {

    /**
     *
     */
    private static final long serialVersionUID = 4655324245971657198L;

    public DataNotFoundException(String type, String id, String msg) {
        super(new StringBuilder().append("type:").append(type).append(" id:")
            .append(id).append(" msg:").append(msg).toString());
    }
}
