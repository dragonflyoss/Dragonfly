package com.dragonflyoss.dragonfly.supernode.rest.response;

import java.util.Date;

import com.dragonflyoss.dragonfly.supernode.common.domain.PreheatTask;
import com.dragonflyoss.dragonfly.supernode.common.enumeration.PreheatTaskStatus;
import com.fasterxml.jackson.annotation.JsonFormat;
import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.Data;

/**
 * @author lowzj
 */
@Data
@JsonInclude(JsonInclude.Include.NON_NULL)
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
        this.startTime = new Date(task.getStartTime());
        this.finishTime = new Date(task.getFinishTime());
        this.status = task.getStatus();
    }
}
