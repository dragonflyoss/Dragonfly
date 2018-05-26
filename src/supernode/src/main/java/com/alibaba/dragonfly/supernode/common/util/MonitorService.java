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
package com.alibaba.dragonfly.supernode.common.util;

import java.io.BufferedReader;
import java.io.InputStreamReader;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.Collections;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

import javax.annotation.PostConstruct;

import com.alibaba.dragonfly.supernode.common.Constants;
import com.alibaba.fastjson.JSON;

import org.apache.commons.collections.CollectionUtils;
import org.apache.commons.lang3.StringUtils;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Service;

@Service
public class MonitorService {
    private static final Logger logger = LoggerFactory.getLogger(MonitorService.class);

    private volatile float cpuLoad;
    private volatile float ioLoad;
    private static int cpuNum = Runtime.getRuntime().availableProcessors();

    @PostConstruct
    public void startMonitor() {
        logger.info("available processors count is {}", cpuNum);

        new Thread(new Runnable() {
            @Override
            public void run() {
                Process outProcess = null;
                String[] fields = null;
                try {
                    final Process process = Runtime.getRuntime().exec("tsar --load --io -s load1,util -l -i 1");
                    Runtime.getRuntime().addShutdownHook(new Thread(new Runnable() {
                        @Override
                        public void run() {
                            process.destroy();
                        }
                    }));
                    outProcess = process;

                    BufferedReader br = new BufferedReader(
                        new InputStreamReader(process.getInputStream(), StandardCharsets.UTF_8), 4 * 1024 * 1024);

                    float tmpValue;
                    Map<String, List<Integer>> fieldIndex = new HashMap<>();
                    for (String line = br.readLine(); line != null; line = br.readLine()) {
                        if (StringUtils.isBlank(line)) {
                            continue;
                        }
                        fields = line.trim().split("\\s+");
                        if (fieldIndex.size() != 2) {
                            fieldIndex.clear();
                            for (int i = 0; i < fields.length; i++) {
                                if (!fillIndex(fieldIndex, fields[i], i, "util")) {
                                    fillIndex(fieldIndex, fields[i], i, "load1");
                                }
                            }
                            if (fieldIndex.size() == 2) {
                                logger.info("monitor field index info:{}", JSON.toJSONString(fieldIndex));
                            }
                        } else {
                            tmpValue = getLoad(fieldIndex, fields, "load1");
                            if (tmpValue >= 0.0f) {
                                cpuLoad = tmpValue;
                            }
                            tmpValue = getLoad(fieldIndex, fields, "util");
                            if (tmpValue >= 0.0f) {
                                ioLoad = tmpValue;
                            }
                            if (Constants.isDebugEnabled()) {
                                logger.info("[DEBUG] load1:{} io util:{}", cpuLoad, ioLoad);
                            }
                        }

                    }
                } catch (Exception e) {
                    cpuLoad = 0.0f;
                    ioLoad = 0.0f;
                    logger.error("process fields:{} error", fields, e);
                    if (outProcess != null) {
                        try {
                            outProcess.destroy();
                        } catch (Exception e1) {
                            logger.error("cannot start monitor again because destroy tsar error", e1);
                        }
                    }

                }
            }
        }).start();

    }

    private boolean fillIndex(Map<String, List<Integer>> fieldIndex, String field, Integer index, String key) {
        if (StringUtils.equalsIgnoreCase(key, field)) {
            List<Integer> indexs = fieldIndex.get(key);
            if (indexs == null) {
                indexs = new ArrayList<>();
                fieldIndex.put(key, indexs);
            }
            indexs.add(index);
            return true;
        }
        return false;
    }

    private float getLoad(Map<String, List<Integer>> fieldIndex, String[] fields, String key) {
        float maxValue = -1.0f;
        int fieldsLen = fields.length;
        List<Integer> indexs = fieldIndex.get(key);
        if (CollectionUtils.isNotEmpty(indexs)) {
            if (Collections.max(indexs) < fieldsLen) {
                for (Integer index : indexs) {
                    String value = fields[index];
                    if (StringUtils.isNotBlank(value) && value.matches("\\d+\\.?\\d*")) {
                        maxValue = maxValue >= Float.parseFloat(value) ? maxValue : Float.parseFloat(value);
                    }
                }
            }
        }
        return maxValue;
    }

    public boolean highPressure() {

        boolean highPressure = ioLoad > 90.0 && cpuLoad * 10 >= cpuNum * 9;

        return highPressure;
    }
}
