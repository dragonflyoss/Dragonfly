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
package com.alibaba.dragonfly.supernode.common.view;

import java.util.HashMap;
import java.util.Map;

import com.fasterxml.jackson.annotation.JsonInclude;

@JsonInclude(JsonInclude.Include.NON_NULL)
public class ResultInfo {

    private int code;
    private String msg;
    private Object data;
    private Map<String, String> addition;

    public ResultInfo() {
        this.code = ResultCode.SUCCESS;
    }

    public ResultInfo(int code, String message, Object data) {
        this.code = code;
        this.msg = message;
        this.data = data;
    }

    public ResultInfo(int code, Object data) {
        this.code = code;
        this.data = data;
    }

    public ResultInfo(int code) {
        this.code = code;
        this.msg = ResultCode.getDesc(code);
    }

    public Map<String, String> getAddition() {
        return addition;
    }

    public void setAddition(Map<String, String> addition) {
        this.addition = addition;
    }

    public void addAddition(String key, String value) {
        if (this.addition == null) {
            this.addition = new HashMap<>();
        }
        this.addition.put(key, value);
    }

    public boolean successCode() {
        return code == ResultCode.SUCCESS;
    }

    public int getCode() {
        return code;
    }

    public String getMsg() {
        return msg;
    }

    public Object getData() {
        return data;
    }

    public ResultInfo withCode(int code) {
        this.code = code;
        return this;
    }

    public ResultInfo withMsg(String msg) {
        this.msg = msg;
        return this;
    }

    public ResultInfo withData(Object data) {
        this.data = data;
        return this;
    }

    public void setCode(int code) {
        this.code = code;
    }

    public void setMsg(String msg) {
        this.msg = msg;
    }

    public void setData(Object data) {
        this.data = data;
    }

}
