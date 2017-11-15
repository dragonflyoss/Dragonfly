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
import java.io.RandomAccessFile;
import java.nio.ByteBuffer;
import java.nio.ByteOrder;
import java.nio.channels.FileChannel;
import java.security.MessageDigest;
import java.util.List;
import java.util.concurrent.BlockingQueue;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.FutureTask;
import java.util.concurrent.SynchronousQueue;
import java.util.concurrent.ThreadPoolExecutor;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicInteger;

import com.alibaba.dragonfly.supernode.common.CdnConstants;
import com.alibaba.dragonfly.supernode.common.domain.DownloadMetaData;
import com.alibaba.dragonfly.supernode.common.domain.Task;
import com.alibaba.dragonfly.supernode.common.enumeration.CdnStatus;
import com.alibaba.dragonfly.supernode.common.enumeration.FromType;
import com.alibaba.dragonfly.supernode.common.enumeration.PeerPieceStatus;
import com.alibaba.dragonfly.supernode.common.util.BeanPoolUtil;
import com.alibaba.dragonfly.supernode.service.TaskService;
import com.alibaba.dragonfly.supernode.service.cdn.util.PathUtil;

import org.apache.commons.codec.binary.Hex;
import org.apache.commons.codec.digest.DigestUtils;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class SuperWriter implements Runnable {
    private static final Logger logger = LoggerFactory.getLogger(SuperWriter.class);

    private final Task task;
    private FutureTask<Boolean> future;
    private BlockingQueue<ProtocolContent> contQu;
    private BlockingQueue<ByteBuffer> reusedCache;
    private CountDownLatch downLatch;
    private RandomAccessFile raf;
    private FileChannel fc;

    private static final CdnReporter cdnReporter = BeanPoolUtil.getBean(CdnReporter.class);
    private static final FileMetaDataService fileMetaDataService = BeanPoolUtil.getBean(FileMetaDataService.class);
    private static final TaskService taskService = BeanPoolUtil.getBean(TaskService.class);

    private static final ExecutorService pieceExecutor =
        new ThreadPoolExecutor(20, 100, 60L, TimeUnit.SECONDS, new SynchronousQueue<Runnable>());

    public SuperWriter(Task task, FutureTask<Boolean> future, BlockingQueue<ProtocolContent> qu,
        BlockingQueue<ByteBuffer> reusedCache) throws IOException {
        this.task = task;
        this.future = future;
        this.contQu = qu;
        this.reusedCache = reusedCache;
    }

    @Override
    public void run() {
        synchronized (this) {
            String taskId = task.getTaskId();
            Long httpFileLen = task.getHttpFileLen();
            boolean needFinal = true;
            try {
                raf = new RandomAccessFile(PathUtil.getDownloadPathStr(taskId), "rw");
                fc = raf.getChannel();

                int threadSize = CdnConstants.WRITER_THREAD_LIMIT;
                Integer pieceSize = task.getPieceSize();
                if (httpFileLen != null && httpFileLen > 0) {
                    int tmpSize = (int)((httpFileLen + pieceSize - 1) / pieceSize);
                    threadSize = tmpSize >= threadSize ? threadSize : tmpSize;
                }
                downLatch = new CountDownLatch(threadSize);
                AtomicInteger sucCount = new AtomicInteger(0);
                for (int i = 0; i < threadSize; i++) {
                    pieceExecutor.submit(
                        new Writer(taskId, contQu, downLatch, reusedCache, sucCount, fc, pieceSize));
                }

                downLatch.await();

                ProtocolContent protocolContent = contQu.poll();
                if (protocolContent != null && sucCount.get() == threadSize) {
                    if (protocolContent.isTaskType() && protocolContent.isFinish()) {
                        boolean isSuc = protocolContent.isSuccess();
                        DownloadMetaData downloadMetaData = protocolContent.getDownloadMetaData();
                        Long wrapFileSize = fc.size();
                        fileMetaDataService.updateStatusAndResult(taskId, true, isSuc, downloadMetaData.getMd5(),
                            isSuc ? wrapFileSize : null);
                        if (isSuc) {
                            cdnReporter.reportTaskStatus(taskId, CdnStatus.SUCCESS, downloadMetaData.getMd5(),
                                wrapFileSize, FromType.LOCAL.type());
                            logger.info("taskId:{} readCost:{},totalCost:{},fileLength:{},realMd5:{}", taskId,
                                downloadMetaData.getReadCost(),
                                System.currentTimeMillis() - downloadMetaData.getStartTime(),
                                downloadMetaData.getFileLength(), downloadMetaData.getMd5());
                            List<String> pieceMd5s = taskService.getFullPieceMd5sByTask(task);
                            fileMetaDataService.writePieceMd5(taskId, downloadMetaData.getMd5(), pieceMd5s);
                            return;
                        }
                    }
                }
            } catch (Exception e) {
                logger.error("super writer error for taskId:{}", taskId, e);
            } finally {
                if (needFinal) {
                    cancelDownloader();
                }
                contQu.clear();
                reusedCache.clear();
                syncFile();
            }
            cdnReporter.reportTaskStatus(taskId, CdnStatus.FAIL, null, null, "writer");
        }
    }

    private void syncFile() {
        try {
            fc.force(false);
            raf.getFD().sync();
            raf.close();
        } catch (Exception e) {
            logger.error("sync file error for taskId:{}", task.getTaskId(), e);
        }
    }

    private void cancelDownloader() {
        try {
            future.cancel(true);
        } catch (Exception e) {
            logger.warn("cancel downloader error for taskId:{}", task.getTaskId(), e);
        }
    }

    private static class Writer implements Runnable {
        private String taskId;
        private BlockingQueue<ProtocolContent> contQu;
        private CountDownLatch downLatch;
        private BlockingQueue<ByteBuffer> reusedCache;
        private AtomicInteger sucCount;
        private FileChannel fc;
        private Integer pieceSize;
        private Integer pieceSizeBit;
        private Integer waitTime;

        private Writer(String taskId, BlockingQueue<ProtocolContent> contQu,
            CountDownLatch downLatch, BlockingQueue<ByteBuffer> reusedCache, AtomicInteger sucCount, FileChannel fc,
            Integer pieceSize) {
            this.taskId = taskId;
            this.contQu = contQu;
            this.downLatch = downLatch;
            this.reusedCache = reusedCache;
            this.sucCount = sucCount;
            this.fc = fc;
            this.pieceSize = pieceSize;
            this.waitTime = pieceSize / (64 * 1024);
            this.pieceSizeBit = pieceSize << 4;
        }

        @Override
        public void run() {
            try {
                MessageDigest pieceM5 = DigestUtils.getMd5Digest();
                int pieceNum;
                ByteBuffer bb, byteBuf = ByteBuffer.allocate(this.pieceSize);
                byteBuf.order(ByteOrder.BIG_ENDIAN);

                while (true) {
                    ProtocolContent protocolContent = contQu.poll(waitTime, TimeUnit.SECONDS);
                    if (protocolContent == null) {
                        logger.warn("taskId:{} get piece timeout", taskId);
                        break;
                    }
                    if (protocolContent.isPieceType()) {
                        pieceNum = protocolContent.getPieceNum();
                        bb = protocolContent.getContent();

                        byteBuf.putInt(bb.limit() | this.pieceSizeBit);
                        byteBuf.put(bb);
                        byteBuf.put((byte)0x7f);
                        bb.clear();
                        reusedCache.offer(bb);
                        reportPiece(pieceM5, byteBuf, pieceNum);
                    } else {
                        contQu.put(protocolContent);
                        break;
                    }
                }
                sucCount.incrementAndGet();
            } catch (Exception e) {
                logger.error("write piece error for taskId:{}", taskId, e);
            } finally {
                downLatch.countDown();
            }
        }

        private void reportPiece(MessageDigest pieceM5, ByteBuffer byteBuf, int curPieceNum) throws IOException {

            byteBuf.flip();
            pieceM5.update(byteBuf);
            byteBuf.flip();
            fc.write(byteBuf, (long)curPieceNum * this.pieceSize);
            cdnReporter.reportPieceStatus(taskId, curPieceNum,
                Hex.encodeHexString(pieceM5.digest()) + ":" + byteBuf.limit(), PeerPieceStatus.SUCCESS,
                FromType.LOCAL.type());
            pieceM5.reset();
            byteBuf.clear();
        }
    }

}
