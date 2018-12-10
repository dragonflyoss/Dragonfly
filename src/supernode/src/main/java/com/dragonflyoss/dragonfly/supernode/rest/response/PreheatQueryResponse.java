package com.dragonflyoss.dragonfly.supernode.rest.response;

import java.util.Date;

import com.dragonflyoss.dragonfly.supernode.common.domain.PreheatTask;
import com.dragonflyoss.dragonfly.supernode.common.enumeration.PreheatTaskStatus;
import com.fasterxml.jackson.annotation.JsonFormat;
import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.Data;

/**
 * @author lowzj
 */
@Data
public class PreheatQueryResponse {
    @JsonProperty("ID")
    private String id;
    @JsonFormat(pattern="yyyy-MM-dd HH:mm:ss", timezone = "GMT+8")
    private Date startTime;
    @JsonFormat(pattern="yyyy-MM-dd HH:mm:ss", timezone = "GMT+8")
    private Date finishTime;
    private PreheatTaskStatus status;

    public PreheatQueryResponse(PreheatTask task) {
        this.id = task.getId();
        if (task.getStartTime() > 0) {
            this.startTime = new Date(task.getStartTime());
        }
        if (task.getFinishTime() > 0) {
            this.finishTime = new Date(task.getFinishTime());
        }
        this.status = task.getStatus();
    }
}
