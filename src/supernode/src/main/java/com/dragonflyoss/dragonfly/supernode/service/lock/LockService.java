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
package com.dragonflyoss.dragonfly.supernode.service.lock;

import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.locks.Lock;
import java.util.concurrent.locks.ReadWriteLock;
import java.util.concurrent.locks.ReentrantLock;
import java.util.concurrent.locks.ReentrantReadWriteLock;

import org.apache.commons.lang3.StringUtils;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Service;

@Service("lockService")
public class LockService {
    private static final Logger gcLogger = LoggerFactory
        .getLogger("DataGcLogger");

    private static final ConcurrentHashMap<String, Lock> lockMap = new ConcurrentHashMap<String, Lock>();
    private static final ConcurrentHashMap<String, Long> lockAccessMap = new ConcurrentHashMap<String, Long>();

    private static final ConcurrentHashMap<String, ReadWriteLock> rwLockMap
        = new ConcurrentHashMap<String, ReadWriteLock>();
    private static final ConcurrentHashMap<String, Long> rwLockAccessMap = new ConcurrentHashMap<String, Long>();

    private static final ReadWriteLock lockOnLock = new ReentrantReadWriteLock(
        true);

    public void lock(String lockName) {
        Lock lock = getLock(lockName);
        lock.lock();
    }

    public boolean tryLock(String lockKey) {
        Lock lock = getLock(lockKey);
        return lock.tryLock();
    }

    public void unlock(String lockName) {
        Lock lock = getLock(lockName);
        lock.unlock();
    }

    public String getLockName(String prefix, String name) {
        return prefix + name;
    }

    public void lockTaskOnRead(String taskId) {
        String lockName = getLockName(LockConstants.TASK_EXPIRE_LOCK, taskId);
        ReadWriteLock lock = getRWLock(lockName);
        lock.readLock().lock();
    }

    public void lockTaskOnWrite(String taskId) {
        String lockName = getLockName(LockConstants.TASK_EXPIRE_LOCK, taskId);
        ReadWriteLock lock = getRWLock(lockName);
        lock.writeLock().lock();
    }

    public void unlockTaskOnRead(String taskId) {
        String lockName = getLockName(LockConstants.TASK_EXPIRE_LOCK, taskId);
        ReadWriteLock lock = getRWLock(lockName);
        lock.readLock().unlock();
    }

    public void unlockTaskOnWrite(String taskId) {
        String lockName = getLockName(LockConstants.TASK_EXPIRE_LOCK, taskId);
        ReadWriteLock lock = getRWLock(lockName);
        lock.writeLock().unlock();
    }

    public void gc(long expire) throws InterruptedException {
        gcLogger.info("****** gc lock ******");
        gcNormalLock(expire);
        gcRWLock(expire);
    }

    public void gcCdnLock(String taskId) {
        if (StringUtils.isBlank(taskId)) {
            return;
        }
        String name = getLockName(LockConstants.CDN_TRIGGER_LOCK, taskId);
        lockOnLock.writeLock().lock();
        lockMap.remove(name);
        lockAccessMap.remove(name);
        lockOnLock.writeLock().unlock();
    }

    private void gcNormalLock(long expire) throws InterruptedException {
        List<String> locks = new ArrayList<String>(lockMap.keySet());
        int count = 0;
        for (String lockName : locks) {
            lockOnLock.writeLock().lock();
            try {
                Long accessTime = lockAccessMap.get(lockName);
                if (accessTime == null
                    || System.currentTimeMillis() - accessTime > expire) {
                    lockAccessMap.remove(lockName);
                    lockMap.remove(lockName);
                    count++;
                }
            } finally {
                lockOnLock.writeLock().unlock();
                Thread.sleep(5);
            }
        }
        gcLogger.info("gcNormalLock count:{}", count);
    }

    private void gcRWLock(long expire) throws InterruptedException {
        List<String> locks = new ArrayList<String>(rwLockMap.keySet());
        int count = 0;
        for (String lockName : locks) {
            lockOnLock.writeLock().lock();
            try {
                Long accessTime = rwLockAccessMap.get(lockName);
                if (accessTime == null
                    || System.currentTimeMillis() - accessTime > expire) {
                    rwLockAccessMap.remove(lockName);
                    rwLockMap.remove(lockName);
                    count++;
                }
            } finally {
                lockOnLock.writeLock().unlock();
                Thread.sleep(5);
            }
        }
        gcLogger.info("gcRWLock count:{}", count);
    }

    private Lock getLock(String lockName) {
        lockOnLock.readLock().lock();
        try {
            lockAccessMap.put(lockName, System.currentTimeMillis());
            Lock lock = lockMap.get(lockName);
            if (lock == null) {
                lock = new ReentrantLock();
                Lock existLock = lockMap.putIfAbsent(lockName, lock);
                if (existLock != null) {
                    return existLock;
                }
            }
            return lock;
        } finally {
            lockOnLock.readLock().unlock();
        }
    }

    private ReadWriteLock getRWLock(String lockName) {
        lockOnLock.readLock().lock();
        try {
            rwLockAccessMap.put(lockName, System.currentTimeMillis());
            ReadWriteLock lock = rwLockMap.get(lockName);
            if (lock == null) {
                lock = new ReentrantReadWriteLock();
                ReadWriteLock existLock = rwLockMap.putIfAbsent(lockName, lock);
                if (existLock != null) {
                    return existLock;
                }
            }
            return lock;
        } finally {
            lockOnLock.readLock().unlock();
        }
    }

    public boolean isAccessWindow(String lockName, long windowTime) {
        Long accessTime = lockAccessMap.get(lockName);
        return accessTime == null
            || System.currentTimeMillis() - accessTime >= windowTime;
    }

}
