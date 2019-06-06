package com.dragonflyoss.dragonfly.supernode.common.domain;

import com.dragonflyoss.dragonfly.supernode.common.enumeration.ClientErrorType;

import lombok.Data;

/**
 * @author lowzj
 */
@Data
public class ClientErrorInfo {
    private ClientErrorType errorType;
    private String srcCid;
    private String dstCid;
    private String dstIp;
    private String taskId;
    private String range;
    private String realMd5;
    private String expectedMd5;
}
