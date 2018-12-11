package com.dragonflyoss.dragonfly.supernode.service.preheat;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.LinkedList;
import java.util.List;
import java.util.Map;
import java.util.concurrent.ScheduledFuture;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

import com.alibaba.fastjson.JSON;
import com.alibaba.fastjson.JSONArray;
import com.alibaba.fastjson.JSONObject;

import com.dragonflyoss.dragonfly.supernode.common.domain.PreheatTask;
import com.dragonflyoss.dragonfly.supernode.common.enumeration.PreheatTaskStatus;
import com.dragonflyoss.dragonfly.supernode.common.exception.PreheatException;
import lombok.AllArgsConstructor;
import lombok.Data;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.apache.logging.log4j.util.Strings;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.HttpEntity;
import org.springframework.http.HttpHeaders;
import org.springframework.http.HttpMethod;
import org.springframework.http.ResponseEntity;
import org.springframework.stereotype.Component;
import org.springframework.web.client.HttpClientErrorException;
import org.springframework.web.client.RestClientException;
import org.springframework.web.client.RestTemplate;

/**
 * @author lowzj
 */
@Component
@Slf4j
public class ImagePreheater extends BasePreheater {
    private static final String TYPE = "image";
    private static final Pattern IMAGE_MANIFESTS_PATTERN = Pattern.compile(
        "^(.*)://(.*)/v2/(.*)/manifests/(.*)");

    @Autowired
    private RestTemplate restTemplate;

    @Override
    public String type() {
        return TYPE;
    }

    @Override
    public BaseWorker newWorker(PreheatTask task, PreheatService service) {
        return new ImagePreheatWorker(task, this, service);
    }

    class ImagePreheatWorker extends BaseWorker {
        private String protocol;
        private String domain;
        private String name;

        ImagePreheatWorker(PreheatTask task, Preheater preheater,
                                  PreheatService service) {
            super(task, preheater, service);
            Matcher m = IMAGE_MANIFESTS_PATTERN.matcher(task.getUrl());
            if (m.matches()) {
                this.protocol = m.group(1);
                this.domain = m.group(2);
                this.name = m.group(3);
            }
        }

        @Override
        boolean preRun() {
            try {
                preheatLayers();
                return true;
            } catch (Exception e) {
                failed(e.getMessage());
            }
            return false;
        }

        @Override
        ScheduledFuture query() {
            Runnable runnable = new Runnable() {
                @Override
                public void run() {
                    PreheatTask task = getTask();
                    int running = task.getChildren().size();
                    for (String child : task.getChildren()) {
                        PreheatTask childTask = getService().get(child);
                        if (childTask.getFinishTime() > 0) {
                            running--;
                        }
                        if (childTask.getStatus() == PreheatTaskStatus.FAILED) {
                            failed(childTask.getUrl() + " " + childTask.getErrorMsg());
                            cancel(task.getId());
                            return;
                        }
                    }
                    if (running <= 0) {
                        succeed();
                        cancel(task.getId());
                    }
                }
            };
            return schedule(getTask().getId(), runnable);
        }

        @Override
        void afterRun() {
            scheduledTasks.remove(getTask().getId());
        }

        void preheatLayers() throws Exception {
            PreheatTask task = getTask();
            List<Layer> layers = getLayers(task.getUrl(), task.getHeaders(), true);

            List<String> children = new LinkedList<>();
            for (Layer layer : layers) {
                String url = layer.getUrl();
                log.info("preheat layer:{} parentId:{}", url, task.getId());
                if (url != null) {
                    PreheatTask child = new PreheatTask();
                    child.setParentId(task.getId());
                    child.setUrl(url);
                    child.setType("file");
                    child.setHeaders(layer.getHeaders());
                    String id;
                    try {
                        id = getService().createPreheatTask(child);
                    } catch (PreheatException e) {
                        if (e.getCode() == 500) {
                            throw e;
                        }
                        id = e.getTaskId();
                        log.warn("create layer preheat task error:{}, parent:{} child:{} ",
                            e.getMessage(), task.getId(), url, e);
                    }
                    if (StringUtils.isNotBlank(id)) {
                        children.add(id);
                    }
                }
            }
            task.setChildren(children);
            task.setStatus(PreheatTaskStatus.RUNNING);
            getService().update(task.getId(), task);
        }

        List<Layer> getLayers(String url, Map<String, String> headerMap, boolean retryIfUnAuth) throws Exception {
            HttpHeaders headers = new HttpHeaders();
            if (headerMap != null) {
                for (Map.Entry<String, String> entry: headerMap.entrySet()) {
                    headers.add(entry.getKey(), entry.getValue());
                }
            }
            HttpEntity entity = new HttpEntity(headers);
            try {
                ResponseEntity<String> res = restTemplate.exchange(
                    url, HttpMethod.GET, entity, String.class);
                if (res.getStatusCode().is2xxSuccessful()) {
                    return parseLayers(res.getBody(), headerMap);
                } else {
                    log.error("getLayers");
                    throw new Exception(res.getStatusCode() + " " + res.getBody());
                }
            } catch (HttpClientErrorException e) {
                if (retryIfUnAuth) {
                    String token = getAuthToken(e.getResponseHeaders());
                    if (!StringUtils.isBlank(token)) {
                        Map<String, String> authHeader = new HashMap<>();
                        authHeader.put("Authorization", "Bearer " + token);
                        return getLayers(url, authHeader, false);
                    }
                }
                log.error("getLayers, url:{} error:{}", url, e.getMessage(), e);
                throw e;
            }
        }

        private List<Layer> parseLayers(String response, Map<String, String> headers) {
            final String schemaVersion = "schemaVersion";
            List<Layer> layers = new LinkedList<>();
            List<String> layerDigest;
            JSONObject json = JSONObject.parseObject(response);
            if ("1".equals(json.getString(schemaVersion))) {
                layerDigest = parseLayers(json, "fsLayers", "blobSum");
            } else {
                layerDigest = parseLayers(json, "layers", "digest");
            }

            for (String digest : layerDigest) {
                Layer layer = new Layer(digest, layerUrl(digest), headers);
                layers.add(layer);
            }
            return layers;
        }

        private List<String> parseLayers(JSONObject json, String layerKey, String digestKey) {
            JSONArray array = json.getJSONArray(layerKey);
            List<String> layers = new ArrayList<>(array.size());
            for (int i = 0; i < array.size(); ++i) {
                JSONObject layer = array.getJSONObject(i);
                layers.add(layer.getString(digestKey));
            }
            return layers;
        }

        private String getAuthToken(HttpHeaders headers) {
            if (headers == null) {
                return null;
            }
            List<String> values = headers.getValuesAsList(HttpHeaders.WWW_AUTHENTICATE);
            String authUrl = authUrl(values);
            if (authUrl == null) {
                return null;
            }
            try {
                String res = restTemplate.getForObject(authUrl, String.class);
                JSONObject resJson = JSON.parseObject(res);
                return resJson.getString("token");
            } catch (RestClientException e) {
                log.error("getAuthToken, authUrl:{} error:{}", authUrl, e.getMessage(), e);
            }
            return null;
        }

        private String authUrl(List<String> wwwAuth) {
            // Bearer realm="<auth-service-url>",service="<service>",scope="repository:<name>:pull"
            if (wwwAuth == null || wwwAuth.isEmpty()) {
                return null;
            }
            List<String> polished = new ArrayList<String>(wwwAuth.size());
            for (String e : wwwAuth) {
                polished.add(e.replaceAll("\"", ""));
            }
            return polished.get(0).split("=")[1]
                + "?"
                + Strings.join(polished.subList(1, polished.size()), '&');
        }

        private String layerUrl(String digest) {
            return protocol + "://" + domain + "/v2/" + name + "/blobs/" + digest;
        }
    }

    @Data
    @AllArgsConstructor
    private class Layer {
        private String digest;
        private String url;
        private Map<String, String> headers;
    }
}
