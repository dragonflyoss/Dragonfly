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

import com.alibaba.dragonfly.supernode.common.exception.ValidateException;
import com.alibaba.dragonfly.supernode.common.util.BeanPoolUtil;
import com.alibaba.dragonfly.supernode.common.view.ResultCode;
import com.alibaba.dragonfly.supernode.common.view.ResultInfo;
import com.alibaba.dragonfly.supernode.service.impl.CommonPeerDispatcher;
import com.alibaba.fastjson.JSON;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

@WebServlet(name = "PeerTaskServlet", urlPatterns = "/peer/task")
public class PeerTaskServlet extends BaseServlet {

    private static final long serialVersionUID = 8361868437713593560L;

    private static final Logger logger = LoggerFactory.getLogger(PeerTaskServlet.class);

    private static final CommonPeerDispatcher commonPeerDispatcher = BeanPoolUtil.getBean(CommonPeerDispatcher.class);

    @Override
    protected void doGet(HttpServletRequest request, HttpServletResponse response)
        throws ServletException, IOException {
        doPost(request, response);
    }

    @Override
    protected void doPost(final HttpServletRequest request, HttpServletResponse response)
        throws ServletException, IOException {
        long start = System.currentTimeMillis();
        try {
            Map<String, String> params = getSingleParameter(request);
            /*
             * src client id
             */
            String srcCid = params.get("srcCid");
            /*
             * dst client id
             */
            String dstCid = params.get("dstCid");
            String range = params.get("range");
            String result = params.get("result");
            String status = params.get("status");
            String taskId = params.get("taskId");

            ResultInfo processResult = commonPeerDispatcher.process(srcCid, dstCid, taskId, range, result, status);
            if (processResult != null) {
                result(response, processResult);
            } else {
                result(response,
                    new ResultInfo(ResultCode.SYSTEM_ERROR, JSON.toJSONString(params), null));
            }
            long end = System.currentTimeMillis();
            if (end - start > 1000) {
                logger.warn("do peer task cost:{}ms", end - start);
            }
        } catch (ValidateException e) {
            result(response, new ResultInfo(e.getCode(), e.getMessage(), null));
        } catch (Exception e) {
            logger.error("E_PeerTaskServlet", e);
            result(response, new ResultInfo(ResultCode.SYSTEM_ERROR, e.getMessage(), null));
        }
    }

    private void result(HttpServletResponse response, ResultInfo resultInfo) {
        writeJson(response, JSON.toJSONString(resultInfo));
    }
}
