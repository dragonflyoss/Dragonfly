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
    public void execute(PreheatTask task) {

    }
}
