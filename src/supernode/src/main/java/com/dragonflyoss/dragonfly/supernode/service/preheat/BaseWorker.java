package com.dragonflyoss.dragonfly.supernode.service.preheat;

import com.dragonflyoss.dragonfly.supernode.common.domain.PreheatTask;
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

    @Override
    public void run() {
    }
}
