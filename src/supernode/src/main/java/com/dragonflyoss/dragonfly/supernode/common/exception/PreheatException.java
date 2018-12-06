package com.dragonflyoss.dragonfly.supernode.common.exception;

/**
 * @author lowzj
 */
public class PreheatException extends Exception {
    private static final long serialVersionUID = -9209747808980837827L;

    private int code;

    public PreheatException(int code, String msg) {
        super(msg);
        this.code = code;
    }

    public int getCode() {
        return this.code;
    }
}
