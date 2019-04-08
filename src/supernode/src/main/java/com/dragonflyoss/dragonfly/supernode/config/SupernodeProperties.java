/*
 * Copyright The Dragonfly Authors.
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

package com.dragonflyoss.dragonfly.supernode.config;

import javax.annotation.PostConstruct;
import java.net.Inet4Address;
import java.net.InetAddress;
import java.net.NetworkInterface;
import java.nio.file.Files;
import java.nio.file.Paths;
import java.util.ArrayList;
import java.util.Enumeration;
import java.util.List;

import com.alibaba.fastjson.JSON;

import com.dragonflyoss.dragonfly.supernode.common.Constants;
import lombok.Data;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.springframework.boot.context.properties.ConfigurationProperties;

/**
 * Created on 2018/05/23
 *
 * @author lowzj
 */
@ConfigurationProperties("supernode")
@Slf4j
@Data
public class SupernodeProperties {
    /**
     * working directory of supernode, default: /home/admin/supernode
     */
    private String baseHome = Constants.DEFAULT_BASE_HOME;

    /**
     * the network rate reserved for system , default: 20 MB/s
     */
    private int systemNeedRate = Constants.DEFAULT_SYSTEM_NEED_RATE;

    /**
     * the network rate that supernode can use, default: 200 MB/s
     */
    private int totalLimit = Constants.DEFAULT_TOTAL_LIMIT;

    /**
     * the core pool size of ScheduledExecutorService, default: 10
     */
    private int schedulerCorePoolSize = Constants.DEFAULT_SCHEDULER_CORE_POOL_SIZE;

    /**
     * members of the Supernode cluster
     */
    private List<ClusterMember> cluster = new ArrayList<>();

    /**
     * the path of dfget installed locally
     */
    private String dfgetPath = Constants.DFGET_PATH;

    /**
     * The advertise ip is used to set the ip that we advertise to other peer in the p2p-network.
     * By default, the first non-loop address is advertised.
     */
    private String advertiseIp;

    /**
     * 	disables the cdn feature of supernode when the value is true.
     * 	Supernode just constructs the p2p-network and schedules the data transmission
     * 	among the peers, it doesn't download files from source file server even the
     * 	files are not cached by supernode.
     * 	And dfget will download files from source file server if they're not available
     * 	on other peer nodes.
     * 	The default value is false.
     */
    private boolean disableCDN = Constants.DEFAULT_DISABLE_CDN;

    /**
     * the number of clients which can download one piece from remote server when
     * {@link SupernodeProperties#disableCDN} is true.
     * The default value is 2(backup for each other).
     */
    private int downloadClientNumberPerPiece = Constants.DEFAULT_DOWNLOAD_CLIENT_NUMBER_PER_PIECE;

    @PostConstruct
    public void init() {
        String cdnHome = baseHome + "/repo";
        Constants.DOWNLOAD_HOME = cdnHome + Constants.DOWN_SUB_PATH;
        Constants.UPLOAD_HOME = cdnHome + Constants.HTTP_SUB_PATH;
        Constants.PREHEAT_HOME = cdnHome + Constants.PREHEAT_SUB_PATH;

        try {
            Files.createDirectories(Paths.get(Constants.DOWNLOAD_HOME));
            Files.createDirectories(Paths.get(Constants.UPLOAD_HOME));
        } catch (Exception e) {
            log.error("create repo dir error", e);
            System.exit(1);
        }

        setLocalIp();

        log.info("cluster members: {}", JSON.toJSONString(cluster));
    }

    private void setLocalIp() {
        if (StringUtils.isNotBlank(advertiseIp)) {
            Constants.localIp = advertiseIp;
            Constants.generateNodeCid();
            log.info("init local ip of supernode, use ip:{}", Constants.localIp);
        } else {
            List<String> ips = getAllIps();
            if (!ips.isEmpty()) {
                Constants.localIp = ips.get(0);
                Constants.generateNodeCid();
            }
            log.info("init local ip of supernode, ip list:{}, use ip:{}",
                JSON.toJSONString(ips), Constants.localIp);
        }
    }

    private List<String> getAllIps() {
        List<String> ips = new ArrayList<String>();
        try {
            Enumeration<NetworkInterface> allNetInterfaces = NetworkInterface
                .getNetworkInterfaces();
            while (allNetInterfaces.hasMoreElements()) {
                NetworkInterface netInterface = allNetInterfaces.nextElement();
                Enumeration<InetAddress> addresses = netInterface
                    .getInetAddresses();
                while (addresses.hasMoreElements()) {
                    InetAddress ip = addresses.nextElement();
                    if (ip instanceof Inet4Address && !ip.isAnyLocalAddress()
                        && !ip.isLoopbackAddress()) {
                        ips.add(ip.getHostAddress());
                    }
                }
            }
        } catch (Exception e) {
            log.error("getAllIps error:{}", e.getMessage(), e);
        }
        return ips;
    }
}
