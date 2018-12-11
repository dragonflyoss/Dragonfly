package com.dragonflyoss.dragonfly.supernode.service.preheat;

import java.util.concurrent.ScheduledFuture;

import com.dragonflyoss.dragonfly.supernode.common.domain.PreheatTask;
import org.springframework.stereotype.Component;

/**
 * @author lowzj
 */
@Component
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
        FilePreheatWorker(PreheatTask task, Preheater preheater,
                                 PreheatService service) {
            super(task, preheater, service);
        }

        @Override
        boolean preRun() {
            return false;
        }

        @Override
        ScheduledFuture query() {
            Runnable runnable = new Runnable() {
                @Override
                public void run() {

                }
            };
            return schedule(getTask().getId(), runnable);
        }

        @Override
        void afterRun() {
            scheduledTasks.remove(getTask().getId());
        }
    }
}
