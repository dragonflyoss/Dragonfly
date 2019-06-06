package com.dragonflyoss.dragonfly.supernode.service.repair;

import java.util.concurrent.BlockingDeque;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.LinkedBlockingDeque;
import java.util.concurrent.RejectedExecutionException;
import java.util.concurrent.TimeUnit;

import com.alibaba.fastjson.JSON;

import com.dragonflyoss.dragonfly.supernode.common.Constants;
import com.dragonflyoss.dragonfly.supernode.common.domain.ClientErrorInfo;
import com.dragonflyoss.dragonfly.supernode.common.enumeration.ClientErrorType;
import com.dragonflyoss.dragonfly.supernode.common.util.ExecutorsUtil;

import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;

/**
 * The client error will be handled immediately. In the process of handling,
 * and within 5 seconds after completing handling, the subsequent client errors
 * with the same taskId and error type will be ignored.
 *
 * @author lowzj
 */
@Slf4j
public abstract class BaseClientErrorHandler implements ClientErrorHandler {
    private static final ExecutorService executor = ExecutorsUtil
        .newThreadPool("client-error", 5, 100);

    /**
     * map(taskId, addTime)
     */
    private ConcurrentHashMap<String, Integer> running = new ConcurrentHashMap<>();

    private Remover remover = new Remover();

    public BaseClientErrorHandler() {
        executor.execute(remover);
    }

    abstract ClientErrorType errorType();
    abstract void doHandle(final ClientErrorInfo info);

    @Override
    public boolean isErrorType(ClientErrorType type) {
        return errorType().equals(type);
    }

    @Override
    public void handle(final ClientErrorInfo info) {
        if (isInvalidInfo(info)) {
            return;
        }
        Integer pre = running.putIfAbsent(info.getTaskId(), currentSeconds());
        if (pre != null) {
            return;
        }
        try {
            executor.execute(new HandlerRunner(info));
        } catch (RejectedExecutionException e) {
            log.error("reject to execute: {}", JSON.toJSONString(info), e);
            removeTask(info.getTaskId());
        }
    }

    boolean isInvalidInfo(ClientErrorInfo info) {
        return info == null || !isErrorType(info.getErrorType())
            || StringUtils.isAnyBlank(info.getTaskId(), info.getDstCid())
            // ignore the error that isn't caused by downloading from supernode
            || !info.getDstCid().startsWith(Constants.CDN_NODE_PREFIX);
    }

    private void removeTask(String taskId) {
        if (StringUtils.isNoneBlank(taskId)) {
            remover.add(taskId, true);
        }
    }

    private class HandlerRunner implements Runnable {
        private final ClientErrorInfo info;

        HandlerRunner(ClientErrorInfo info) {
            this.info = info;
        }

        @Override
        public void run() {
            try {
                log.info("start to handle client error: {}", JSON.toJSONString(info));
                doHandle(info);
            } catch (Exception e) {
                log.error("failed to handle client error: {}", JSON.toJSONString(info), e);
            } finally {
                removeTask(info.getTaskId());
            }
        }
    }

    private class Remover implements Runnable {
        private static final int EXPIRED = 5;
        private static final int POLL_INTERVAL = 2;
        private static final int MAX_SIZE = 100;

        private BlockingDeque<String> queue = new LinkedBlockingDeque<>();
        private boolean interrupted = false;

        @Override
        public void run() {
            while (true) {
                try {
                    if (queue.size() >= MAX_SIZE) {
                        for (int i = 0; i < MAX_SIZE / 2; i++) {
                            remove(queue.takeFirst(), true);
                        }
                    }
                    String taskId = queue.pollFirst(POLL_INTERVAL, TimeUnit.SECONDS);
                    if (taskId == null) {
                        continue;
                    }
                    int expires = remove(taskId, false);
                    if (expires > 0) {
                        add(taskId, false);
                        Thread.sleep(expires * 1000);
                    }
                } catch (InterruptedException e) {
                    interrupted = true;
                    log.warn("remover of {} is interrupted", errorType(), e);
                    break;
                }
            }
        }

        public void add(String taskId, boolean last) {
            if (interrupted) {
                running.remove(taskId);
                return;
            }
            try {
                if (last) {
                    queue.addLast(taskId);
                } else {
                    queue.addFirst(taskId);
                }
            } catch (Throwable t) {
                running.remove(taskId);
                log.warn("failed to add into remover, directly remove taskId:{}", taskId, t);
            }
        }

        private int remove(String taskId, boolean force) {
            Integer addTime = running.get(taskId);
            if (force || addTime == null) {
                running.remove(taskId);
                return 0;
            }

            int expires = addTime + EXPIRED - currentSeconds();
            if (expires <= 0) {
                running.remove(taskId);
            }
            return expires;
        }
    }

    private int currentSeconds() {
        return (int)(System.currentTimeMillis() / 1000);
    }
}
