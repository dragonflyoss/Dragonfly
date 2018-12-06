package com.dragonflyoss.dragonfly.supernode.service.preheat;

import java.util.List;

import com.dragonflyoss.dragonfly.supernode.common.domain.PreheatTask;
import com.dragonflyoss.dragonfly.supernode.common.exception.PreheatException;

/**
 * @author lowzj
 */
public interface PreheatService {
    /**
     * Get detailed preheat task information
     *
     * @param id preheat task id
     * @return preheat task information
     */
    PreheatTask get(String id);

    /**
     * Get all preheat tasks
     *
     * @return list of preheat tasks
     */
    List<PreheatTask> getAll();

    /**
     * Preheat a task
     *
     * @param task the preheat task information
     * @return preheat task's id
     * @throws PreheatException exception
     */
    String preheat(PreheatTask task) throws PreheatException;
}
