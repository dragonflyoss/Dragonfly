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
package com.alibaba.dragonfly.supernode.service.impl;

import java.util.List;

import com.alibaba.dragonfly.supernode.common.domain.PeerInfo;
import com.alibaba.dragonfly.supernode.common.domain.gc.GcMeta;
import com.alibaba.dragonfly.supernode.common.exception.DataNotFoundException;
import com.alibaba.dragonfly.supernode.repository.PeerRepository;
import com.alibaba.dragonfly.supernode.service.PeerService;

import org.apache.commons.collections.CollectionUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;

@Service("peerService")
public class PeerServiceImpl implements PeerService {
    @Autowired
    private PeerRepository peerRepo;

    @Override
    public void add(PeerInfo peerInfo) {
        peerRepo.add(peerInfo);
    }

    @Override
    public PeerInfo get(String cid) {
        PeerInfo peerInfo = peerRepo.get(cid);
        if (peerInfo == null) {
            throw new DataNotFoundException("PeerInfo", cid,
                "peerInfo not found");
        }
        return peerInfo;
    }

    @Override
    public boolean gc(GcMeta gcMeta) {
        boolean result = false;
        if (gcMeta != null) {
            List<String> cids = gcMeta.getCids();
            if (CollectionUtils.isNotEmpty(cids)) {
                for (String cid : cids) {
                    peerRepo.remove(cid);
                }
            }
            result = true;
        }
        return result;
    }
}
