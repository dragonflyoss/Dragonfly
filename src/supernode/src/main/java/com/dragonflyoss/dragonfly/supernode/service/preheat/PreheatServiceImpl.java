package com.dragonflyoss.dragonfly.supernode.service.preheat;

import javax.annotation.PostConstruct;
import java.util.Arrays;
import java.util.HashMap;
import java.util.LinkedList;
import java.util.List;
import java.util.Map;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.LinkedBlockingQueue;
import java.util.concurrent.ThreadFactory;
import java.util.concurrent.ThreadPoolExecutor;
import java.util.concurrent.TimeUnit;

import com.dragonflyoss.dragonfly.supernode.common.domain.PreheatTask;
import com.dragonflyoss.dragonfly.supernode.common.exception.PreheatException;
import com.dragonflyoss.dragonfly.supernode.repository.PreheatTaskRepository;
import com.dragonflyoss.dragonfly.supernode.service.TaskService;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.apache.commons.lang3.concurrent.BasicThreadFactory;
import org.apache.logging.log4j.util.Strings;
import org.springframework.beans.factory.annotation.Autowired;
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
    public String preheat(PreheatTask task) throws PreheatException {
        Preheater preheater = preheaterMap.get(task.getType().toLowerCase());
        if (preheater == null) {
            throw new PreheatException(400, task.getType() + " isn't supported");
        }
        String id = createTaskId(task.getUrl(), task.getFilter(), task.getIdentifier());
        task.setId(id);
        executorService.execute(new PreheatWorker(task, preheater, this));
        return id;
    }

    //-------------------------------------------------------------------------
    // private methods

    private String createTaskId(String url, String filter, String identifier) {
        String taskUrl = taskUrl(url, filter);
        return downloadTaskService.createTaskId(taskUrl, null, identifier);
    }

    private String taskUrl(String url, String filter) {
        final String sep = "&";

        if (StringUtils.isBlank(filter)) {
            return url;
        }
        String[] rawUrls = url.split("\\?", 2);
        if (rawUrls.length < 2 || StringUtils.isBlank(rawUrls[1])) {
            return url;
        }

        List<String> filters = Arrays.asList(filter.split(sep));
        List<String> params = new LinkedList<>();
        for (String param: rawUrls[1].split(sep)) {
            String[] kv = param.split("=");
            if (!(kv.length >= 1 && filters.contains(kv[0]))) {
                params.add(param);
            }
        }

        return rawUrls[0] + "?" + Strings.join(params, sep.charAt(0));
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
