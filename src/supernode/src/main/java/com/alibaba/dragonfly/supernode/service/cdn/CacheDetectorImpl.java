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
package com.alibaba.dragonfly.supernode.service.cdn;

import java.io.FileInputStream;
import java.io.IOException;
import java.net.MalformedURLException;
import java.nio.ByteBuffer;
import java.nio.ByteOrder;
import java.nio.channels.FileChannel;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.attribute.PosixFilePermission;
import java.nio.file.attribute.PosixFilePermissions;
import java.security.MessageDigest;
import java.util.ArrayList;
import java.util.List;
import java.util.Set;

import com.alibaba.dragonfly.supernode.common.Constants;
import com.alibaba.dragonfly.supernode.common.domain.FileMetaData;
import com.alibaba.dragonfly.supernode.common.domain.Task;
import com.alibaba.dragonfly.supernode.common.domain.dto.CacheResult;
import com.alibaba.dragonfly.supernode.common.enumeration.CdnStatus;
import com.alibaba.dragonfly.supernode.common.enumeration.FromType;
import com.alibaba.dragonfly.supernode.common.enumeration.PeerPieceStatus;
import com.alibaba.dragonfly.supernode.common.util.FileUtil;
import com.alibaba.dragonfly.supernode.common.util.HttpClientUtil;
import com.alibaba.dragonfly.supernode.service.TaskService;
import com.alibaba.dragonfly.supernode.service.cdn.util.PathUtil;
import com.alibaba.dragonfly.supernode.service.lock.LockConstants;
import com.alibaba.dragonfly.supernode.service.lock.LockService;

import org.apache.commons.codec.binary.Hex;
import org.apache.commons.codec.digest.DigestUtils;
import org.apache.commons.collections.CollectionUtils;
import org.apache.commons.lang3.StringUtils;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;

@Service
public class CacheDetectorImpl implements CacheDetector {
    private static final Logger logger = LoggerFactory.getLogger(CacheDetectorImpl.class);

    @Autowired
    private FileMetaDataService fileMetaDataService;
    @Autowired
    private CdnReporter cdnReporter;
    @Autowired
    private LockService lockService;
    @Autowired
    private TaskService taskService;

    @Override
    public CacheResult detectCache(Task task) {
        String taskId = task.getTaskId();
        String lockName = lockService.getLockName(LockConstants.FILE_META_DATA_LOCK, taskId);
        lockService.lock(lockName);
        int breakNum = 0;
        CacheResult cacheResult = new CacheResult();
        FileMetaData fileMetaData = fileMetaDataService.readFileMetaData(taskId);
        try {
            Path downloadPath = PathUtil.getDownloadPath(taskId);
            if (!FileUtil.createSymbolicLink(PathUtil.getUploadPath(taskId), downloadPath)) {
                throw new RuntimeException("create symbolic link fail.");
            }

            if (checkSameFile(task, fileMetaData)) {
                breakNum = parseBreakNum(task, fileMetaData);
            }
            if (breakNum == 0) {
                fileMetaData = resetRepo(task);
            } else {
                fileMetaDataService.updateAccessTime(taskId, System.currentTimeMillis());
            }
        } catch (Exception e) {
            logger.error("detect cache error for taskId:{}", taskId, e);
        } finally {
            lockService.unlock(lockName);
        }
        try {
            reportCache(task, breakNum, cacheResult, fileMetaData);
        } catch (Exception e) {
            logger.error("report cache error for taskId:{}", taskId, e);
            cacheResult.getFileM5().reset();
            cacheResult.setStartPieceNum(0);
        }

        return cacheResult;
    }

    private FileMetaData resetRepo(Task task) throws IOException {
        String taskId = task.getTaskId();
        PathUtil.deleteTaskFiles(taskId, false);
        Set<PosixFilePermission> perms = PosixFilePermissions.fromString("rw-r--r--");
        Files.createFile(PathUtil.getDownloadPath(taskId), PosixFilePermissions.asFileAttribute(perms));
        return fileMetaDataService.createMetaData(task);
    }

    private int parseBreakNum(Task task, FileMetaData fileMetaData) throws MalformedURLException {
        if (fileMetaData.getLastModified() >= 0
            && HttpClientUtil.isExpired(task.getSourceUrl(), fileMetaData.getLastModified(), task.getHeaders())) {
            return 0;
        }

        if (fileMetaData.isFinish()) {
            return fileMetaData.isSuccess() ? -1 : 0;
        }
        if (!HttpClientUtil.isSupportRange(task.getSourceUrl(), task.getHeaders()) || task.getHttpFileLen() <= 0) {
            return 0;
        }
        return parseBreakNumByCheck(task);
    }

    private int parseBreakNumByCheck(Task task) {
        String taskId = task.getTaskId();
        Integer pieceSize = task.getPieceSize();
        long position = 0;
        ByteBuffer bb = generateByteBuffer();
        try (FileInputStream fis = new FileInputStream(PathUtil.getDownloadPathStr(taskId))) {
            FileChannel fc = fis.getChannel();
            long curFileLen = (fc.size() / pieceSize - 1) * pieceSize;
            curFileLen = curFileLen > 0 ? curFileLen : 0;
            int pieceLen;
            while (position < curFileLen) {
                bb.clear();
                bb.limit(4);
                fc.read(bb, position);
                bb.flip();
                pieceLen = bb.getInt() & 0xffffff;
                if (pieceLen > 0) {
                    bb.clear();
                    bb.limit(1);
                    fc.read(bb, pieceLen + position + 4);
                    bb.flip();
                    if (bb.get() == (byte)0x7f) {
                        position += pieceSize;
                        continue;
                    }
                }
                break;
            }
        } catch (Exception e) {
            logger.error("parse break num by check error for taskId:{}", taskId, e);
        }
        return (int)(position / pieceSize);
    }

    @Override
    public boolean checkSameFile(Task task, FileMetaData fileMetaData) {
        if (fileMetaData != null && task != null) {
            String taskId = task.getTaskId();
            if (Files.exists(PathUtil.getDownloadPath(taskId))) {
                if (fileMetaData.getPieceSize().equals(task.getPieceSize())) {
                    if (StringUtils.equals(fileMetaData.getTaskId(), task.getTaskId())
                        && StringUtils.equals(fileMetaData.getUrl(), task.getTaskUrl())) {
                        if (StringUtils.isNotBlank(task.getMd5())) {
                            return StringUtils.equals(task.getMd5(), fileMetaData.getMd5());
                        } else if (StringUtils.isBlank(fileMetaData.getMd5())) {
                            return StringUtils.equals(task.getBizId() == null ? "" : task.getBizId(),
                                fileMetaData.getBizId() == null ? "" : fileMetaData.getBizId());
                        }
                    }
                }
            }
        }
        return false;
    }

    private void reportCache(Task task, Long fileLength, String fileMd5,
        List<String> pieceMd5s)
        throws IOException {
        String taskId = task.getTaskId();
        int size = pieceMd5s.size();
        if (size > 0) {
            for (int pieceNum = 0; pieceNum < size; pieceNum++) {
                cdnReporter.reportPieceStatus(taskId, pieceNum, pieceMd5s.get(pieceNum),
                    PeerPieceStatus.SUCCESS, FromType.LOCAL.type());
            }
            cdnReporter.reportTaskStatus(taskId, CdnStatus.SUCCESS, fileMd5, fileLength, FromType.LOCAL.type());
        }
    }

    private void reportCache(Task task, int breakNum, CacheResult cacheResult, FileMetaData metaData)
        throws IOException {

        /*
         * cache not hit
         */
        if (breakNum == 0) {
            return;
        }

        if (processCacheByQuick(task, breakNum, cacheResult, metaData)) {
            return;
        }
        processCacheByChannel(breakNum, cacheResult, metaData, task);
    }

    /**
     * @param breakNum
     * @param cacheResult
     * @param metaData
     * @param task
     */
    private void processCacheByChannel(int breakNum, CacheResult cacheResult, FileMetaData metaData,
        Task task) {
        String taskId = task.getTaskId();
        Integer pieceSize = task.getPieceSize();
        try (FileInputStream fis = new FileInputStream(PathUtil.getDownloadPath(taskId).toFile());
            FileChannel fc = fis.getChannel()) {

            List<String> pieceMd5s = new ArrayList<>();
            MessageDigest pieceMd5 = DigestUtils.getMd5Digest();
            MessageDigest fileM5 = cacheResult.getFileM5();
            if (breakNum == -1 && StringUtils.isNotBlank(metaData.getRealMd5())) {
                fileM5 = null;
            }

            ByteBuffer bb = generateByteBuffer();
            String pieceMd5Value;
            long curFileLen = fc.size();
            int curPieceTotal =
                breakNum > 0 ? breakNum : (int)((curFileLen + pieceSize - 1) / pieceSize);
            int pieceHead, pieceLen;

            for (int pieceNum = 0; pieceNum < curPieceTotal; pieceNum++) {
                fc.position(pieceNum * (long)pieceSize);
                bb.limit(Constants.PIECE_HEAD_SIZE);
                fc.read(bb);
                bb.flip();
                pieceHead = bb.getInt();
                pieceLen = pieceHead & 0xffffff;
                bb.limit(pieceLen + Constants.PIECE_WRAP_SIZE);
                fc.read(bb);
                bb.flip();
                pieceMd5.update(bb);
                pieceMd5Value = Hex.encodeHexString(pieceMd5.digest()) + ":" + bb.limit();
                cdnReporter.reportPieceStatus(taskId, pieceNum, pieceMd5Value, PeerPieceStatus.SUCCESS,
                    FromType.LOCAL.type());
                pieceMd5s.add(pieceMd5Value);
                pieceMd5.reset();

                if (fileM5 != null) {
                    bb.flip();
                    bb.limit(bb.limit() - 1);
                    bb.position(Constants.PIECE_HEAD_SIZE);
                    fileM5.update(bb);

                }
                bb.clear();
            }
            if (breakNum == -1) {
                String fileMd5Value = metaData.getRealMd5();
                if (StringUtils.isBlank(fileMd5Value)) {
                    fileMd5Value = Hex.encodeHexString(fileM5.digest());
                    fileMetaDataService.updateStatusAndResult(taskId, true, true, fileMd5Value, curFileLen);
                }
                cdnReporter.reportTaskStatus(taskId, CdnStatus.SUCCESS, fileMd5Value, curFileLen,
                    FromType.LOCAL.type());
                fileMetaDataService.writePieceMd5(taskId, fileMd5Value, pieceMd5s);
            }
            cacheResult.setStartPieceNum(breakNum);

        } catch (Exception e) {
            throw new RuntimeException("report cache by channel error for taskId:" + taskId, e);
        }
    }

    private boolean processCacheByQuick(Task task, int breakNum, CacheResult cacheResult,
        FileMetaData metaData) throws IOException {
        List<String> pieceMd5s;
        if (breakNum == -1) {
            if (StringUtils.isNotBlank(metaData.getRealMd5())) {
                pieceMd5s = getFullPieceMd5s(task, metaData);
                Long fileLen = metaData.getFileLength();
                if (CollectionUtils.isNotEmpty(pieceMd5s)) {
                    reportCache(task, fileLen, metaData.getRealMd5(), pieceMd5s);
                    cacheResult.setStartPieceNum(breakNum);
                    return true;
                }
            }
        }
        return false;
    }

    private ByteBuffer generateByteBuffer() {
        ByteBuffer bb = ByteBuffer.allocate(Constants.PIECE_SIZE_LIMIT);
        bb.order(ByteOrder.BIG_ENDIAN);
        return bb;
    }

    private List<String> getFullPieceMd5s(Task task, FileMetaData metaData) {
        List<String> pieceMd5s = taskService.getFullPieceMd5sByTask(task);
        if (CollectionUtils.isEmpty(pieceMd5s)) {
            pieceMd5s = fileMetaDataService.readPieceMd5(task.getTaskId(), metaData.getRealMd5());
        }
        return pieceMd5s;
    }

}
