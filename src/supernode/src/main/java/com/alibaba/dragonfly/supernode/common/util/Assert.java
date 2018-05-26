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

import java.util.List;

import com.alibaba.dragonfly.supernode.common.exception.AssertException;
import com.alibaba.fastjson.JSON;

import org.apache.commons.lang3.StringUtils;
import org.springframework.util.CollectionUtils;

public final class Assert {

    public final static void assertNotEmpty(String str, int errCode,
        String errorMessage) throws AssertException {
        if (StringUtils.isEmpty(str)) {
            fail(errCode, errorMessage);
        }
    }

    public final static void assertNotEmpty(List<?> list, int errCode,
        String errorMessage) throws AssertException {
        if (CollectionUtils.isEmpty(list)) {
            fail(errCode, errorMessage);
        }
    }

    public final static void assertNotNull(Object obj, int errCode,
        String errorMessage) throws AssertException {
        if (obj == null) {
            fail(errCode, errorMessage);
        }
    }

    public final static void assertNull(Object obj, int errCode,
        String errorMessage) throws AssertException {
        if (obj != null) {
            fail(errCode, errorMessage);
        }
    }

    public final static void assertTrue(boolean val, int errCode,
        String errorMessage) throws AssertException {
        if (!val) {
            fail(errCode, errorMessage);
        }
    }

    public final static void assertPositive(Long val, int errCode,
        String errorMessage) throws AssertException {
        if (val == null || val.longValue() <= 0) {
            fail(errCode, errorMessage);
        }
    }

    public final static void assertFalse(boolean val, int errCode,
        String errorMessage) throws AssertException {
        if (val) {
            fail(errCode, errorMessage);
        }
    }

    public final static void assertJsonArray(String val, Class<?> cls,
        int errCode, String errorMessage) throws AssertException {

        boolean valid = false;
        try {
            JSON.parseArray(val, cls);
            valid = true;
        } catch (Exception e) {

        }

        if (!valid) {
            fail(errCode, errorMessage);
        }
    }

    public final static void assertNumeric(String val, int errCode,
        String errorMessage) {
        boolean valid = StringUtils.isNumeric(val);
        if (!valid) {
            fail(errCode, errorMessage);
        }
    }

    private final static void fail(int errCode, String errorMessage)
        throws AssertException {
        throw new AssertException(errCode, errorMessage);
    }
}
