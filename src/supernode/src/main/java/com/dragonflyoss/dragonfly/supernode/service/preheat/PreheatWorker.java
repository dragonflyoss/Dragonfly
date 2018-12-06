package com.dragonflyoss.dragonfly.supernode.service.preheat;

import com.dragonflyoss.dragonfly.supernode.common.domain.PreheatTask;
import lombok.AllArgsConstructor;
import lombok.Data;

/**
 * @author lowzj
 */
@Data
@AllArgsConstructor
public class PreheatWorker implements Runnable {
    private PreheatTask task;
    private Preheater preheater;
    private PreheatService service;

    @Override
    public void run() {
        preheater.execute(task);
    }
}
