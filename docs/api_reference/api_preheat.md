---
title: "Preheat API"
weight: 5
---

This topic explains how to use the Preheat API.
<!--more-->

## GET /api/check

> check whether the connection to Dragonfly is available

* **Parameters**
* **Response**: `Content-type: application/json`
  <table width="100%">
  <thead><tr><th>HTTP Code</th><th>Response Body</th></tr></thead>
  <tbody>
  <tr>
  <td>200</td>
  <td>
  <pre>
  {
    "code": 200
  }
  </pre>
  </td>
  </tr>
  </tbody></table>

## POST /api/preheat

> request to Dragonfly to start a preheat task

* **Parameters**: `Content-type: application/json`
  <table width="100%">
  <thead><tr><th>Parameter Type</th><th>Data Type</th></tr></thead>
  <tbody>
  <tr><td>body</td>
  <td><pre>
  {
    "type": "image|file",
    "url": "&lt;string&gt;",
    "header": {
      "&lt;name&gt;": "&lt;value&gt;"
    }
  }
  </pre>Dragonfly sends a request taking the 'header' to the 'url'.</td></tr>
  </tbody></table>

  If there is any authentication step of the remote server, the `header` should contain authenticated information.

  If the `type` is `image`, then the `url` should be image url: `<registry_host>/<image_name>:<image_tag>`.
  Dragonfly will preheat the image according to [registry API spec](https://docs.docker.com/registry/spec/api/#pulling-an-image), the steps are:
  * construct `manifest_url`:

      ```
      https://<harbor_host>/v2/<image_name>/manifests/<image_tag>
      ```

  * pull the manifest of the image from `manifest_url`
  * get the `fsLayers` from manifest and construct `layer_url` of each layer:

      ```
      https://<harbor_host>/v2/<name>/blobs/<digest>
      ```

  * request these `layer_url`s above to handle any redirection response to get real downloading urls
  * supernodes use these real downloading urls to preheat layers of this image

* **Response**: `Content-type: application/json`
  <table width="100%">
  <thead><tr><th>HTTP Code</th><th>Response Body</th></tr></thead>
  <tbody>
  <tr><td>200</td>
  <td>Success response:<pre>
  {
    "code": 200,
    "data": {
        "taskId": "&lt;string&gt;"
    }
  }
  </pre>Use 'taskId' to query the status of the preheat task.</td></tr>
  <tr><td>200</td>
  <td>Error Response:<pre>
  {
    "code": 400,
    "msg": "&lt;detailed error message&gt;"
  }
  </pre></td></tr>
  </tbody></table>

## GET /api/preheat/{taskId}

> query the current status of the preheat task which id is `taskId`

* **Response**: `Content-type: application/json`
  <table width="100%">
  <thead><tr><th>HTTP Code</th><th>Response Body</th></tr></thead>
  <tbody>
  <tr><td>200</td>
  <td>Success response:<pre>
  {
    "code": 200,
    "data": {
        "taskId": "&lt;string&gt;",
        "status": "RUNNING|SUCCESS|FAIL"
    }
  }
  </pre></td></tr>
  <tr><td>200</td>
  <td>Error Response:<pre>
  {
    "code": 400,
    "msg": "&lt;detailed error message&gt;"
  }
  </pre></td></tr>
  </tbody></table>
