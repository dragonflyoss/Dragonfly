package com.dragonflyoss.dragonfly.supernode.common.enumeration;

/**
 * ClientErrorType is the error that may happen on client-side when downloading
 * files. Supernode will handle these errors reported by clients.
 *
 * @author lowzj
 */
public enum ClientErrorType {
    /**
     * the md5 calculated by clients doesn't equal to the md5 calculated by supernode
     */
    FILE_MD5_NOT_MATCH,

    /**
     * the downloading file isn't exist
     */
    FILE_NOT_EXIST,
}
