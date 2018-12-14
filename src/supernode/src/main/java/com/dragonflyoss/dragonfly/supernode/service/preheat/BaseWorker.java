package com.dragonflyoss.dragonfly.supernode.service.preheat;

import java.util.concurrent.CancellationException;
import java.util.concurrent.ExecutionException;
import java.util.concurrent.ScheduledFuture;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.TimeoutException;

import com.dragonflyoss.dragonfly.supernode.common.domain.PreheatTask;
import com.dragonflyoss.dragonfly.supernode.common.enumeration.PreheatTaskStatus;
import lombok.AllArgsConstructor;
import lombok.Data;

/**
 * @author lowzj
 */
@Data
@AllArgsConstructor
public abstract class BaseWorker implements Runnable {
    private PreheatTask task;
    private Preheater preheater;
    private PreheatService service;

    private static final long TIMEOUT = 30 * 60;

    @Override
    public void run() {
        if (preRun()) {
            ScheduledFuture result = query();
            try {
                result.get(TIMEOUT, TimeUnit.SECONDS);
            } catch (CancellationException e) {
                // canceled by user
                if (task.getFinishTime() <= 0) {
                   failed(e.getMessage());
                }
            } catch (InterruptedException | ExecutionException | TimeoutException e) {
                failed(e.getMessage());
            }
        }
        afterRun();
    }

    /**
     * pre-work before preheat the task
     *
     * @return true if run successfully
     */
    abstract boolean preRun();

    /**
     * query the preheat task whether is finished
     *
     * @return used for the caller to wait for executing thread finished
     */
    abstract ScheduledFuture query();

    /**
     * the operation of after running
     */
    abstract void afterRun();

    void succeed() {
        task.setFinishTime(System.currentTimeMillis());
        task.setStatus(PreheatTaskStatus.SUCCESS);
        service.update(task.getId(), task);
    }

    void failed(String errMsg) {
        task.setStatus(PreheatTaskStatus.FAILED);
        task.setFinishTime(System.currentTimeMillis());
        task.setErrorMsg(errMsg);
        service.update(task.getId(), task);
    }
}
