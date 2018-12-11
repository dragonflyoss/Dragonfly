package com.dragonflyoss.dragonfly.supernode.service.preheat;

import java.util.concurrent.ExecutorService;
import java.util.concurrent.LinkedBlockingQueue;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.ScheduledThreadPoolExecutor;
import java.util.concurrent.ThreadFactory;
import java.util.concurrent.ThreadPoolExecutor;
import java.util.concurrent.TimeUnit;

import org.apache.commons.lang3.concurrent.BasicThreadFactory;

/**
 * @author lowzj
 */
final class ExecutorUtils {
    static ScheduledExecutorService newScheduler(String name, int corePoolSize) {
        return new ScheduledThreadPoolExecutor(corePoolSize, newThreadFactory(name));
    }

    static ExecutorService newThreadPool(String name, int corePoolSize, int maxPoolSize) {
        return new ThreadPoolExecutor(corePoolSize, maxPoolSize,
            60L, TimeUnit.SECONDS,
            new LinkedBlockingQueue<Runnable>(), newThreadFactory(name));
    }

    private static ThreadFactory newThreadFactory(String name) {
        return new BasicThreadFactory.Builder()
            .namingPattern(name + "-%d")
            .daemon(true)
            .build();
    }
}

