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

package com.alibaba.dragonfly.supernode.rest.controller;

import com.alibaba.dragonfly.supernode.common.Constants;
import com.alibaba.dragonfly.supernode.common.util.NetConfigNotification;
import com.alibaba.dragonfly.supernode.common.view.ResultCode;
import com.alibaba.dragonfly.supernode.common.view.ResultInfo;

import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

/**
 * @author lowzj
 */
@RestController
@Slf4j
public class ConfigController {

    @Autowired
    private NetConfigNotification netConfigNotification;

    @PostMapping(value = "/super/config")
    public ResultInfo update(@RequestParam(required = false) Integer netRate,
                             @RequestParam(required = false) Boolean debug) {
        // netRate unit is MB/s
        if (netRate != null) {
            netConfigNotification.freshNetRate(netRate);
        }
        if (debug != null) {
            Constants.debugSwitcher = debug;
        }
        log.info("update config, netRate:{} debug:{}", netRate, debug);
        return new ResultInfo(ResultCode.SUCCESS);
    }
}
