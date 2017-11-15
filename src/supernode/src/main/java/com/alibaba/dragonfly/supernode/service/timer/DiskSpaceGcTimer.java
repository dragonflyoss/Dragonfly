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

import javax.annotation.PostConstruct;

import com.alibaba.dragonfly.supernode.common.Constants;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Service;

@Service
public class DiskSpaceGcTimer {

    @Autowired
    private DownSpaceCleaner downSpaceCleaner;

    @PostConstruct
    public void init() {
        long totalSpace = 0;
        try {
            File file = new File(Constants.DOWNLOAD_HOME);
            totalSpace = file.getTotalSpace();
        } catch (Exception e) {
        }
        long youngGcThreshold = totalSpace > 0 && totalSpace / 4 < 100 * 1024 * 1024 * 1024L ? totalSpace / 4
            : 100 * 1024 * 1024 * 1024L;
        downSpaceCleaner.fillConf(DownSpaceCleaner.SPACE_TYPE_DISK, Constants.DOWNLOAD_HOME,
            5 * 1024 * 1024 * 1024L,
            youngGcThreshold, 1, 2 * 3600 * 1000L);
    }

    /**
     * space gc
     */
    @Scheduled(fixedDelay = 15 * 1000L)
    public void gc() {
        downSpaceCleaner.gc(false);
    }

}
