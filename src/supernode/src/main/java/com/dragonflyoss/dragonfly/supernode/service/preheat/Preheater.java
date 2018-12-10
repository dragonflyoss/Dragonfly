package com.dragonflyoss.dragonfly.supernode.service.preheat;

import com.dragonflyoss.dragonfly.supernode.common.domain.PreheatTask;

/**
 * @author lowzj
 */
public interface Preheater {
    /**
     * The type of this preheater
     *
     * @return String
     */
    String type();

    /**
     * Create a worker to preheat the task.
     *
     * @param task preheat task information
     * @param service preheat service
     * @return BaseWorker
     */
    BaseWorker newWorker(PreheatTask task, PreheatService service);
}
