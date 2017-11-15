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
import java.util.HashMap;
import java.util.Map;

import javax.servlet.http.HttpServlet;
import javax.servlet.http.HttpServletRequest;
import javax.servlet.http.HttpServletResponse;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class BaseServlet extends HttpServlet {

    private static final Logger logger = LoggerFactory
        .getLogger(BaseServlet.class);

    private static final long serialVersionUID = -7085568621102245916L;

    protected void writeJson(HttpServletResponse resp, String content) {
        writeResponse(resp, "application/json", content);
    }

    protected void writeText(HttpServletResponse resp, String content) {
        writeResponse(resp, "text/html", content);
    }

    protected void writeResponse(HttpServletResponse resp, String type,
        String content) {
        resp.setContentType(type);
        resp.setCharacterEncoding("UTF-8");
        try {
            resp.getWriter().append(content);
            resp.getWriter().flush();
            resp.getWriter().close();
        } catch (IOException e) {
            logger.error("E_writeResponse", e);
        }
    }

    public static Map<String, String> getSingleParameter(
        HttpServletRequest request) {
        Map<String, String[]> valueMap = request.getParameterMap();
        Map<String, String> parameters = new HashMap<String, String>();
        for (String key : valueMap.keySet()) {
            String[] values = valueMap.get(key);
            if (values != null && values.length == 1) {
                parameters.put(key, values[0]);
            }
        }
        return parameters;
    }

    public static String getString(HttpServletRequest request, String key) {
        return request.getParameter(key);
    }

    public static Integer getInteger(HttpServletRequest request, String key) {
        String value = getString(request, key);
        try {
            return Integer.valueOf(value);
        } catch (Exception e) {
            return null;
        }
    }
}
