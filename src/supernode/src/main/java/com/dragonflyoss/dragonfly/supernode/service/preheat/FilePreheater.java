package com.dragonflyoss.dragonfly.supernode.service.preheat;

import java.util.concurrent.ScheduledFuture;

import com.dragonflyoss.dragonfly.supernode.common.domain.PreheatTask;
import com.dragonflyoss.dragonfly.supernode.common.enumeration.PreheatTaskStatus;
import com.dragonflyoss.dragonfly.supernode.common.exception.PreheatException;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Component;

/**
 * @author lowzj
 */
@Component
@Slf4j
public class FilePreheater extends BasePreheater {
    private static final String TYPE = "file";

    @Override
    public String type() {
        return TYPE;
    }

    @Override
    public BaseWorker newWorker(PreheatTask task, PreheatService service) {
        return new FilePreheatWorker(task, this, service);
    }

    class FilePreheatWorker extends BaseWorker {
        private PreheatProcess process;

        FilePreheatWorker(PreheatTask task, Preheater preheater,
                                 PreheatService service) {
            super(task, preheater, service);
        }

        @Override
        boolean preRun() {
            try {
                PreheatTask task = getTask();
                task.setStatus(PreheatTaskStatus.RUNNING);
                getService().update(task.getId(), task);
                this.process = getService().executePreheat(task);
                return true;
            } catch (PreheatException e) {
                failed(e.getMessage());
            }
            return false;
        }

        @Override
        ScheduledFuture query() {
            Runnable runnable = new Runnable() {
                private int count = 0;
                @Override
                public void run() {
                    PreheatTask task = getTask();
                    log.info("query preheat task:{} status:{} count:{}",
                        task.getId(), task.getStatus(), count++);

                    if (task.getFinishTime() > 0) {
                        cancel(task.getId());
                        return;
                    }
                    if (process == null) {
                        // needn't to preheat because the task has been completed
                        succeed();
                        return;
                    }
                    if (!process.isAlive()) {
                        int code = process.exitValue();
                        if (code == 0) {
                            succeed();
                            cancel(task.getId());
                        } else {
                            failed("dfget code:" + code + " out:" + process.getError());
                            cancel(task.getId());
                        }
                    }
                }
            };
            return schedule(getTask().getId(), runnable);
        }

        @Override
        void afterRun() {
            if (process != null && process.isAlive()) {
                process.destroy();
            }
            super.afterRun();
        }
    }
}
