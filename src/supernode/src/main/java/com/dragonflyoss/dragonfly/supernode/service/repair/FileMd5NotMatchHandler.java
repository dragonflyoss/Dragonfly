package com.dragonflyoss.dragonfly.supernode.service.repair;

import java.io.Closeable;
import java.io.IOException;
import java.io.RandomAccessFile;
import java.nio.file.Path;
import java.security.MessageDigest;
import java.util.List;

import com.dragonflyoss.dragonfly.supernode.common.domain.ClientErrorInfo;
import com.dragonflyoss.dragonfly.supernode.common.domain.FileMetaData;
import com.dragonflyoss.dragonfly.supernode.common.enumeration.ClientErrorType;
import com.dragonflyoss.dragonfly.supernode.common.util.RangeParseUtil;
import com.dragonflyoss.dragonfly.supernode.service.cdn.FileMetaDataService;
import com.dragonflyoss.dragonfly.supernode.service.cdn.util.PathUtil;
import com.dragonflyoss.dragonfly.supernode.service.timer.DataGcService;

import lombok.extern.slf4j.Slf4j;
import org.apache.commons.codec.binary.Hex;
import org.apache.commons.codec.digest.DigestUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Component;

/**
 * This class only handle the errors that are caused by downloading from supernode.
 * It checks the piece md5 on local cache, verify it and delete the related task
 * if the md5 is not expected.
 *
 * @author lowzj
 */
@Component
@Slf4j
public class FileMd5NotMatchHandler extends BaseClientErrorHandler {

    @Autowired
    private FileMetaDataService fileMetaDataService;

    @Autowired
    private DataGcService gcService;

    @Override
    ClientErrorType errorType() {
        return ClientErrorType.FILE_MD5_NOT_MATCH;
    }

    @Override
    void doHandle(final ClientErrorInfo info) {
        boolean gc = false;
        String md5FromMeta = getMd5FromMeta(info.getTaskId(), info.getRange());
        String localMd5 = getMd5(info.getTaskId(), info.getRange());
        if (StringUtils.isNotBlank(localMd5)) {
            if (!StringUtils.equals(md5FromMeta, localMd5) // data in supernode is not right
                && !StringUtils.equals(localMd5, info.getExpectedMd5())) {
                gcService.gcOneTask(info.getTaskId());
                gc = true;
            }
        }
        log.info("taskId:{} range:{} realMd5:{} expectedMd5:{} localMd5:{} metaMd5:{} gc:{}",
            info.getTaskId(), info.getRange(), info.getRealMd5(), info.getExpectedMd5(),
            localMd5, md5FromMeta, gc);
    }

    @Override
    boolean isInvalidInfo(ClientErrorInfo info) {
        return super.isInvalidInfo(info)
            || StringUtils.isAnyBlank(
                info.getDstIp(), info.getRange(), info.getExpectedMd5(), info.getRealMd5());
    }

    private String getMd5(String taskId, String range) {
        long[] pieceRange = RangeParseUtil.calculatePieceRange(range);
        if (pieceRange == null) {
            return null;
        }

        Path uploadPath = PathUtil.getUploadPath(taskId);
        RandomAccessFile raf = null;
        try {
            raf = new RandomAccessFile(uploadPath.toFile(), "r");
            return readMd5(raf, pieceRange[0], pieceRange[1]);
        } catch (IOException e) {
            log.error("getMd5 taskId:{} range:{} error:{}", taskId, range, e.getMessage(), e);
        } finally {
            closeFile(raf);
        }
        return null;
    }

    private String readMd5(RandomAccessFile raf, long start, long end) throws IOException {
        final int size = 4 * 1024 * 1024;
        final byte[] buff = new byte[size];

        MessageDigest pieceMd5 = DigestUtils.getMd5Digest();
        int remain = (int)(end - start + 1);
        raf.seek(start);
        do {
            int readLen = remain > size ? size : remain;
            readLen = raf.read(buff, 0, readLen);
            log.debug("readMd5, remain:{} readLen:{}", remain, readLen);
            if (readLen > 0) {
                pieceMd5.update(buff, 0, readLen);
                remain -= readLen;
            } else {
                break;
            }
        } while (remain > 0);
        return Hex.encodeHexString(pieceMd5.digest());
    }

    private void closeFile(Closeable c) {
        if (c != null) {
            try {
                c.close();
            } catch (IOException e) {
                // pass
            }
        }
    }

    private String getMd5FromMeta(String taskId, String range) {
        int pieceNum = RangeParseUtil.calculatePieceNum(range);
        if (pieceNum < 0) {
            return null;
        }

        FileMetaData meta = fileMetaDataService.readFileMetaData(taskId);
        if (meta == null) {
            return null;
        }
        List<String> pieceMd5s = fileMetaDataService.readPieceMd5(taskId, meta.getRealMd5());

        return pieceMd5s.size() > pieceNum ? pieceMd5s.get(pieceNum) : null;
    }
}
