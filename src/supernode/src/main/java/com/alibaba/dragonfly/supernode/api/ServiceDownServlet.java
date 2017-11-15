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

import com.alibaba.dragonfly.supernode.common.util.BeanPoolUtil;
import com.alibaba.dragonfly.supernode.common.view.ResultCode;
import com.alibaba.dragonfly.supernode.common.view.ResultInfo;
import com.alibaba.dragonfly.supernode.service.lock.LockService;
import com.alibaba.dragonfly.supernode.service.scheduler.ProgressService;
import com.alibaba.fastjson.JSON;

@WebServlet(name = "ServiceDownServlet", urlPatterns = "/peer/service/down")
public class ServiceDownServlet extends BaseServlet {

    private static final long serialVersionUID = 2353743793926314411L;

    private static final ProgressService progressService = (ProgressService)BeanPoolUtil
        .getObject("progressService");
    private static final LockService lockService = BeanPoolUtil
        .getBean(LockService.class);

    @Override
    protected void doGet(HttpServletRequest request,
        HttpServletResponse response) throws ServletException, IOException {
        doPost(request, response);
    }

    @Override
    protected void doPost(HttpServletRequest request,
        HttpServletResponse response) throws ServletException, IOException {
        try {
            Map<String, String> params = getSingleParameter(request);
            String cid = params.get("cid");
            String taskId = params.get("taskId");
            lockService.lockTaskOnRead(taskId);
            try {
                progressService.updateDownInfo(cid);
            } finally {
                lockService.unlockTaskOnRead(taskId);
            }

            result(response, new ResultInfo(ResultCode.SUCCESS));
        } catch (Exception e) {
            result(response,
                new ResultInfo(ResultCode.SYSTEM_ERROR, e.getMessage(),
                    null));
        }
    }

    private void result(HttpServletResponse response, ResultInfo resultInfo) {
        writeJson(response, JSON.toJSONString(resultInfo));
    }

}
