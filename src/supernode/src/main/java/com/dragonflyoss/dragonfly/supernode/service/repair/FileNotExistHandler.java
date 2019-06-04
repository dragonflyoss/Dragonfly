package com.dragonflyoss.dragonfly.supernode.service.repair;

import java.nio.file.Path;

import com.dragonflyoss.dragonfly.supernode.common.domain.ClientErrorInfo;
import com.dragonflyoss.dragonfly.supernode.common.enumeration.ClientErrorType;
import com.dragonflyoss.dragonfly.supernode.common.util.FileUtil;
import com.dragonflyoss.dragonfly.supernode.service.cdn.util.PathUtil;
import com.dragonflyoss.dragonfly.supernode.service.timer.DataGcService;

import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Component;

/**
 * @author lowzj
 */
@Component
@Slf4j
public class FileNotExistHandler extends BaseClientErrorHandler {
    @Autowired
    private DataGcService dataGcService;

    @Override
    ClientErrorType errorType() {
        return ClientErrorType.FILE_NOT_EXIST;
    }

    @Override
    void doHandle(final ClientErrorInfo info) {
        Path downloadPath = PathUtil.getDownloadPath(info.getTaskId());
        if (!downloadPath.toFile().exists()) {
            boolean res = dataGcService.gcOneTask(info.getTaskId());
            log.info("taskId:{} data file doesn't exist, task gc:{}", info.getTaskId(), res);
            return;
        }

        Path uploadPath = PathUtil.getUploadPath(info.getTaskId());
        if (!uploadPath.toFile().exists()) {
            boolean res = FileUtil.createSymbolicLink(uploadPath, downloadPath);
            log.info("taskId:{} upload file doesn't exist, link again:{}", info.getTaskId(), res);
        }
    }
}
