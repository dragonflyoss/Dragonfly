package com.dragonflyoss.dragonfly.supernode.service.repair;

import com.dragonflyoss.dragonfly.supernode.common.domain.ClientErrorInfo;
import com.dragonflyoss.dragonfly.supernode.common.enumeration.ClientErrorType;

/**
 * @author lowzj
 */
public interface ClientErrorHandler {
    boolean isErrorType(ClientErrorType type);

    void handle(ClientErrorInfo info);
}
