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

import java.io.IOException;
import java.net.HttpURLConnection;
import java.net.MalformedURLException;
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

import com.alibaba.dragonfly.supernode.common.exception.AuthenticationRequiredException;
import com.alibaba.dragonfly.supernode.common.exception.UrlNotReachableException;

import org.apache.commons.lang3.StringUtils;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class HttpClientUtil {
    private static final Logger logger = LoggerFactory.getLogger(HttpClientUtil.class);

    public static final List<Integer> REDIRECTED_CODE = Arrays.asList(301, 302, 303, 307);

    static {
        HttpURLConnection.setFollowRedirects(false);
    }

    private static class TrustAnyTrustManager implements X509TrustManager {
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

    public static boolean isExpired(String fileUrl, long lastModified, String[] headers) throws MalformedURLException {
        if (lastModified <= 0) {
            return true;
        }
        int times = 2;
        HttpURLConnection conn = null;
        URL url = new URL(fileUrl);
        int code;
        while (times-- > 0) {
            try {
                conn = openConnection(url);
                fillHeaders(conn, headers);
                conn.setUseCaches(false);
                conn.setConnectTimeout(2000);
                conn.setReadTimeout(1000);
                conn.setIfModifiedSince(lastModified);

                conn.connect();
                code = conn.getResponseCode();
                if (REDIRECTED_CODE.contains(code)) {
                    fileUrl = conn.getHeaderField("Location");
                    if (StringUtils.isNotBlank(fileUrl)) {
                        return isExpired(fileUrl, lastModified, null);
                    }
                }
                if (code == HttpURLConnection.HTTP_NOT_MODIFIED) {
                    return false;
                }
                break;
            } catch (Exception e) {
                logger.warn("url:{} isExpired error", fileUrl, e);
            } finally {
                closeConn(conn);
                conn = null;
            }
        }
        return true;
    }

    public static long getContentLength(String fileUrl, String[] headers, boolean dfdaemon)
        throws MalformedURLException, UrlNotReachableException, AuthenticationRequiredException {
        int times = 2;
        URL url = new URL(fileUrl);
        HttpURLConnection conn = null;
        int code;
        while (times-- > 0) {
            try {
                conn = openConnection(url);
                conn.setUseCaches(false);
                fillHeaders(conn, headers);
                conn.setConnectTimeout(2000);
                conn.setReadTimeout(2000);
                try {
                    conn.connect();
                    code = conn.getResponseCode();
                    if (dfdaemon && (code == HttpURLConnection.HTTP_UNAUTHORIZED
                        || code == HttpURLConnection.HTTP_PROXY_AUTH)) {
                        throw new AuthenticationRequiredException();
                    }
                    if (REDIRECTED_CODE.contains(code)) {
                        fileUrl = conn.getHeaderField("Location");
                        if (StringUtils.isNotBlank(fileUrl)) {
                            return getContentLength(fileUrl, null, dfdaemon);
                        }
                    }
                    if (code == HttpURLConnection.HTTP_OK) {
                        return conn.getContentLengthLong();
                    }
                } catch (AuthenticationRequiredException e) {
                    throw e;
                } catch (Exception e) {
                    if (times < 1) {
                        logger.error("connect to url:{} error", fileUrl, e);
                        throw new UrlNotReachableException();
                    }
                }

            } catch (UrlNotReachableException e) {
                throw e;
            } catch (AuthenticationRequiredException e) {
                throw e;
            } catch (Exception e) {
                if (times < 1) {
                    logger.warn("url:{} getContentLength error", fileUrl, e);
                }
            } finally {
                closeConn(conn);
                conn = null;
            }
        }
        return -1;
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

    public static boolean isSupportRange(String fileUrl, String[] headers) throws MalformedURLException {
        int times = 2;
        URL url = new URL(fileUrl);
        HttpURLConnection conn = null;
        int code;
        while (times-- > 0) {
            try {
                conn = openConnection(url);
                conn.setUseCaches(false);
                conn.setRequestProperty("Range", "bytes=0-0");
                fillHeaders(conn, headers);
                conn.setInstanceFollowRedirects(true);
                HttpURLConnection.setFollowRedirects(true);
                conn.setConnectTimeout(2000);
                conn.setReadTimeout(1000);

                conn.connect();
                code = conn.getResponseCode();
                if (REDIRECTED_CODE.contains(code)) {
                    fileUrl = conn.getHeaderField("Location");
                    if (StringUtils.isNotBlank(fileUrl)) {
                        return isSupportRange(fileUrl, null);
                    }
                }
                return code == HttpURLConnection.HTTP_PARTIAL;
            } catch (Exception e) {
                logger.warn("url:{} isSupportRange error", fileUrl, e);
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

    /**
     * @param conn
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
        HttpURLConnection conn = null;
        if (url.getProtocol().toLowerCase().equals("https")) {
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
}
