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

package com.dragonflyoss.dragonfly.supernode.rest.controller;

import javax.servlet.http.HttpServletRequest;
import java.util.LinkedList;
import java.util.List;
import java.util.concurrent.RejectedExecutionException;

import com.alibaba.fastjson.JSON;

import com.dragonflyoss.dragonfly.supernode.common.domain.PreheatTask;
import com.dragonflyoss.dragonfly.supernode.common.exception.PreheatException;
import com.dragonflyoss.dragonfly.supernode.rest.request.PreheatCreateRequest;
import com.dragonflyoss.dragonfly.supernode.rest.response.ErrorResponse;
import com.dragonflyoss.dragonfly.supernode.rest.response.PreheatCreateResponse;
import com.dragonflyoss.dragonfly.supernode.rest.response.PreheatQueryResponse;
import com.dragonflyoss.dragonfly.supernode.service.preheat.PreheatService;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.HttpRequestMethodNotSupportedException;
import org.springframework.web.bind.annotation.DeleteMapping;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.ResponseBody;
import org.springframework.web.bind.annotation.ResponseStatus;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.method.annotation.MethodArgumentTypeMismatchException;

import static org.springframework.http.HttpStatus.BAD_REQUEST;

/**
 * Created on 2018/11/02
 *
 * @author lowzj
 */
@RestController
@Slf4j
public class PreheatController {
    @Autowired
    protected HttpServletRequest request;

    @Autowired
    protected PreheatService preheatService;

    @PostMapping("/preheats")
    public ResponseEntity createPreheatTask(@RequestBody PreheatCreateRequest request) {
        ResponseEntity response;

        PreheatTask task = new PreheatTask();
        task.setType(request.getType());
        task.setUrl(request.getUrl());
        task.setFilter(request.getFilter());
        task.setIdentifier(request.getIdentifier());
        task.setHeaders(request.getHeaders());
        try {
            String id = preheatService.createPreheatTask(task);
            PreheatCreateResponse res = new PreheatCreateResponse();
            res.setId(id);
            response = new ResponseEntity<>(res, HttpStatus.OK);
        } catch (PreheatException e) {
            log.error("createPreheatTask req:{}", JSON.toJSONString(request), e);
            response = new ResponseEntity<>(new ErrorResponse(e.getCode(), e.getMessage()),
                HttpStatus.BAD_REQUEST);
        } catch (RejectedExecutionException e) {
            log.error("createPreheatTask req:{}", JSON.toJSONString(request), e);
            response = new ResponseEntity<>(new ErrorResponse(500, e.getMessage()),
                HttpStatus.INTERNAL_SERVER_ERROR);
        }
        log.info("createPreheatTask, req:{}, res:{}", JSON.toJSONString(request),
            JSON.toJSONString(response));
        return response;
    }

    @GetMapping("/preheats")
    public ResponseEntity getPreheatTasks() {
        List<PreheatTask> tasks = preheatService.getAll();
        List<PreheatQueryResponse> res = new LinkedList<>();
        for (PreheatTask task : tasks) {
            if (task != null) {
                res.add(new PreheatQueryResponse(task));
            }
        }
        return new ResponseEntity<>(res, HttpStatus.OK);
    }

    @GetMapping("/preheats/{id}")
    public ResponseEntity queryPreheatTask(@PathVariable("id") String id) {
        PreheatTask task = preheatService.get(id);
        if (task == null) {
            return new ResponseEntity<>(new ErrorResponse(HttpStatus.NOT_FOUND.value(), id + " doesn't exists"),
                HttpStatus.NOT_FOUND);
        }
        PreheatQueryResponse res = new PreheatQueryResponse(task);
        return new ResponseEntity<>(res, HttpStatus.OK);
    }

    @DeleteMapping("/preheats/{id}")
    public ResponseEntity deletePreheatTask(@PathVariable("id") String id) {
        boolean res = preheatService.delete(id);
        return new ResponseEntity<>(res, HttpStatus.OK);
    }

    //--------------------------------------------------------------------------
    // exception handlers in class

    @ExceptionHandler({MethodArgumentTypeMismatchException.class})
    @ResponseStatus(BAD_REQUEST)
    @ResponseBody
    public ErrorResponse handleBufferClientException(RuntimeException ex) {
        String errMsg = BAD_REQUEST.getReasonPhrase() + ", " + request.getMethod() + " "
            + request.getRequestURI() + ", error: " + ex.getMessage();
        log.error(errMsg, ex);
        return new ErrorResponse(HttpStatus.BAD_REQUEST.value(), errMsg);
    }

    @ExceptionHandler(HttpRequestMethodNotSupportedException.class)
    @ResponseStatus(value = HttpStatus.NOT_FOUND)
    public ErrorResponse handleHttpRequestMethodNotSupportedException(HttpRequestMethodNotSupportedException ex) {
        return new ErrorResponse(HttpStatus.NOT_FOUND.value(), HttpStatus.NOT_FOUND.getReasonPhrase());
    }

}
