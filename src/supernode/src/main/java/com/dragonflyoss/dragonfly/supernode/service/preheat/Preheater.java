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
     * Executing the preheat task
     *
     * @param task the preheat task base information
     */
    void execute(PreheatTask task);
}
