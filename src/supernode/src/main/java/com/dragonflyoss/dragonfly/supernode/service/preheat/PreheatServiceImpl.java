package com.dragonflyoss.dragonfly.supernode.service.preheat;

import javax.annotation.PostConstruct;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.LinkedBlockingQueue;
import java.util.concurrent.ThreadFactory;
import java.util.concurrent.ThreadPoolExecutor;
import java.util.concurrent.TimeUnit;

import com.dragonflyoss.dragonfly.supernode.common.domain.PreheatTask;
import com.dragonflyoss.dragonfly.supernode.common.enumeration.PreheatTaskStatus;
import com.dragonflyoss.dragonfly.supernode.common.exception.PreheatException;
import com.dragonflyoss.dragonfly.supernode.common.util.UrlUtil;
import com.dragonflyoss.dragonfly.supernode.repository.PreheatTaskRepository;
import com.dragonflyoss.dragonfly.supernode.service.TaskService;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.apache.commons.lang3.concurrent.BasicThreadFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Service;

/**
 * @author lowzj
 */
@Service("preheatService")
@Slf4j
public class PreheatServiceImpl implements PreheatService {

    @Autowired
    private TaskService downloadTaskService;

    @Autowired
    private PreheatTaskRepository repository;

    @Autowired
    private List<Preheater> preheaterBeans;

    private Map<String, Preheater> preheaterMap = new HashMap<>();

    private static ExecutorService executorService = newThreadPool("preheat", 20, 100);

    @PostConstruct
    public void init() {
        if (preheaterBeans != null) {
            for (Preheater preheater : preheaterBeans) {
                preheaterMap.put(preheater.type().toLowerCase(), preheater);
            }
        }
    }

    @Override
    public PreheatTask get(String id) {
        if (StringUtils.isNotBlank(id)) {
            return repository.get(id);
        }
        return null;
    }

    @Override
    public List<PreheatTask> getAll() {
        return repository.getAll();
    }

    @Override
    public boolean delete(String id) {
        PreheatTask task = repository.get(id);
        if (task != null && task.getChildren() != null) {
            for (String child : task.getChildren()) {
                repository.delete(child);
            }
        }
        return repository.delete(id);
    }

    @Override
    public String createPreheatTask(PreheatTask task) throws PreheatException {
        Preheater preheater = preheaterMap.get(task.getType().toLowerCase());
        if (preheater == null) {
            throw new PreheatException(400, task.getType() + " isn't supported");
        }
        String id = createTaskId(task.getUrl(), task.getFilter(), task.getIdentifier());
        task.setId(id);
        task.setStartTime(System.currentTimeMillis());
        task.setStatus(PreheatTaskStatus.WAITING);
        PreheatTask previous;
        try {
            previous = repository.add(task);
        } catch (Exception e) {
            throw new PreheatException(500, e.getMessage());
        }
        if (previous != null) {
            throw new PreheatException(400, "preheat task already exists, id:" + task.getId());
        }
        executorService.execute(preheater.newWorker(task, this));
        return id;
    }

    @Scheduled(initialDelay = 6000, fixedDelay = 1800000)
    public void deleteExpiresPreheatTask() {
        List<String> ids = repository.getAllIds();
        int count = 0;
        for (String id : ids) {
            if (repository.isExpired(id)) {
                repository.delete(id);
                count++;
            }
        }
        log.info("deleteExpiresPreheatTask, count:{}", count);
    }

    //-------------------------------------------------------------------------
    // private methods

    private String createTaskId(String url, String filter, String identifier) {
        String taskUrl = UrlUtil.filterParam(url, filter);
        return downloadTaskService.createTaskId(taskUrl, null, identifier);
    }

    private static ExecutorService newThreadPool(String name, int corePoolSize, int maxPoolSize) {
        return new ThreadPoolExecutor(corePoolSize, maxPoolSize,
            60L, TimeUnit.SECONDS,
            new LinkedBlockingQueue<Runnable>(), newThreadFactory(name));
    }

    private static ThreadFactory newThreadFactory(String name) {
        return new BasicThreadFactory.Builder()
            .namingPattern(name + "-%d")
            .daemon(true)
            .build();
    }
}
