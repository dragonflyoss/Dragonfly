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

package com.alibaba.dragonfly.supernode.config;

import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.ScheduledThreadPoolExecutor;

import com.google.common.util.concurrent.ThreadFactoryBuilder;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.context.properties.EnableConfigurationProperties;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.scheduling.annotation.EnableScheduling;
import org.springframework.scheduling.annotation.SchedulingConfigurer;
import org.springframework.scheduling.config.ScheduledTaskRegistrar;

/**
 * Created on 2018/05/23
 *
 * @author lowzj
 */
@Configuration
@EnableConfigurationProperties(SupernodeProperties.class)
public class SupernodeAutoConfiguration {

    @Configuration
    @EnableScheduling
    public static class SpringSchedulerConfig implements SchedulingConfigurer {

        @Autowired
        private SupernodeProperties properties;

        @Override
        public void configureTasks(ScheduledTaskRegistrar taskRegistrar) {
            taskRegistrar.setScheduler(taskExecutor());
        }

        @Bean(destroyMethod="shutdown")
        public ScheduledExecutorService taskExecutor() {
            return new ScheduledThreadPoolExecutor(properties.getSchedulerCorePoolSize(),
                new ThreadFactoryBuilder().setNameFormat("spring-%d").setDaemon(true).build());
        }
    }

}
