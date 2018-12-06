package com.dragonflyoss.dragonfly.supernode.config;

import lombok.Data;

/**
 * Created on 2018/11/06
 *
 * @author lowzj
 */
@Data
public class ClusterMember {
    private String ip;
    private int registerPort = 8001;
    private int downloadPort = 8002;
}
