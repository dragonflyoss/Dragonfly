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
package com.dragonflyoss.dragonfly.supernode.common.util;

import java.util.Arrays;
import java.util.LinkedList;
import java.util.List;

import org.apache.commons.lang3.StringUtils;
import org.apache.logging.log4j.util.Strings;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

/**
 * @author zj
 */
public class UrlUtil {
    private static Logger logger = LoggerFactory.getLogger(UrlUtil.class);

    public static String filterParam(String url, String filter) {
        final String sep = "&";

        if (StringUtils.isBlank(filter)) {
            return url;
        }
        String[] rawUrls = url.split("\\?", 2);
        if (rawUrls.length < 2 || StringUtils.isBlank(rawUrls[1])) {
            return url;
        }

        List<String> filters = Arrays.asList(filter.split(sep));
        List<String> params = new LinkedList<>();
        for (String param: rawUrls[1].split(sep)) {
            String[] kv = param.split("=");
            if (!(kv.length >= 1 && filters.contains(kv[0]))) {
                params.add(param);
            }
        }

        return rawUrls[0] + "?" + Strings.join(params, sep.charAt(0));
    }

    public static boolean isValidUrl(String url) {
        if (StringUtils.isBlank(url)) {
            return false;
        }
        return url.matches("https?://.+");
    }

    public static boolean isValidIp(String ip) {
        try {
            if (StringUtils.isNotBlank(ip)) {
                String[] fieldArr = ip.split("\\.");
                if (fieldArr.length == 4) {
                    for (String field : fieldArr) {
                        if (!StringUtils.isNumeric(field)) {
                            return false;
                        }
                    }
                    return true;
                }
            }
        } catch (Exception e) {
            logger.error("ip:{} is illegal", ip);
        }

        return false;
    }

}
