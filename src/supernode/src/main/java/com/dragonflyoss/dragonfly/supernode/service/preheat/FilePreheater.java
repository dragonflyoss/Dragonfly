package com.dragonflyoss.dragonfly.supernode.service.preheat;

import com.dragonflyoss.dragonfly.supernode.common.domain.PreheatTask;
import org.springframework.stereotype.Component;

/**
 * @author lowzj
 */
@Component
public class FilePreheater extends BasePreheater {
    @Override
    public String type() {
        return "file";
    }

    @Override
    public BaseWorker newWorker(PreheatTask task, PreheatService service) {
        return null;
    }
}
