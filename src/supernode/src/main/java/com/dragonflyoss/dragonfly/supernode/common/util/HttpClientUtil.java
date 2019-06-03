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
package com.dragonflyoss.dragonfly.supernode.common.util;

import java.io.IOException;
import java.net.HttpURLConnection;
import java.net.URL;
import java.security.KeyManagementException;
import java.security.NoSuchAlgorithmException;
import java.security.cert.CertificateException;
import java.security.cert.X509Certificate;
import java.util.Arrays;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import javax.net.ssl.HostnameVerifier;
import javax.net.ssl.HttpsURLConnection;
import javax.net.ssl.SSLContext;
import javax.net.ssl.SSLSession;
import javax.net.ssl.TrustManager;
import javax.net.ssl.X509TrustManager;

import com.dragonflyoss.dragonfly.supernode.common.exception.AuthenticationRequiredException;
import com.dragonflyoss.dragonfly.supernode.common.exception.UrlNotReachableException;

import org.apache.commons.lang3.StringUtils;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class HttpClientUtil {
    private static final Logger logger = LoggerFactory.getLogger(HttpClientUtil.class);

    public static final List<Integer> REDIRECTED_CODE = Arrays.asList(301, 302, 303, 307);

    static {
        HttpURLConnection.setFollowRedirects(false);
    }

    public static class TrustAnyTrustManager implements X509TrustManager {
        @Override
        public void checkClientTrusted(X509Certificate[] chain, String authType) throws CertificateException {}

        @Override
        public void checkServerTrusted(X509Certificate[] chain, String authType) throws CertificateException {}

        @Override
        public X509Certificate[] getAcceptedIssuers() {
            return new X509Certificate[] {};
        }
    }

    private static class TrustAnyHostnameVerifier implements HostnameVerifier {
        @Override
        public boolean verify(String hostname, SSLSession session) {
            return true;
        }
    }

    public static boolean isExpired(final String url,
                                    final long lastModified, final String eTag,
                                    final String[] headers) {
        if (lastModified <= 0 && eTag == null) {
            return true;
        }
        int times = 2;
        HttpURLConnection conn = null;
        BaseRedirectHandler handler = new BaseRedirectHandler() {
            @Override
            void init(HttpURLConnection conn) {
                conn.setUseCaches(false);
                conn.setConnectTimeout(2000);
                conn.setReadTimeout(2000);
                if (lastModified > 0) {
                    conn.setIfModifiedSince(lastModified);
                }
                if (eTag != null) {
                    conn.setRequestProperty("If-None-Match", eTag);
                }
            }
        };
        while (times-- > 0) {
            try {
                conn = handler.connect(url, headers);
                return conn.getResponseCode() != HttpURLConnection.HTTP_NOT_MODIFIED;
            } catch (Exception e) {
                logger.warn("url:{} isExpired error:{}", url, e.getMessage(), e);
                if (times == 0) {
                    return false;
                }
            } finally {
                closeConn(conn);
                conn = null;
            }
        }
        return true;
    }

    public static long getContentLength(final String url, final String[] headers, final boolean dfdaemon)
        throws UrlNotReachableException, AuthenticationRequiredException {
        int times = 2;
        HttpURLConnection conn = null;
        BaseRedirectHandler handler = new BaseRedirectHandler() {
            @Override
            void init(HttpURLConnection conn) {
                conn.setUseCaches(false);
                conn.setConnectTimeout(2000);
                conn.setReadTimeout(2000);
            }
        };
        while (times-- > 0) {
            try {
                try {
                    conn = handler.connect(url, headers);
                    int code = conn.getResponseCode();
                    if (dfdaemon && (code == HttpURLConnection.HTTP_UNAUTHORIZED
                        || code == HttpURLConnection.HTTP_PROXY_AUTH)) {
                        throw new AuthenticationRequiredException();
                    }

                    if (code == HttpURLConnection.HTTP_OK) {
                        return conn.getContentLengthLong();
                    }
                } catch (AuthenticationRequiredException e) {
                    throw e;
                } catch (Exception e) {
                    if (times < 1) {
                        logger.error("connect to url:{} error", url, e);
                        throw new UrlNotReachableException(e.getMessage());
                    }
                }

            } catch (UrlNotReachableException | AuthenticationRequiredException e) {
                throw e;
            } catch (Exception e) {
                if (times < 1) {
                    logger.warn("url:{} getContentLength error", url, e);
                }
            } finally {
                closeConn(conn);
                conn = null;
            }
        }
        return -1;
    }

    public static boolean isSupportRange(String url, String[] headers) {
        int times = 2;
        HttpURLConnection conn = null;

        BaseRedirectHandler handler = new BaseRedirectHandler() {
            @Override
            void init(HttpURLConnection conn) {
                conn.setUseCaches(false);
                conn.setRequestProperty("Range", "bytes=0-0");
                conn.setInstanceFollowRedirects(true);
                HttpURLConnection.setFollowRedirects(true);
                conn.setConnectTimeout(2000);
                conn.setReadTimeout(1000);
            }
        };
        while (times-- > 0) {
            try {
                conn = handler.connect(url, headers);
                return conn.getResponseCode() == HttpURLConnection.HTTP_PARTIAL;
            } catch (Exception e) {
                logger.warn("url:{} isSupportRange error", url, e);
            } finally {
                closeConn(conn);
                conn = null;
            }
        }
        return false;
    }

    private static String[] split(String field, String reg) {
        String[] result = null;
        if (StringUtils.isNotBlank(field)) {
            String[] arr = field.split(reg, 2);
            result = new String[2];
            result[0] = arr[0].trim();
            if (arr.length == 2) {
                result[1] = arr[1].trim();
            } else {
                result[1] = "";
            }
        }
        return result;
    }

    private static void closeConn(HttpURLConnection conn) {
        try {
            if (conn != null) {
                conn.disconnect();
            }
        } catch (Exception e) {
            logger.warn("E_disconnect", e);
        }
    }

    /**
     * @param conn an opened connection
     * @param headers:["a:b","c:d"]
     */
    public static void fillHeaders(HttpURLConnection conn, String[] headers) {
        Map<String, String> result = parseHeader(headers);
        if (result != null && !result.isEmpty()) {
            for (String headKey : result.keySet()) {
                conn.setRequestProperty(headKey, result.get(headKey));
            }
        }

    }

    private static Map<String, String> parseHeader(String[] headers) {
        Map<String, String> result = new HashMap<>();
        if (headers != null) {
            String existValue;
            String[] oneHeaderArr;
            for (String oneHeader : headers) {
                oneHeaderArr = split(oneHeader, "\\s*:\\s*");
                if (oneHeaderArr != null) {
                    existValue = result.get(oneHeaderArr[0]);
                    if (StringUtils.isNotBlank(existValue)) {
                        if (StringUtils.isNotBlank(oneHeaderArr[1])) {
                            existValue += "," + oneHeaderArr[1];
                        }

                    } else {
                        existValue = oneHeaderArr[1];
                    }
                    result.put(oneHeaderArr[0], existValue);
                }
            }
        }
        return result;
    }

    public static HttpURLConnection openConnection(URL url)
        throws NoSuchAlgorithmException, KeyManagementException, IOException {
        HttpURLConnection conn;
        if ("https".equals(url.getProtocol().toLowerCase())) {
            SSLContext sc = SSLContext.getInstance("TLS");
            sc.init(null, new TrustManager[] {new TrustAnyTrustManager()},
                new java.security.SecureRandom());
            HttpsURLConnection https = (HttpsURLConnection)url.openConnection();
            https.setSSLSocketFactory(sc.getSocketFactory());
            https.setHostnameVerifier(new TrustAnyHostnameVerifier());
            conn = https;
        } else {
            conn = (HttpURLConnection)url.openConnection();
        }
        return conn;
    }

    private static abstract class BaseRedirectHandler {
        static final int MAX_REDIRECT_TIMES = 20;

        /**
         * init an opened connection
         * @param conn an opened connection
         */
        abstract void init(HttpURLConnection conn);

        private HttpURLConnection open(String url, String[] headers)
            throws IOException, NoSuchAlgorithmException, KeyManagementException {
            HttpURLConnection conn = openConnection(new URL(url));
            fillHeaders(conn, headers);
            init(conn);
            return conn;
        }

        HttpURLConnection connect(String url, String[] headers)
            throws IOException, KeyManagementException, NoSuchAlgorithmException, UrlNotReachableException {
            HttpURLConnection conn = null;
            try {
                for (int i = 0; i < MAX_REDIRECT_TIMES; i++) {
                    conn = open(url, headers);
                    conn.connect();
                    int code = conn.getResponseCode();
                    if (!REDIRECTED_CODE.contains(code)) {
                        return conn;
                    }
                    headers = null;
                    url = conn.getHeaderField("Location");
                    closeConn(conn);
                    conn = null;
                    if (StringUtils.isBlank(url)) {
                        throw new UrlNotReachableException(code + ": redirect but no location");
                    }
                }
            } catch (Exception e) {
                closeConn(conn);
                throw e;
            }
            throw new UrlNotReachableException("stopped after " + MAX_REDIRECT_TIMES + " redirects");
        }

    }
}
