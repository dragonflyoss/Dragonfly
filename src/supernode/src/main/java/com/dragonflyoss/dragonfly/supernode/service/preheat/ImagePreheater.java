package com.dragonflyoss.dragonfly.supernode.service.preheat;

import com.dragonflyoss.dragonfly.supernode.common.domain.PreheatTask;
import org.springframework.stereotype.Component;

/**
 * @author lowzj
 */
@Component
public class ImagePreheater extends BasePreheater {
    @Override
    public String type() {
        return "image";
    }

    @Override
    public BaseWorker newWorker(PreheatTask task, PreheatService service) {
        return new ImagePreheatWorker(task, this, service);
    }

    static class ImagePreheatWorker extends BaseWorker {
        ImagePreheatWorker(PreheatTask task, Preheater preheater,
                                  PreheatService service) {
            super(task, preheater, service);
        }
    }
}
