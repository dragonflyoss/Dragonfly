# Preheating API

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
  </tbody
  </table>
  

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
    "url": "string",
    "header": ["string"]
  }
  </pre>Dragonfly sends a request taking the 'header' to the 'url'.</td></tr>
  </tbody
  </table>

* **Response**: `Content-type: application/json`
  <table width="100%">
  <thead><tr><th>HTTP Code</th><th>Response Body</th></tr></thead>
  <tbody>
  <tr><td>200</td>
  <td>Success response:<pre>
  {
    "code": 200,
    "data": {
        "taskId": "string"
    }
  }
  </pre>Use 'taskId' to query the status of the preheat task.</td></tr>
  <tr><td>200</td>
  <td>Error Response:<pre>
  {
    "code": 400,
    "msg": "detailed error message"
  }
  </pre></td></tr>
  </tbody

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
        "taskId": "string",
        "status": "RUNNING|SUCCESS|FAIL"
    }
  }
  </pre></td></tr>
  <tr><td>200</td>
  <td>Error Response:<pre>
  {
    "code": 400,
    "msg": "detailed error message"
  }
  </pre></td></tr>
  </tbody
