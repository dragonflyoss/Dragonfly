package com.dragonflyoss.dragonfly.supernode.service.repair;

import java.util.LinkedList;
import java.util.List;

import com.dragonflyoss.dragonfly.supernode.common.domain.ClientErrorInfo;

import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;

/**
 * @author lowzj
 */
@Service("clientErrorHandleService")
@Slf4j
public class ClientErrorHandleServiceImpl implements ClientErrorHandleService {

    @Autowired
    private List<ClientErrorHandler> handlers = new LinkedList<>();

    @Override
    public void handleClientError(ClientErrorInfo info) {
        if (handlers == null || info == null || info.getErrorType() == null) {
            return;
        }

        for (ClientErrorHandler handler : handlers) {
            if (handler != null && handler.isErrorType(info.getErrorType())) {
                handler.handle(info);
                break;
            }
        }
    }
}
