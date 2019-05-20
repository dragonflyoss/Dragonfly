package com.dragonflyoss.dragonfly.supernode.common.enumeration;

/**
 * @author lowzj
 */
public enum PreheatTaskStatus {
    /**
     * the preheat task is waiting for executing
     */
    WAITING,

    /**
     * the preheat task is running
     */
    RUNNING,

    /**
     * the preheat task is finished and executed successfully
     */
    SUCCESS,

    /**
     * the preheat task is finished and executed failed
     */
    FAILED,
}
