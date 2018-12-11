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

package com.dragonflyoss.dragonfly.supernode.config;

import javax.net.ssl.SSLContext;
import java.security.KeyManagementException;
import java.security.KeyStoreException;
import java.security.NoSuchAlgorithmException;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.ScheduledThreadPoolExecutor;

import com.google.common.util.concurrent.ThreadFactoryBuilder;
import org.apache.http.conn.ssl.SSLConnectionSocketFactory;
import org.apache.http.conn.ssl.TrustAllStrategy;
import org.apache.http.impl.client.CloseableHttpClient;
import org.apache.http.impl.client.HttpClients;
import org.apache.http.ssl.SSLContextBuilder;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.context.properties.EnableConfigurationProperties;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.http.client.HttpComponentsClientHttpRequestFactory;
import org.springframework.scheduling.annotation.EnableScheduling;
import org.springframework.scheduling.annotation.SchedulingConfigurer;
import org.springframework.scheduling.config.ScheduledTaskRegistrar;
import org.springframework.web.client.RestTemplate;

/**
 * Created on 2018/05/23
 *
 * @author lowzj
 */
@Configuration
@EnableConfigurationProperties(SupernodeProperties.class)
public class SupernodeAutoConfiguration {

    @Bean
    public RestTemplate restTemplate() throws NoSuchAlgorithmException, KeyManagementException, KeyStoreException {
        SSLContext sc = new SSLContextBuilder().loadTrustMaterial(null, new TrustAllStrategy()).build();
        CloseableHttpClient httpClient = HttpClients.custom()
            .setSSLSocketFactory(new SSLConnectionSocketFactory(sc))
            .build();
        return new RestTemplate(new HttpComponentsClientHttpRequestFactory(httpClient));
    }

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
