package com.dragonflyoss.dragonfly.supernode.service.preheat;

import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.ScheduledFuture;
import java.util.concurrent.TimeUnit;

/**
 * @author lowzj
 */
public abstract class BasePreheater implements Preheater {
    private ScheduledExecutorService scheduler;

    ConcurrentHashMap<String, ScheduledFuture> scheduledTasks;

    public BasePreheater() {
        scheduler = ExecutorUtils.newScheduler(BasePreheater.this.type(), 30);
        scheduledTasks = new ConcurrentHashMap<>();
    }

    ScheduledFuture schedule(String id, Runnable runnable) {
        ScheduledFuture future = scheduler.scheduleAtFixedRate(runnable, 2, 10, TimeUnit.SECONDS);
        scheduledTasks.put(id, future);
        return future;
    }

    @Override
    public void cancel(String id) {
        ScheduledFuture future = scheduledTasks.get(id);
        if (future != null && !future.isDone()) {
            future.cancel(true);
        }
    }
}
