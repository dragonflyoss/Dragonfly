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

import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.locks.Lock;
import java.util.concurrent.locks.ReentrantLock;

import com.alibaba.dragonfly.supernode.config.SupernodeProperties;

import com.google.common.util.concurrent.RateLimiter;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Component;

/**
 * @author lowzj
 */
@Component
public class NetConfigNotification {

    @Autowired
    private SupernodeProperties properties;

    private static final Logger logger = LoggerFactory.getLogger(NetConfigNotification.class);

    private List<RateLimiter> rateLimiters = new ArrayList<RateLimiter>();
    private Lock lock = new ReentrantLock();

    void addRateLimiter(RateLimiter rateLimiter) {
        lock.lock();
        try {
            rateLimiters.add(rateLimiter);
            logger.info("current rate limiter count is {}", rateLimiters.size());
        } finally {
            lock.unlock();
        }
    }

    public void freshNetRate(int rate) {
        if (rate <= 0) {
            logger.error("net rate:{} is illegal", rate);
            return;
        }
        int downRate = rate > properties.getSystemNeedRate() ?
            rate - properties.getSystemNeedRate() : (rate + 1) / 2;
        long rateOnByte = downRate * 1024L * 1024L;
        try {
            boolean updated = false;
            for (RateLimiter rateLimiter : rateLimiters) {
                if (Math.abs(rateLimiter.getRate() - rateOnByte) >= 1024) {
                    rateLimiter.setRate(rateOnByte);
                    updated = true;
                }
            }
            if (updated) {
                logger.info("update net rate to {} MB", rate);
            }
        } catch (Exception e) {
            logger.error("E_freshNetRate", e);
        }
    }

}
