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

import com.alibaba.dragonfly.supernode.common.domain.PeerInfo;
import com.alibaba.dragonfly.supernode.common.exception.ValidateException;
import com.alibaba.dragonfly.supernode.common.util.BeanPoolUtil;
import com.alibaba.dragonfly.supernode.common.view.ResultCode;
import com.alibaba.dragonfly.supernode.common.view.ResultInfo;
import com.alibaba.dragonfly.supernode.service.PeerRegistryService;
import com.alibaba.fastjson.JSON;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

@WebServlet(name = "PeerRegistryServlet", urlPatterns = "/peer/registry")
public class PeerRegistryServlet extends BaseServlet {

    private static final long serialVersionUID = 8361868437713593560L;
    private static final Logger logger = LoggerFactory.getLogger(PeerRegistryServlet.class);

    private static final PeerRegistryService peerRegistryService = BeanPoolUtil.getBean(PeerRegistryService.class);

    @Override
    protected void doGet(HttpServletRequest request, HttpServletResponse response)
        throws ServletException, IOException {
        doPost(request, response);
    }

    @Override
    protected void doPost(HttpServletRequest request, HttpServletResponse response)
        throws ServletException, IOException {
        try {
            Map<String, String> params = getSingleParameter(request);
            String url = params.get("rawUrl");
            String taskUrl = params.get("taskUrl");
            String md5 = params.get("md5");
            String bizId = params.get("identifier");
            String port = params.get("port");
            String path = params.get("path");
            String version = params.get("version");
            String superNodeIp = params.get("superNodeIp");
            /*
             * ["a:b","c:d","c:e"]
             */
            String[] headers = request.getParameterValues("headers");

            boolean dfdaemon = Boolean.parseBoolean(params.get("dfdaemon"));

            PeerInfo peerInfo = PeerInfo.newInstance(params);
            ResultInfo resultInfo = peerRegistryService.registryTask(url, taskUrl, md5, bizId, port, peerInfo, path,
                version, superNodeIp, headers, dfdaemon);
            result(response, resultInfo);
        } catch (ValidateException e) {
            logger.error("param is illegal", e);
            result(response, new ResultInfo(e.getCode(), e.getMessage(), null));
        } catch (Exception e) {
            logger.error("E_registry", e);
            result(response, new ResultInfo(ResultCode.SYSTEM_ERROR, e.getMessage(), null));
        }
    }

    private void result(HttpServletResponse response, ResultInfo resultInfo) {
        writeJson(response, JSON.toJSONString(resultInfo));
    }
}
