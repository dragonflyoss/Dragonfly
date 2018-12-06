package com.dragonflyoss.dragonfly.supernode.repository;

import java.util.LinkedList;
import java.util.List;
import java.util.concurrent.ConcurrentHashMap;

import com.dragonflyoss.dragonfly.supernode.common.domain.PreheatTask;
import org.springframework.stereotype.Repository;

/**
 * @author lowzj
 */
@Repository
public class PreheatTaskRepository {
    private static ConcurrentHashMap<String, PreheatTask> preheatTasks =
        new ConcurrentHashMap<String, PreheatTask>();

    public PreheatTask get(String id) {
        return preheatTasks.get(id);
    }

    public List<PreheatTask> getAll() {
        return new LinkedList<>(preheatTasks.values());
    }

    public PreheatTask add(PreheatTask task) {
        return preheatTasks.putIfAbsent(task.getId(), task);
    }

    public boolean update(PreheatTask task) {
        PreheatTask preTask = preheatTasks.get(task.getId());
        if (preTask != null) {
            if (task.getStatus() != null) {
                preTask.setStatus(task.getStatus());
            }
            if (task.getStartTime() > 0) {
                preTask.setStartTime(task.getStartTime());
            }
            if (task.getFinishTime() > 0) {
                preTask.setFinishTime(task.getFinishTime());
            }
            return true;
        }
        return false;
    }

    public boolean delete(String id) {
        PreheatTask task = preheatTasks.remove(id);
        return task != null;
    }
}
