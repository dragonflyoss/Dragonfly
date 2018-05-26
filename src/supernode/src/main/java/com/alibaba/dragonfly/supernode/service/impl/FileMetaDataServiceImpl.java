/*
 * Copyright 1999-2017 Alibaba Group.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package com.alibaba.dragonfly.supernode.service.impl;

import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.List;

import com.alibaba.dragonfly.supernode.common.Constants;
import com.alibaba.dragonfly.supernode.common.domain.FileMetaData;
import com.alibaba.dragonfly.supernode.common.domain.Task;
import com.alibaba.dragonfly.supernode.common.util.DigestUtil;
import com.alibaba.dragonfly.supernode.service.cdn.FileMetaDataService;
import com.alibaba.dragonfly.supernode.service.cdn.util.PathUtil;
import com.alibaba.dragonfly.supernode.service.lock.LockConstants;
import com.alibaba.dragonfly.supernode.service.lock.LockService;
import com.alibaba.fastjson.JSON;

import org.apache.commons.collections.CollectionUtils;
import org.apache.commons.lang3.StringUtils;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;

@Service("fileMetaDataService")
public class FileMetaDataServiceImpl implements FileMetaDataService {

    private static final Logger logger = LoggerFactory.getLogger(FileMetaDataServiceImpl.class);
    @Autowired
    private LockService lockService;

    @Override
    public FileMetaData readFileMetaData(String taskId) {
        String lockName = lockService.getLockName(LockConstants.FILE_META_DATA_LOCK, taskId);
        lockService.lock(lockName);
        try {
            Path metaPath = getMetaPath(taskId);
            FileMetaData metaData = readMetaDataByUnlock(metaPath);
            if (metaData != null && metaData.getPieceSize() == null) {
                metaData.setPieceSize(Constants.DEFAULT_PIECE_SIZE);
            }
            return metaData;
        } catch (Exception e) {
            logger.error("E_readFileMetaData taskId:{}", taskId, e);
        } finally {
            lockService.unlock(lockName);
        }
        return null;
    }

    @Override
    public FileMetaData createMetaData(Task task) throws IOException {
        String taskId = task.getTaskId();
        String lockName = lockService.getLockName(LockConstants.FILE_META_DATA_LOCK, taskId);
        lockService.lock(lockName);
        FileMetaData fileMetaData = new FileMetaData();
        try {
            fileMetaData.setTaskId(taskId);
            fileMetaData.setAccessTime(System.currentTimeMillis());
            fileMetaData.setBizId(task.getBizId());
            fileMetaData.setFileLength(task.getFileLength());
            fileMetaData.setHttpFileLen(task.getHttpFileLen());
            fileMetaData.setPieceSize(task.getPieceSize());
            fileMetaData.setMd5(task.getMd5());
            fileMetaData.setUrl(task.getTaskUrl());

            Path metaPath = getMetaPath(taskId);
            Files.createDirectories(metaPath.getParent());
            Files.write(metaPath, JSON.toJSONBytes(fileMetaData));
        } finally {
            lockService.unlock(lockName);
        }
        return fileMetaData;
    }

    @Override
    public boolean updateAccessTime(String taskId, long accessTime) {
        String lockName = lockService.getLockName(LockConstants.FILE_META_DATA_LOCK, taskId);
        lockService.lock(lockName);
        try {
            Path metaPath = getMetaPath(taskId);
            FileMetaData metaData = readMetaDataByUnlock(metaPath);
            if (metaData != null) {
                metaData.setInterval(accessTime - metaData.getAccessTime());
                if (metaData.getInterval() <= 0) {
                    logger.warn("taskId:{} file hit interval:{}", taskId, metaData.getInterval());
                    metaData.setInterval(0);
                }
                metaData.setAccessTime(accessTime);
                Files.write(metaPath, JSON.toJSONBytes(metaData));
                return true;
            }
        } catch (Exception e) {
            logger.error("E_updateLastModified taskId:{}", taskId, e);
        } finally {
            lockService.unlock(lockName);
        }
        return false;

    }

    @Override
    public boolean updateLastModified(String taskId, long lastModified) {
        String lockName = lockService.getLockName(LockConstants.FILE_META_DATA_LOCK, taskId);
        lockService.lock(lockName);
        try {
            Path metaPath = getMetaPath(taskId);
            FileMetaData metaData = readMetaDataByUnlock(metaPath);
            if (metaData != null) {
                metaData.setLastModified(lastModified);
                Files.write(metaPath, JSON.toJSONBytes(metaData));
                return true;
            }
        } catch (Exception e) {
            logger.error("E_updateLastModified taskId:{}", taskId, e);
        } finally {
            lockService.unlock(lockName);
        }
        return false;
    }

    private Path getMetaPath(String taskId) {
        Path metaPath = PathUtil.getMetaDataPath(taskId);
        return metaPath;
    }

    private FileMetaData readMetaDataByUnlock(Path metaPath) throws IOException {
        if (Files.notExists(metaPath)) {
            return null;
        }
        String metaString = new String(Files.readAllBytes(metaPath), "UTF-8");
        FileMetaData metaData = JSON.parseObject(metaString, FileMetaData.class);
        return metaData;
    }

    @Override
    public boolean updateStatusAndResult(String taskId, boolean finish, boolean success, String realMd5,
        Long fileLength) {
        String lockName = lockService.getLockName(LockConstants.FILE_META_DATA_LOCK, taskId);
        lockService.lock(lockName);
        try {
            Path metaPath = getMetaPath(taskId);
            FileMetaData metaData = readMetaDataByUnlock(metaPath);
            if (metaData != null) {
                metaData.setFinish(finish);

                metaData.setSuccess(success);
                if (success) {
                    if (realMd5 != null) {
                        metaData.setRealMd5(realMd5);
                    }
                    metaData.setFileLength(fileLength);
                }
                Files.write(metaPath, JSON.toJSONBytes(metaData));
                return true;
            }
        } catch (Exception e) {
            logger.error("E_updateStatusAndResult taskId:{}", taskId, e);
        } finally {
            lockService.unlock(lockName);
        }
        return false;
    }

    @Override
    public void writePieceMd5(String taskId, String fileMd5, List<String> pieceMd5s) {
        if (CollectionUtils.isEmpty(pieceMd5s)) {
            logger.error("pieceMd5s is empty for taskId:{}", taskId);
            return;
        }
        pieceMd5s.add(fileMd5);
        String sha1Value = DigestUtil.sha1(pieceMd5s);
        pieceMd5s.add(sha1Value);
        String lockName = lockService.getLockName(LockConstants.FILE_MD5_DATA_LOCK, taskId);
        lockService.lock(lockName);
        try {
            Files.write(PathUtil.getMd5DataPath(taskId), pieceMd5s, StandardCharsets.UTF_8);
        } catch (Exception e) {
            logger.error("write piece md5 error for taskId:{}", taskId);
        } finally {
            lockService.unlock(lockName);
        }
    }

    @Override
    public List<String> readPieceMd5(String taskId, String fileMd5) {
        List<String> pieceMd5s = null;
        String lockName = lockService.getLockName(LockConstants.FILE_MD5_DATA_LOCK, taskId);
        lockService.lock(lockName);
        try {
            Path path = PathUtil.getMd5DataPath(taskId);
            if (Files.exists(path)) {
                pieceMd5s = Files.readAllLines(path, StandardCharsets.UTF_8);
            }
        } catch (Exception e) {
            logger.error("read piece md5 error for taskId:{}", taskId, e);
        } finally {
            lockService.unlock(lockName);
        }
        if (CollectionUtils.isNotEmpty(pieceMd5s)) {
            String sha1Value = pieceMd5s.remove(pieceMd5s.size() - 1);
            if (StringUtils.equalsIgnoreCase(sha1Value, DigestUtil.sha1(pieceMd5s)) && !pieceMd5s.isEmpty()) {
                String realFileMd5 = pieceMd5s.remove(pieceMd5s.size() - 1);
                if (StringUtils.equalsIgnoreCase(realFileMd5, fileMd5)) {
                    return pieceMd5s;
                }
            }
        }
        return null;
    }

}
