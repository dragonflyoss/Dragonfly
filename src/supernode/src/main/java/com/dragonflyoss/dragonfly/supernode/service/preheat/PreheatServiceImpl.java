package com.dragonflyoss.dragonfly.supernode.service.preheat;

import javax.annotation.PostConstruct;
import java.util.HashMap;
import java.util.LinkedList;
import java.util.List;
import java.util.Map;
import java.util.UUID;
import java.util.concurrent.ExecutorService;

import com.dragonflyoss.dragonfly.supernode.common.Constants;
import com.dragonflyoss.dragonfly.supernode.common.domain.FileMetaData;
import com.dragonflyoss.dragonfly.supernode.common.domain.PreheatTask;
import com.dragonflyoss.dragonfly.supernode.common.domain.Task;
import com.dragonflyoss.dragonfly.supernode.common.enumeration.PreheatTaskStatus;
import com.dragonflyoss.dragonfly.supernode.common.exception.DataNotFoundException;
import com.dragonflyoss.dragonfly.supernode.common.exception.PreheatException;
import com.dragonflyoss.dragonfly.supernode.common.util.NetConfigNotification;
import com.dragonflyoss.dragonfly.supernode.common.util.UrlUtil;
import com.dragonflyoss.dragonfly.supernode.repository.PreheatTaskRepository;
import com.dragonflyoss.dragonfly.supernode.service.TaskService;
import com.dragonflyoss.dragonfly.supernode.service.cdn.FileMetaDataService;
import com.dragonflyoss.dragonfly.supernode.service.lock.LockService;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
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
    private FileMetaDataService fileMetaDataService;

    @Autowired
    private LockService lockService;

    @Autowired
    private NetConfigNotification netConfigNotification;

    @Autowired
    private PreheatTaskRepository repository;

    @Autowired
    private List<Preheater> preheaterBeans;

    private Map<String, Preheater> preheaterMap = new HashMap<>();

    private static ExecutorService executorService = ExecutorUtils.newThreadPool("preheat", 20, 100);

    @PostConstruct
    public void init() {
        if (preheaterBeans != null) {
            for (Preheater preheater : preheaterBeans) {
                preheaterMap.put(preheater.type().toLowerCase(), preheater);
            }
        }
    }

    //-------------------------------------------------------------------------
    // implement interface

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
    public boolean update(String id, PreheatTask task) {
        return repository.update(id, task);
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
        if (previous != null && previous.getFinishTime() > 0) {
            throw new PreheatException(409, "preheat task already exists, id:" + task.getId(), task.getId());
        }
        executorService.execute(preheater.newWorker(task, this));
        return id;
    }

    @Override
    public Process executePreheat(PreheatTask task) throws PreheatException {
        if (!needPreheat(task.getId())) {
            return null;
        }

        String tmpName = UUID.randomUUID().toString();
        String tmpTarget = Constants.PREHEAT_HOME + tmpName;

        String[] cmd = createCommand(task.getUrl(), task.getHeaders(), task.getFilter(),
            task.getIdentifier(), tmpTarget);
        log.info("command: {}", StringUtils.join(cmd, " "));
        try {
            return Runtime.getRuntime().exec(cmd);
        } catch (Exception e) {
            throw new PreheatException(500, e.getMessage());
        }
    }

    //-------------------------------------------------------------------------

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

    private boolean needPreheat(String taskId) {
        Task existTask = null;
        try {
            existTask = downloadTaskService.get(taskId);
        } catch (DataNotFoundException e) {
            // pass
        }
        FileMetaData fileMetaData = fileMetaDataService.readFileMetaData(taskId);

        if (existTask != null || (fileMetaData != null && fileMetaData.isSuccess())) {
            log.info("task or file exist for taskId:{}, need not to preheat", taskId);
            return false;
        }
        return true;
    }

    private String[] createCommand(String url, Map<String, String> headers, String filter, String identifier, String tmpTarget) {
        String heatCommand;
        int netRate = netConfigNotification.getNetRate();
        String rate = netRate / 2 + "M";

        Command cmd = new Command();
        cmd.add(Constants.DFGET_PATH)
            .add("-u").add(url)
            .add("-o").add(tmpTarget)
            .add("--callsystem").add("dragonfly_preheat")
            .add("--totallimit").add(rate)
            .add("-s").add(rate)
            .add("--notbs");

        if (headers != null && !headers.isEmpty()) {
            for (Map.Entry<String, String> header : headers.entrySet()) {
                cmd.add("--header")
                    .add(header.getKey() + ":" + header.getValue());
            }
        }
        if (StringUtils.isNotBlank(Constants.localIp)) {
            cmd.add("-n").add(Constants.localIp);
        }
        if (StringUtils.isNotBlank(filter)) {
            cmd.add("-f").add(filter);
        }
        if (StringUtils.isNotBlank(identifier)) {
            cmd.add("-i").add(identifier);
        }
        return cmd.toArray();
    }

    private List<String> add(List<String> cmd, String arg) {
        cmd.add(arg);
        return cmd;
    }

    class Command {
        private List<String> args = new LinkedList<>();

        public Command add(String arg) {
            args.add(arg);
            return this;
        }

        String[] toArray() {
            return args.toArray(new String[0]);
        }
    }
}
