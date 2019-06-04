/*
 * Copyright The Dragonfly Authors.
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

package com.dragonflyoss.dragonfly.supernode.rest.controller;

import javax.servlet.http.HttpServletRequest;

import com.alibaba.fastjson.JSON;

import com.dragonflyoss.dragonfly.supernode.common.domain.ClientErrorInfo;
import com.dragonflyoss.dragonfly.supernode.common.domain.PeerInfo;
import com.dragonflyoss.dragonfly.supernode.common.enumeration.PeerPieceStatus;
import com.dragonflyoss.dragonfly.supernode.common.exception.ValidateException;
import com.dragonflyoss.dragonfly.supernode.common.util.RangeParseUtil;
import com.dragonflyoss.dragonfly.supernode.common.view.ResultCode;
import com.dragonflyoss.dragonfly.supernode.common.view.ResultInfo;
import com.dragonflyoss.dragonfly.supernode.rest.request.PullPieceTaskRequest;
import com.dragonflyoss.dragonfly.supernode.rest.request.RegistryRequest;
import com.dragonflyoss.dragonfly.supernode.rest.request.ReportPieceRequest;
import com.dragonflyoss.dragonfly.supernode.rest.request.ReportServiceDownRequest;
import com.dragonflyoss.dragonfly.supernode.service.PeerRegistryService;
import com.dragonflyoss.dragonfly.supernode.service.impl.CommonPeerDispatcher;
import com.dragonflyoss.dragonfly.supernode.service.lock.LockService;
import com.dragonflyoss.dragonfly.supernode.service.repair.ClientErrorHandleService;
import com.dragonflyoss.dragonfly.supernode.service.scheduler.ProgressService;
import com.dragonflyoss.dragonfly.supernode.service.timer.DataGcService;

import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.MediaType;
import org.springframework.web.bind.annotation.DeleteMapping;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

/**
 * @author lowzj
 */
@RestController
@RequestMapping(value = "/peer")
@Slf4j
public class PeerController {

    @Autowired
    private HttpServletRequest request;

    @Autowired
    private ClientErrorHandleService clientErrorHandleService;

    @Autowired
    private PeerRegistryService peerRegistryService;

    @Autowired
    private CommonPeerDispatcher commonPeerDispatcher;

    @Autowired
    private ProgressService progressService;

    @Autowired
    private LockService lockService;

    @Autowired
    private DataGcService dataGcService;

    @PostMapping(value = "/registry")
    public ResultInfo doRegistry(RegistryRequest req) {
        ResultInfo res = null;
        try {
            res = peerRegistryService.registryTask(req.getRawUrl(),
                req.getTaskUrl(),
                req.getMd5(),
                req.getIdentifier(),
                req.getPort(),
                PeerInfo.newInstance(req.getCid(), req.getIp(), req.getHostName()),
                req.getPath(),
                req.getVersion(),
                req.getSuperNodeIp(),
                req.getHeaders(),
                req.isDfdaemon());
        } catch (ValidateException e) {
            log.error("param is illegal", e);
            res = new ResultInfo(e.getCode(), e.getMessage(), null);
        } catch (Exception e) {
            log.error("E_registry", e);
            res = new ResultInfo(ResultCode.SYSTEM_ERROR, e.getMessage(), null);
        }
        debug("doRegistry", req, res);
        return res;
    }

    @PostMapping(value = "/registry", consumes = {MediaType.APPLICATION_JSON_UTF8_VALUE, MediaType.APPLICATION_JSON_VALUE})
    public ResultInfo doRegistryWithJson(@RequestBody RegistryRequest req) {
        return doRegistry(req);
    }

    @GetMapping(value = "/task")
    public ResultInfo pullPieceTask(PullPieceTaskRequest req) {
        ResultInfo res = null;
        long start = System.currentTimeMillis();
        try {
            res = commonPeerDispatcher.process(
                req.getSrcCid(),
                req.getDstCid(),
                req.getTaskId(),
                req.getRange(),
                req.getResult(),
                req.getStatus());
            if (res == null) {
                res = new ResultInfo(ResultCode.SYSTEM_ERROR, JSON.toJSONString(req), null);
            }
            long end = System.currentTimeMillis();
            if (end - start > 1000) {
                log.warn("do peer task cost:{}ms", end - start);
            }
        } catch (ValidateException e) {
            res = new ResultInfo(e.getCode(), e.getMessage(), null);
        } catch (Exception e) {
            log.error("E_PeerTaskServlet", e);
            res = new ResultInfo(ResultCode.SYSTEM_ERROR, e.getMessage(), null);
        }
        debug("pullPieceTask", req, res);
        return res;
    }

    @GetMapping(value = "/piece/suc")
    public ResultInfo reportPieceSuc(ReportPieceRequest req) {
        ResultInfo res = null;
        try {
            String taskId = req.getTaskId();
            String cid = req.getCid();
            String dstCid = req.getDstCid();
            String range = req.getPieceRange();

            if (StringUtils.isBlank(taskId) || StringUtils.isBlank(cid)
                || StringUtils.isBlank(range)) {
                res = new ResultInfo(ResultCode.PARAM_ERROR,
                    "some param is empty", null);
            } else {
                int pieceNum = RangeParseUtil.calculatePieceNum(range);
                if (pieceNum >= 0) {
                    res = progressService.updateProgress(taskId, cid,
                        dstCid, pieceNum, PeerPieceStatus.SUCCESS);
                } else {
                    log.error("do piece suc fail for cid:{} pieceNum:{}", cid,
                        pieceNum);
                }
                if (res == null) {
                    res = new ResultInfo(ResultCode.SYSTEM_ERROR);
                }
            }
        } catch (Exception e) {
            res = new ResultInfo(ResultCode.SYSTEM_ERROR, e.getMessage(), null);
        }
        debug("reportPiece", req, res);
        return res;
    }

    @GetMapping(value = "/piece/error")
    public ResultInfo reportPieceError(ClientErrorInfo req) {
        clientErrorHandleService.handleClientError(req);
        ResultInfo res = new ResultInfo(ResultCode.SUCCESS);
        debug("reportPieceError", req, res);
        return res;
    }

    @GetMapping(value = "/service/down")
    public ResultInfo reportServiceDown(ReportServiceDownRequest req) {
        ResultInfo res = null;
        try {
            String cid = req.getCid();
            String taskId = req.getTaskId();
            lockService.lockTaskOnRead(taskId);
            try {
                progressService.updateDownInfo(cid);
            } finally {
                lockService.unlockTaskOnRead(taskId);
            }

            res = new ResultInfo(ResultCode.SUCCESS);
        } catch (Exception e) {
            res = new ResultInfo(ResultCode.SYSTEM_ERROR, e.getMessage(), null);
        }
        debug("reportServiceDown", req, res);
        return res;
    }

    @DeleteMapping(value = "/tasks/{taskId}")
    public ResultInfo deleteTask(@PathVariable String taskId){
        ResultInfo res = null;
        if (StringUtils.isNotBlank(taskId)) {
            boolean gc = dataGcService.gcOneTask(taskId);
            res = new ResultInfo(gc ? ResultCode.SUCCESS : ResultCode.SYSTEM_ERROR);
        }
        log.info("delete task:{} result:{}", taskId, JSON.toJSONString(res));
        return res;
    }

    private void debug(String msg, Object req, ResultInfo res) {
        if (log.isDebugEnabled()) {
            log.debug("{}, req: {} res: {}", msg, JSON.toJSONString(req), JSON.toJSON(res));
        }
    }
}
