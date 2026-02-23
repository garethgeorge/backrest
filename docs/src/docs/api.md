# API

Backrest provides a limited HTTP API for interacting with the backrest service. To use the API without a username and password authentication must be disabled. Otherwise, provide a username and password with basic auth headers. e.g. `curl http://localhost:9898/v1/<endpoint> -u USERNAME:PASSWORD`. Usernames and passwords consisting of upper and lower case letters (A-Z, a-z) and numbers (0-9) will work as is. Special characters may need to be escaped. e.g. single quotes in a password `-u 'user:p@ss\'w0rd'`

All of Backrest's API endpoints are defined as a gRPC service and are exposed over HTTP by a JSON RPC gateway for easy scripting. For the full service definition see [service.proto](https://github.com/garethgeorge/backrest/blob/main/proto/v1/service.proto).

::: warning
Only the APIs documented below are considered stable, other endpoints may be subject to change.
:::

### Backup API

The backup API can be used to trigger execution of a plan e.g. 

```
curl -X POST 'localhost:9898/v1.Backrest/Backup' --data '{"value": "YOUR_PLAN_ID"}' -H 'Content-Type: application/json' -u USERNAME:PASSWORD
```
The request will block until the operation has completed. A 200 response means the backup completed successfully, if the request times out the operation will continue in the background.

The backup API can also be used to trigger backup plans sequentially. This hook will start the next backup plan immediately after this plan finishes (or fails):

**Event:** `CONDITION_SNAPSHOT_END`
**Error Behavior:** `ON_ERROR_IGNORE`

```
curl -X POST 'localhost:9898/v1.Backrest/Backup' --data '{"value": "ID_OF_NEXT_PLAN"}' -H 'Content-Type: application/json' -u USERNAME:PASSWORD -m 1 || exit 0
```

`-m 1 || exit 0` tells curl to _not_ wait for a 200 response, instead quickly exit with status 0, which allows the next plan to start running immediately.

### Operations API 

The operations API can be used to fetch operation history e.g. 

```
curl -X POST 'localhost:9898/v1.Backrest/GetOperations' --data '{}' -H 'Content-Type: application/json' -u USERNAME:PASSWORD
```

More complex selectors can be applied e.g. 

```
curl -X POST 'localhost:9898/v1.Backrest/GetOperations' --data '{"selector": {"planId": "YOUR_PLAN_ID"}}' -H 'Content-Type: application/json' -u USERNAME:PASSWORD
```

For details on the structure of operations returned see the [operations.proto](https://github.com/garethgeorge/backrest/blob/main/proto/v1/operations.proto).

::: warning
The structure of the operation history is subject to change over time. Different fields may be added or removed in future versions.
:::
