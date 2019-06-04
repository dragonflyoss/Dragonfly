package com.dragonflyoss.dragonfly.supernode.service.repair;

import com.dragonflyoss.dragonfly.supernode.common.domain.ClientErrorInfo;

/**
 * @author lowzj
 */
public interface ClientErrorHandleService {
    /**
     * handle the client error
     *
     * @param info the information of client error
     */
    void handleClientError(ClientErrorInfo info);
}
