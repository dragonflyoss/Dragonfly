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

package com.dragonflyoss.dragonfly.supernode.common.domain;

import java.util.Map;

import com.dragonflyoss.dragonfly.supernode.common.enumeration.PreheatTaskStatus;
import lombok.Data;

/**
 * Created on 2018/11/02
 *
 * @author lowzj
 */
@Data
public class PreheatTask {
    private String id;
    private String url;
    private String type;
    private String filter;
    private String identifier;
    private Map<String, String> headers;

    private PreheatTaskStatus status;
    private long startTime;
    private long finishTime;
    private String errorMsg;
}
