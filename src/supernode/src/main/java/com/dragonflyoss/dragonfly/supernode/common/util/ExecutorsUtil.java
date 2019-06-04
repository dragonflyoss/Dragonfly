package com.dragonflyoss.dragonfly.supernode.common.util;

import java.util.concurrent.ExecutorService;
import java.util.concurrent.LinkedBlockingQueue;
import java.util.concurrent.ThreadFactory;
import java.util.concurrent.ThreadPoolExecutor;
import java.util.concurrent.TimeUnit;

import org.apache.commons.lang3.concurrent.BasicThreadFactory;

/**
 * @author lowzj
 */
public final class ExecutorsUtil {
        public static ExecutorService newThreadPool(String name, int corePoolSize, int maxPoolSize) {
            return new ThreadPoolExecutor(corePoolSize, maxPoolSize, 60L, TimeUnit.SECONDS,
                new LinkedBlockingQueue<Runnable>(), newThreadFactory(name));
        }

        private static ThreadFactory newThreadFactory(String name) {
            return new BasicThreadFactory.Builder()
                .namingPattern(name + "-%d")
                .daemon(true)
                .build();
        }
}
