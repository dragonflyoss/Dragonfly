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
package com.alibaba.dragonfly.supernode.common.util;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class RangeParseUtil {

    private static final String SEPERATOR = "-";
    private static final Logger logger = LoggerFactory.getLogger(RangeParseUtil.class);

    /**
     * @param range
     * @return
     */
    public static int calculatePieceNum(String range) {
        try {
            String[] rangeArr = range.split(SEPERATOR);
            long startIndex = Long.parseLong(rangeArr[0]);
            long endIndex = Long.parseLong(rangeArr[1]);
            long pieceSize = endIndex - startIndex + 1;
            return (int)(startIndex / pieceSize);
        } catch (Exception e) {
            logger.error("E_caculatePieceNum for range:{}", range, e);
        }
        return -1;
    }

    /**
     * @param startPieceNum
     * @return
     */
    public static String calculateBreakRange(int startPieceNum, long rangeLength, int pieceContSize) {
        if (startPieceNum <= 0) {
            throw new IllegalArgumentException("startPieceNum is illegal for value:" + startPieceNum);
        }

        if (rangeLength <= 0) {
            throw new IllegalArgumentException("rangeLength is illegal for value:" + rangeLength);
        }

        long start = startPieceNum * (long)pieceContSize;
        long end = rangeLength - 1;
        if (start > end) {
            throw new IndexOutOfBoundsException("start:" + start + " larger than end:" + end);
        }
        return start + SEPERATOR + end;
    }

    public static String calculatePieceRange(int pieceNum, Integer pieceSize) {
        long startIndex = pieceNum * (long)pieceSize;
        long endIndex = startIndex + pieceSize - 1;
        return startIndex + SEPERATOR + endIndex;
    }

}
