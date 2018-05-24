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

import javax.annotation.PostConstruct;
import java.util.concurrent.locks.Lock;
import java.util.concurrent.locks.ReentrantLock;

import com.alibaba.dragonfly.supernode.config.SupernodeProperties;

import com.google.common.util.concurrent.RateLimiter;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.config.ConfigurableBeanFactory;
import org.springframework.context.annotation.Scope;
import org.springframework.stereotype.Component;

/**
 * @author lowzj
 */
@Component
@Scope(value = ConfigurableBeanFactory.SCOPE_PROTOTYPE)
public class PowerRateLimiter {

    private static final Logger logger = LoggerFactory.getLogger(PowerRateLimiter.class);

    @Autowired
    private NetConfigNotification netConfigNotification;

    @Autowired
    private MonitorService monitorService;

    @Autowired
    private SupernodeProperties properties;

    private Lock lock = new ReentrantLock(true);
    private RateLimiter rateLimiter;

    @PostConstruct
    public void init() {
        rateLimiter = RateLimiter
                .create((properties.getTotalLimit() - properties.getSystemNeedRate()) * 1024L * 1024L);
        netConfigNotification.addRateLimiter(rateLimiter);
    }

    public void acquire(int tokens, boolean fair) {
        if (fair) {
            lock.lock();
        }
        try {
            double sleepCost = rateLimiter.acquire(tokens);
            if (sleepCost > 2.0) {
                logger.warn("sleep cost:{} for rate limiter", sleepCost);
            }
            return;
        } finally {
            if (fair) {
                lock.unlock();
            }
        }
    }

    public boolean tryAcquire(int tokens, boolean checkPressure) {
        return (!checkPressure || !monitorService.highPressure()) && rateLimiter.tryAcquire(tokens);
    }
}
