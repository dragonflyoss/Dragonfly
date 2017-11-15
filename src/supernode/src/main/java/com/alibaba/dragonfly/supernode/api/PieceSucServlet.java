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

import com.alibaba.dragonfly.supernode.common.enumeration.PeerPieceStatus;
import com.alibaba.dragonfly.supernode.common.util.BeanPoolUtil;
import com.alibaba.dragonfly.supernode.common.util.RangeParseUtil;
import com.alibaba.dragonfly.supernode.common.view.ResultCode;
import com.alibaba.dragonfly.supernode.common.view.ResultInfo;
import com.alibaba.dragonfly.supernode.service.scheduler.ProgressService;
import com.alibaba.fastjson.JSON;

import org.apache.commons.lang.StringUtils;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

@WebServlet(name = "PieceSucServlet", urlPatterns = "/peer/piece/suc")
public class PieceSucServlet extends BaseServlet {

    private static final long serialVersionUID = -319968641844201812L;
    private static final Logger logger = LoggerFactory
        .getLogger(PieceSucServlet.class);

    private static final ProgressService progressService = (ProgressService)BeanPoolUtil
        .getObject("progressService");

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
            String taskId = params.get("taskId");
            String cid = params.get("cid");
            String dstCid = params.get("dstCid");
            String range = params.get("pieceRange");

            if (StringUtils.isBlank(taskId) || StringUtils.isBlank(cid)
                || StringUtils.isBlank(range)) {
                result(response, new ResultInfo(ResultCode.PARAM_ERROR,
                    "some param is empty", null));
                return;
            }
            int pieceNum = RangeParseUtil.calculatePieceNum(range);
            ResultInfo resultInfo = null;
            if (pieceNum >= 0) {
                resultInfo = progressService.updateProgress(taskId, cid,
                    dstCid, pieceNum, PeerPieceStatus.SUCCESS);
            } else {
                logger.error("do piece suc fail for cid:{} pieceNum:{}", cid,
                    pieceNum);
            }
            if (resultInfo == null) {
                resultInfo = new ResultInfo(ResultCode.SYSTEM_ERROR);
            }
            result(response, resultInfo);
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
