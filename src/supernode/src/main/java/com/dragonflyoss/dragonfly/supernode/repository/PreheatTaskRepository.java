package com.dragonflyoss.dragonfly.supernode.repository;

import javax.annotation.PostConstruct;
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.StandardOpenOption;
import java.util.ArrayList;
import java.util.LinkedList;
import java.util.List;
import java.util.concurrent.ConcurrentHashMap;

import com.alibaba.fastjson.JSON;

import com.dragonflyoss.dragonfly.supernode.common.domain.PreheatTask;
import com.dragonflyoss.dragonfly.supernode.service.cdn.util.PathUtil;
import com.dragonflyoss.dragonfly.supernode.service.lock.LockConstants;
import com.dragonflyoss.dragonfly.supernode.service.lock.LockService;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Repository;

/**
 * @author lowzj
 */
@Repository
@Slf4j
public class PreheatTaskRepository {
    /**
     * preheat task expires time: 7 days
     */
    private static final long EXPIRED_TIME = 7 * 24 * 3600 * 1000;

    private static ConcurrentHashMap<String, PreheatTask> preheatTasks =
        new ConcurrentHashMap<String, PreheatTask>();

    private boolean loaded = false;

    @Autowired
    private LockService lockService;

    @PostConstruct
    public void init() {
        load();
    }

    public PreheatTask get(String id) {
        return preheatTasks.get(id);
    }

    public List<PreheatTask> getAll() {
        return new ArrayList<>(preheatTasks.values());
    }

    public List<String> getAllIds() {
        return new ArrayList<>(preheatTasks.keySet());
    }

    public PreheatTask add(PreheatTask task) throws Exception {
        if (preheatTasks.containsKey(task.getId())) {
            return preheatTasks.get(task.getId());
        }
        if (!addId(task.getId(), task.getStartTime()) || !writeTask(task)) {
            throw new Exception("store task information error");
        }
        return preheatTasks.putIfAbsent(task.getId(), task);
    }

    public boolean update(String id, PreheatTask task) {
        PreheatTask preTask = preheatTasks.get(id);
        if (preTask != null) {
            if (task.getParentId() != null) {
                preTask.setParentId(task.getParentId());
            }
            if (task.getChildren() != null) {
                preTask.setChildren(task.getChildren());
            }
            if (task.getStatus() != null) {
                preTask.setStatus(task.getStatus());
            }
            if (task.getStartTime() > 0) {
                preTask.setStartTime(task.getStartTime());
            }
            if (task.getFinishTime() > 0) {
                preTask.setFinishTime(task.getFinishTime());
            }
            return writeTask(preTask);
        }
        return false;
    }

    public boolean delete(String id) {
        if (!deleteTask(id)) {
            log.warn("delete preheat task:{} from disk failed", id);
        }
        PreheatTask task = preheatTasks.remove(id);
        return task != null;
    }

    public boolean isExpired(String id) {
        PreheatTask task = preheatTasks.get(id);
        return task != null && isExpired(task.getStartTime());
    }

    // ------------------------------------------------------------------------
    // private methods

    private boolean isExpired(long timestamp) {
        return timestamp < System.currentTimeMillis() - EXPIRED_TIME;
    }

    private void load() {
        if (loaded) {
            return;
        }
        synchronized (this) {
            if (loaded) {
                return;
            }
            List<String> ids = readIds();
            if (ids == null) {
                return;
            }
            StringBuilder sb = new StringBuilder();
            for (String id : ids) {
                PreheatTask task = readTask(id);
                if (task != null) {
                    sb.append(indexLine(task.getId(), task.getStartTime()));
                    preheatTasks.put(id, task);
                }
            }
            writeIds(sb.toString(), false);
            loaded = true;
        }
    }

    private boolean writeTask(PreheatTask task) {
        String taskId = task.getId();
        String lockName = lockService.getLockName(LockConstants.PREHEAT_TASK_LOCK, taskId);
        lockService.lock(lockName);
        try {
            Path metaPath = PathUtil.getPreheatMetaPath(taskId);
            Files.createDirectories(metaPath.getParent());
            Files.write(metaPath, JSON.toJSONBytes(task));
            return true;
        } catch (IOException e) {
            log.error("writeTask preheat meta error, task:{}", JSON.toJSONString(task), e);
        } finally {
            lockService.unlock(lockName);
        }
        return false;
    }

    private PreheatTask readTask(String id) {
        String lockName = lockService.getLockName(LockConstants.PREHEAT_TASK_LOCK, id);
        lockService.lock(lockName);
        try {
            Path metaPath = PathUtil.getPreheatMetaPath(id);
            if (Files.notExists(metaPath)) {
                return null;
            }
            String metaString = new String(Files.readAllBytes(metaPath), StandardCharsets.UTF_8);
            return JSON.parseObject(metaString, PreheatTask.class);
        } catch (Exception e) {
            log.error("readTask preheat task from file, id:{} error:{}", id, e.getMessage(), e);
        } finally {
            lockService.unlock(lockName);
        }
        return null;
    }

    private boolean deleteTask(String id) {
        String lockName = lockService.getLockName(LockConstants.PREHEAT_TASK_LOCK, id);
        lockService.lock(lockName);
        try {
            Path metaPath = PathUtil.getPreheatMetaPath(id);
            Files.deleteIfExists(metaPath);
        } catch (Exception e) {
            log.error("readTask preheat task from file, id:{} error:{}", id, e.getMessage(), e);
        } finally {
            lockService.unlock(lockName);
        }
        return false;
    }

    private String indexLine(String id, long timestamp) {
        return timestamp + " " + id + System.lineSeparator();
    }

    private boolean addId(String id, long timestamp) {
        return writeIds(indexLine(id, timestamp), true);
    }

    private boolean writeIds(String data, boolean append) {
        lockService.lock(LockConstants.PREHEAT_INDEX_LOCK);
        try {
            Path indexPath = PathUtil.getPreheatIndexPath();
            Files.createDirectories(indexPath.getParent());
            if (append) {
                Files.write(indexPath, data.getBytes(StandardCharsets.UTF_8),
                    StandardOpenOption.CREATE,
                    StandardOpenOption.APPEND);
            } else {
                Files.write(indexPath, data.getBytes(StandardCharsets.UTF_8));
            }
            return true;
        } catch (IOException e) {
            log.error("addIds, error:{}", e.getMessage(), e);
        } finally {
            lockService.unlock(LockConstants.PREHEAT_INDEX_LOCK);
        }
        return false;
    }

    private List<String> readIds() {
        // file format: timestamp id
        final int timestampLength = 13;
        lockService.lock(LockConstants.PREHEAT_INDEX_LOCK);
        try {
            Path indexPath = PathUtil.getPreheatIndexPath();
            if (Files.notExists(indexPath)) {
                return null;
            }
            List<String> ids = new LinkedList<>();
            List<String> timestampIds = Files.readAllLines(indexPath, StandardCharsets.UTF_8);
            for (String tsId : timestampIds) {
                try {
                    long timestamp = Long.parseLong(tsId.substring(0, timestampLength));
                    if (isExpired(timestamp)) {
                        String id = tsId.substring(timestampLength + 1);
                        if (StringUtils.isNotBlank(id)) {
                            ids.add(id);
                        }
                    }
                } catch (NumberFormatException e) {
                    log.warn("readIds {}, parseLong error:{}", tsId, e.getMessage());
                }
            }
            return ids;
        } catch (IOException e) {
            e.printStackTrace();
        } finally {
            lockService.unlock(LockConstants.PREHEAT_INDEX_LOCK);
        }
        return null;
    }
}
