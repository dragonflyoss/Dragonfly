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
package com.alibaba.dragonfly.supernode.service.timer;

import java.io.File;
import java.io.IOException;
import java.nio.file.FileVisitResult;
import java.nio.file.FileVisitor;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.nio.file.attribute.BasicFileAttributes;
import java.util.ArrayList;
import java.util.Comparator;
import java.util.List;
import java.util.TreeMap;
import java.util.concurrent.atomic.AtomicInteger;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

import com.alibaba.dragonfly.supernode.common.domain.FileMetaData;
import com.alibaba.dragonfly.supernode.common.exception.DataNotFoundException;
import com.alibaba.dragonfly.supernode.service.TaskService;
import com.alibaba.dragonfly.supernode.service.cdn.FileMetaDataService;
import com.alibaba.dragonfly.supernode.service.cdn.util.PathUtil;
import com.alibaba.dragonfly.supernode.service.lock.LockConstants;
import com.alibaba.dragonfly.supernode.service.lock.LockService;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.config.ConfigurableBeanFactory;
import org.springframework.context.annotation.Scope;
import org.springframework.stereotype.Service;

@Service
@Scope(value = ConfigurableBeanFactory.SCOPE_PROTOTYPE)
public class DownSpaceCleaner {

    private static final Logger logger = LoggerFactory.getLogger("SpaceGcLogger");

    protected static final String SPACE_TYPE_DISK = "disk";
    private static final Pattern taskIdPattern = Pattern.compile("^(.+?)(\\.md5|\\.meta)?$");

    @Autowired
    private FileMetaDataService fileMetaDataService;
    @Autowired
    private LockService lockService;
    @Autowired
    private TaskService taskService;

    private String type;
    private String path;
    private long fullGcThreshold;
    private long youngGcThreshold;
    private int cleanRatio;
    private long intervalThreshold;

    private final List<String> gcTasks = new ArrayList<>();

    private TreeMap<Long, List<String>> intervalInert = new TreeMap<>(new Comparator<Long>() {

        @Override
        public int compare(Long o1, Long o2) {
            if (o2 - o1 > 0) {
                return 1;
            } else if (o2 - o1 < 0) {
                return -1;
            }
            return 0;
        }

    });
    private TreeMap<Long, List<String>> gapInert = new TreeMap<>(new Comparator<Long>() {

        @Override
        public int compare(Long o1, Long o2) {
            if (o2 - o1 > 0) {
                return 1;
            } else if (o2 - o1 < 0) {
                return -1;
            }
            return 0;
        }

    });

    private class Cleaner implements FileVisitor<Path> {
        private AtomicInteger count;
        private boolean fullGc;
        private boolean alreadyParse;

        private Cleaner(AtomicInteger count, boolean fullGc, boolean alreadyParse) {
            this.count = count;
            this.fullGc = fullGc;
            this.alreadyParse = alreadyParse;
        }

        private void delTask(String taskId) {
            PathUtil.deleteTaskFiles(taskId, true);
            count.incrementAndGet();
        }

        private void parseInert(FileMetaData fileMetaData) {
            long gap = System.currentTimeMillis() - fileMetaData.getAccessTime();
            long interval = fileMetaData.getInterval();
            if (interval > 0) {
                try {
                    if (gap <= interval + intervalThreshold) {
                        long len = PathUtil.getDownloadPath(fileMetaData.getTaskId()).toFile().length();
                        List<String> intervalTasks = intervalInert.get(len);
                        if (intervalTasks == null) {
                            intervalTasks = new ArrayList<>();
                            intervalInert.put(len, intervalTasks);
                        }
                        intervalTasks.add(fileMetaData.getTaskId());
                        return;
                    }
                } catch (Exception e) {
                    logger.error("E_parseInert", e);
                }
            }
            List<String> gapTasks = gapInert.get(gap);
            if (gapTasks == null) {
                gapTasks = new ArrayList<>();
                gapInert.put(gap, gapTasks);
            }
            gapTasks.add(fileMetaData.getTaskId());
        }

        @Override
        public FileVisitResult preVisitDirectory(Path dir, BasicFileAttributes attrs) throws IOException {
            return FileVisitResult.CONTINUE;
        }

        private String extractTaskId(String filePathStr) {
            String taskId;
            Matcher matcher = taskIdPattern.matcher(filePathStr);
            if (matcher.matches()) {
                taskId = matcher.group(1);
            } else {
                taskId = filePathStr;
            }
            return taskId;
        }

        @Override
        public FileVisitResult visitFile(Path file, BasicFileAttributes attrs) throws IOException {
            if (Files.isRegularFile(file)) {
                String taskId = extractTaskId(file.getFileName().toString());
                String lockName = lockService.getLockName(LockConstants.FILE_META_DATA_LOCK, taskId);
                lockService.lock(lockName);
                try {
                    boolean canDel = false;
                    try {
                        taskService.get(taskId);
                    } catch (DataNotFoundException e) {
                        canDel = true;
                    }
                    if (canDel) {
                        if (!alreadyParse) {
                            FileMetaData metaData = fileMetaDataService.readFileMetaData(taskId);
                            if (metaData == null && Files.deleteIfExists(file)) {
                                logger.warn("meta not found and delete file:{}", file.toString());
                            } else {
                                parseInert(metaData);
                            }
                        } else if (fullGc || gcTasks.contains(taskId)) {
                            delTask(taskId);
                        }
                    }
                } catch (Exception e) {
                    logger.error("E_visitFile for file:{}", file.toString(), e);
                } finally {
                    lockService.unlock(lockName);
                }
            } else if (Files.exists(file)) {
                logger.error("path:{} is not regular file", file.toString());
            }
            return FileVisitResult.CONTINUE;
        }

        @Override
        public FileVisitResult visitFileFailed(Path file, IOException exc) throws IOException {
            return FileVisitResult.CONTINUE;
        }

        @Override
        public FileVisitResult postVisitDirectory(Path dir, IOException exc) throws IOException {
            return FileVisitResult.CONTINUE;
        }

    }

    public void fillConf(String type, String path, long fullGcThreshold, long youngGcThreshold, int cleanRatio,
        long intervalThreshold) {
        this.type = type;
        this.path = path;
        this.fullGcThreshold = fullGcThreshold;
        this.youngGcThreshold = youngGcThreshold;
        this.cleanRatio = cleanRatio;
        this.intervalThreshold = intervalThreshold;
    }

    public void gc(boolean force) {
        try {
            boolean fullGc = force;
            long usableSpace = getUsableSpace();
            if (!fullGc) {
                if (usableSpace <= fullGcThreshold) {
                    fullGc = true;
                } else if (usableSpace >= youngGcThreshold) {
                    return;
                }
            }

            AtomicInteger count = new AtomicInteger(0);
            boolean alreadyParse = fullGc ? true : generateGcTasks();
            if (alreadyParse) {
                logger.info("do space:{} gc with usable:{}GB", type, usableSpace / 1024 / 1024 / 1024);
            }
            Files.walkFileTree(Paths.get(this.path), this.new Cleaner(count, fullGc, alreadyParse));
            if (alreadyParse) {
                gcTasks.clear();
                logger.info("-----delete task count:{},usableSpace:{}GB for {}-----", count.get(),
                    getUsableSpace() / 1024 / 1024 / 1024,
                    type);
            }

        } catch (Exception e) {
            logger.error("E_SpaceGc", e);
        }
    }

    private long getUsableSpace() {
        long usableSpace = 0;
        try {
            File file = new File(this.path);
            usableSpace = file.getUsableSpace();
        } catch (Exception e) {
            logger.error("E_check {}", type, e);
        }
        return usableSpace;
    }

    private boolean generateGcTasks() {
        for (List<String> tasks : gapInert.values()) {
            gcTasks.addAll(tasks);
        }
        for (List<String> tasks : intervalInert.values()) {
            gcTasks.addAll(tasks);
        }
        if (gcTasks.isEmpty()) {
            return false;
        }
        int gcLen = (gcTasks.size() * this.cleanRatio + 9) / 10;
        gcTasks.retainAll(gcTasks.subList(0, gcLen));
        gapInert.clear();
        intervalInert.clear();
        return true;
    }
}
