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

import java.io.IOException;
import java.io.InputStream;
import java.net.HttpURLConnection;
import java.net.URL;
import java.nio.ByteBuffer;
import java.security.MessageDigest;
import java.util.concurrent.ArrayBlockingQueue;
import java.util.concurrent.BlockingQueue;
import java.util.concurrent.Callable;
import java.util.concurrent.FutureTask;

import com.alibaba.dragonfly.supernode.common.CdnConstants;
import com.alibaba.dragonfly.supernode.common.Constants;
import com.alibaba.dragonfly.supernode.common.domain.DownloadMetaData;
import com.alibaba.dragonfly.supernode.common.domain.Task;
import com.alibaba.dragonfly.supernode.common.domain.dto.CacheResult;
import com.alibaba.dragonfly.supernode.common.enumeration.CdnStatus;
import com.alibaba.dragonfly.supernode.common.enumeration.FromType;
import com.alibaba.dragonfly.supernode.common.util.BeanPoolUtil;
import com.alibaba.dragonfly.supernode.common.util.HttpClientUtil;
import com.alibaba.dragonfly.supernode.common.util.PowerRateLimiter;
import com.alibaba.dragonfly.supernode.common.util.RangeParseUtil;

import org.apache.commons.codec.binary.Hex;
import org.apache.commons.lang3.StringUtils;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class Downloader implements Callable<Boolean> {
    private static final Logger logger = LoggerFactory.getLogger(Downloader.class);

    private final Task task;
    private FutureTask<Boolean> future;

    private static final CacheDetector cacheDetector = BeanPoolUtil.getBean(CacheDetector.class);
    private static final CdnReporter cdnReporter = BeanPoolUtil.getBean(CdnReporter.class);
    private static final FileMetaDataService fileMetaDataService = BeanPoolUtil.getBean(FileMetaDataService.class);
    private static final PowerRateLimiter rateLimiter = BeanPoolUtil.getBean(PowerRateLimiter.class);

    public Downloader(Task task) {
        this.task = task;
    }

    @Override
    public Boolean call() {
        final String taskId = task.getTaskId();
        final String paramMd5 = task.getMd5();
        final String fileUrl = task.getSourceUrl();
        Long httpFileLength = task.getHttpFileLen();
        if (httpFileLength == null) {
            httpFileLength = -1L;
        }

        CacheResult cacheResult = cacheDetector.detectCache(task);
        int startPieceNum = cacheResult.getStartPieceNum();
        if (startPieceNum == -1) {
            logger.info("cache full hit for taskId:{} on local", taskId);
            return true;
        }
        int pieceContSize = task.getPieceSize() - Constants.PIECE_WRAP_SIZE;

        InputStream is = null;
        HttpURLConnection connection = null;
        BlockingQueue<ProtocolContent> qu = null;
        BlockingQueue<ByteBuffer> reusedCache = null;
        long startTime = System.currentTimeMillis();

        logger.info("taskId:{} fileUrl:{} on downloader", taskId, fileUrl);
        try {
            connection = createConn(taskId, fileUrl, task.getHeaders(), startPieceNum, httpFileLength, pieceContSize);
            if (connection != null) {
                try {
                    qu = new ArrayBlockingQueue<>(3);
                    reusedCache = new ArrayBlockingQueue<>(CdnConstants.WRITER_THREAD_LIMIT + 2);
                    startWriter(task, qu, reusedCache);

                    MessageDigest fileM5 = cacheResult.getFileM5();

                    ByteBuffer bb = ByteBuffer.allocate(pieceContSize);
                    int pieceContLeft = pieceContSize;

                    int curPieceNum = startPieceNum;
                    int bufSize = 256 * 1024;
                    byte[] buf = new byte[bufSize];
                    int count;

                    is = connection.getInputStream();

                    long readCost = 0;
                    long realFileLength = startPieceNum * pieceContSize;
                    long beforeTime = System.currentTimeMillis();
                    rateLimiter.acquire(bufSize, true);
                    while ((count = is.read(buf)) != -1) {
                        readCost += System.currentTimeMillis() - beforeTime;
                        if (count > 0) {
                            fileM5.update(buf, 0, count);
                            realFileLength += count;

                            if (pieceContLeft <= count) {
                                bb.put(buf, 0, pieceContLeft);
                                bb.flip();
                                qu.put(ProtocolContent.buildPieceContent(bb, curPieceNum++));

                                bb = reusedCache.poll();
                                if (bb == null) {
                                    bb = ByteBuffer.allocate(pieceContSize);
                                }
                                count -= pieceContLeft;
                                if (count > 0) {
                                    bb.put(buf, pieceContLeft, count);
                                }
                                pieceContLeft = pieceContSize;
                            } else {
                                bb.put(buf, 0, count);
                            }
                            pieceContLeft -= count;
                        } else {
                            logger.warn("taskId:{} read count is zero when do is.read(buf)", taskId);
                        }
                        beforeTime = System.currentTimeMillis();
                        rateLimiter.acquire(bufSize, true);
                    }

                    String realMd5 = Hex.encodeHexString(fileM5.digest());
                    boolean isSuccess = true;
                    if (StringUtils.isNotBlank(paramMd5) && !StringUtils.equalsIgnoreCase(paramMd5, realMd5)) {
                        logger.error("taskId:{} url:{} file md5 not match expected:{} real:{}", taskId, fileUrl,
                            paramMd5, realMd5);
                        isSuccess = false;
                    }
                    if (isSuccess && httpFileLength >= 0 && httpFileLength != realFileLength) {
                        logger.error("taskId:{} url:{} file length not match expected:{} real:{}", taskId, fileUrl,
                            httpFileLength, realFileLength);
                        isSuccess = false;
                    }

                    if (isSuccess) {
                        if (pieceContLeft < pieceContSize) {
                            bb.flip();
                            qu.put(ProtocolContent.buildPieceContent(bb, curPieceNum));
                        }
                    }
                    qu.put(ProtocolContent.buildTaskResult(true, isSuccess,
                        new DownloadMetaData(realMd5, realFileLength, startTime, readCost)));
                    return isSuccess;
                } catch (Exception e) {
                    logger.error("downloader read error for taskId:{},fileUrl:{}", taskId, fileUrl, e);
                }
            }
        } finally {
            try {
                if (is != null) {
                    is.close();
                }
            } catch (Exception e) {
                logger.error("E_downloader_close", e);
            }
            try {
                if (connection != null) {
                    connection.disconnect();
                }
            } catch (Exception e) {
                logger.error("E_downloader_disconnect", e);
            }
        }

        cdnReporter.reportTaskStatus(taskId, CdnStatus.FAIL, null, null, FromType.DOWNLOADER.type());
        return false;
    }

    private SuperWriter startWriter(Task task, BlockingQueue<ProtocolContent> qu, BlockingQueue<ByteBuffer> reusedCache)
        throws IOException {

        SuperWriter superWriter = new SuperWriter(task, future, qu, reusedCache);
        new Thread(superWriter).start();
        return superWriter;
    }

    public void setFuture(FutureTask<Boolean> future) {
        this.future = future;
    }

    private HttpURLConnection createConn(String taskId, String fileUrl, String[] header, int startPieceNum,
        Long httpFileLength,
        int pieceContSize) {
        HttpURLConnection conn = null;
        try {
            boolean isConnected = false;
            URL url = new URL(fileUrl);
            int checkCode = HttpURLConnection.HTTP_OK;
            int code;
            for (int i = 0; i < 3; i++) {
                try {
                    conn = HttpClientUtil.openConnection(url);
                    HttpClientUtil.fillHeaders(conn, header);
                    conn.setConnectTimeout(2000);
                    conn.setUseCaches(false);
                    conn.setRequestMethod("GET");
                    if (startPieceNum > 0) {
                        String breakRange =
                            RangeParseUtil.calculateBreakRange(startPieceNum, httpFileLength, pieceContSize);
                        logger.info("taskId:{} download start range:{}", taskId, breakRange);
                        conn.setRequestProperty("Range", "bytes=" + breakRange);
                        checkCode = HttpURLConnection.HTTP_PARTIAL;
                    }
                    conn.connect();
                    code = conn.getResponseCode();
                    if (HttpClientUtil.REDIRECTED_CODE.contains(code)) {
                        fileUrl = conn.getHeaderField("Location");
                        if (StringUtils.isNotBlank(fileUrl)) {
                            logger.info("taskId:{} redirected to url:{}", taskId, fileUrl);
                            return createConn(taskId, fileUrl, null, startPieceNum, httpFileLength, pieceContSize);
                        }
                    }
                    if (code == checkCode) {
                        fileMetaDataService.updateLastModified(taskId, conn.getLastModified());
                        isConnected = true;
                        break;
                    }
                    logger.error("taskId:{} url:{} connect code:{} at times:{}", taskId, fileUrl,
                        code, i + 1);
                } catch (Exception e) {
                    logger.warn("connect error for taskId:{},url:{}", taskId, fileUrl, e);
                } finally {
                    if (!isConnected && conn != null) {
                        try {
                            conn.disconnect();
                        } catch (Exception e) {
                            logger.error("disconnect error", e);
                        } finally {
                            conn = null;
                        }

                    }
                }

            }

        } catch (Exception e) {
            logger.error("downloader connect error for taskId:{}", taskId, e);
        }
        return conn;
    }

}
