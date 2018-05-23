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
package com.alibaba.dragonfly.supernode.api;

import java.io.IOException;
import java.util.Map;

import javax.servlet.ServletException;
import javax.servlet.annotation.WebServlet;
import javax.servlet.http.HttpServletRequest;
import javax.servlet.http.HttpServletResponse;

import com.alibaba.dragonfly.supernode.common.Constants;
import com.alibaba.dragonfly.supernode.common.util.BeanPoolUtil;
import com.alibaba.dragonfly.supernode.common.util.NetConfigNotification;
import com.alibaba.dragonfly.supernode.common.view.ResultCode;
import com.alibaba.dragonfly.supernode.common.view.ResultInfo;
import com.alibaba.fastjson.JSON;

import org.apache.commons.lang.StringUtils;

@WebServlet(name = "ConfigServlet", urlPatterns = "/super/config")
public class ConfigServlet extends BaseServlet {

    private NetConfigNotification netConfigNotification = BeanPoolUtil.getBean(NetConfigNotification.class);

    @Override
    protected void doGet(HttpServletRequest request, HttpServletResponse response)
        throws ServletException, IOException {
        Map<String, String> params = getSingleParameter(request);
        /*
         * netRate unit is MB/s
         */
        String netRate = params.get("netRate");
        if (StringUtils.isNotBlank(netRate)) {
            netConfigNotification.freshNetRate(Integer.parseInt(netRate));
        }
        String debug = params.get("debug");
        if (debug != null) {
            if (StringUtils.equalsIgnoreCase(debug, "on")) {
                Constants.debugSwitcher = true;
            } else if (StringUtils.equalsIgnoreCase(debug, "off")) {
                Constants.debugSwitcher = false;
            }
        }

        result(response, new ResultInfo(ResultCode.SUCCESS));
    }

    @Override
    protected void doPost(HttpServletRequest request, HttpServletResponse response)
        throws ServletException, IOException {
        doGet(request, response);
    }

    private void result(HttpServletResponse response, ResultInfo resultInfo) {
        writeJson(response, JSON.toJSONString(resultInfo));
    }
}
