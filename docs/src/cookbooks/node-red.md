# Node-RED

## Overview

[Node-RED](https://nodered.org) is a low-code tool for wiring together devices, APIs and online services in a visual editor. The [@skylord123/node-red-contrib-backrest](https://flows.nodered.org/node/@skylord123/node-red-contrib-backrest) package adds Backrest nodes to it, so you can start backups, get alerts when they fail, react to operations as they happen, or pass backup status on to Home Assistant, Discord, Matrix, email or anything else Node-RED can talk to.

| Node | What it does |
| --- | --- |
| **Run Backup** | Starts a backup of a plan and sends the result when it finishes: successful backups on the first output, failed ones on the second. |
| **Backup Status** | Reports the state of each plan or repository (ok, warning, error, running, overdue), using the same rules as the Backrest dashboard. |
| **Operation Events** | Sends a message when operations start, change status or finish, filtered by plan, repository and operation type. |
| **Get Logs** | Streams the log of an operation, such as a backup or a restic command. |
| **API** | Calls any endpoint of the [Backrest API](../docs/api). |

The recipes below are flows you can paste into Node-RED. The package also comes with examples for every node: in Node-RED open **Import > Examples > @skylord123/node-red-contrib-backrest**, or see them [on GitHub](https://github.com/skylord123/node-red-backrest/tree/master/examples).

## Setup

You need Backrest 1.14.0 or newer and Node-RED 4.0 or newer.

1. In Node-RED, open **Manage palette** from the menu, search the **Install** tab for `backrest` and install `@skylord123/node-red-contrib-backrest`. Or run `npm install @skylord123/node-red-contrib-backrest` in your Node-RED directory (usually `~/.node-red`) and restart Node-RED.
2. Import a recipe: open its flow file (linked under each recipe's picture), copy all of its text, then in Node-RED choose **Import** from the menu, paste it and click **Import**. You can also save the file and pick it with **select a file to import** in the same dialog.
3. For recipes that use the Backrest nodes, double-click any Backrest node, then click the pencil next to **Backrest Server** to open the server config. Enter the **Backrest URL** Node-RED can reach Backrest at (for example `http://192.168.1.10:9898`) and the username and password of a Backrest user. Leave the login empty if Backrest's authentication is disabled. `https://` URLs work too, and the TLS setting lets Node-RED trust a self-signed certificate.
4. Click **Deploy**.

In the recipes, debug nodes named **Send notification** stand in for your notifications: swap them for the node that sends to your phone, chat or email. They receive the alert text in `msg.payload` (and a title in `msg.topic` where there is one).

## Get Notified When a Backup Fails

The **Operation Events** node keeps a connection open to Backrest and sends a message for each backup that finishes, whether it was started by a schedule, from the Backrest UI or through the API. The switch node lets through every backup that didn't succeed (failed, finished with warnings or was cancelled), so the alert arrives as soon as the backup ends and includes Backrest's error message.

![Get notified when a backup fails](/screenshots/node-red-failed-backups.png)

**Flow to import:** [failed-backups.json](/node-red-flows/failed-backups.json)

To be told about other operations too, such as failed prune or check runs, tick them under **Types** in the Operation Events node. If Node-RED loses its connection to Backrest, the node sends the operations that finished in the meantime once it reconnects.

## Check for Failed and Overdue Backups

An event can't tell you about a backup that never ran, for example because the machine was off when it was scheduled. The **Backup Status** node checks every plan once an hour, using the same rules as the Backrest dashboard, and sends only the plans that need attention: their last backup failed or had warnings, or they're overdue. When everything is fine it sends nothing. The filter node passes each problem on once, so you aren't alerted about the same failure every hour.

![Check for failed and overdue backups](/screenshots/node-red-backup-status.png)

**Flow to import:** [backup-status.json](/node-red-flows/backup-status.json)

Set **Status of** to `Repositories` to check repositories instead, or set **Output** to `One message with a list` to get every plan at once, for example to show on a dashboard.

## Start a Backup From a Flow

The **Run Backup** node starts a backup when it receives a message and sends the result when the backup finishes, with the snapshot ID, duration and restic's summary (files and bytes added). Failed backups come out of the second output. While the backup runs, the node shows its progress.

After importing, double-click the Run Backup node and pick the plan from the list (it's loaded from your Backrest server; click the refresh button to reload it). Then trigger it from anything: a dashboard button, a Home Assistant automation, or a flow that backs up a device before updating it.

![Start a backup from a flow](/screenshots/node-red-run-backup.png)

**Flow to import:** [run-backup.json](/node-red-flows/run-backup.json)

To choose the plan in the flow instead, set **Plan** to `msg.plan` and send messages with `msg.plan` set to the plan ID. **Dry run** can be read from the message the same way.

## Stop a Service While a Backup Runs

Some things are best backed up while they're stopped, such as a database or a container that keeps its files open. With two [command hooks](../docs/hooks), Backrest asks Node-RED to stop the service before the backup starts and to start it again when the backup is done. Backrest waits for Node-RED to answer before starting the backup, and cancels the backup if Node-RED reports an error. This recipe uses only Node-RED's built-in nodes.

![Stop a service while a backup runs](/screenshots/node-red-hooks.png)

**Flow to import:** [hooks.json](/node-red-flows/hooks.json)

Import the flow, put what has to happen in the **Stop the service** and **Start the service again** function nodes (or replace them with an exec node, a Docker node or a Home Assistant service call), and deploy. Then edit the plan in Backrest, click **Add Hook**, choose **Command** and add these two hooks. Replace `NODE-RED-HOST` with the address Backrest can reach Node-RED at. If Backrest runs in Docker, that's the host's IP address or a shared Docker network, not `localhost`.

**Hook 1, before the backup**

**Event:** `CONDITION_SNAPSHOT_START`  
**Error Behavior:** `ON_ERROR_CANCEL`

```bash v-pre
curl -fsS -m 300 -X POST -H 'Content-Type: application/json' --data-binary @- http://NODE-RED-HOST:1880/backrest-hook/start <<'JSON'
{"event": {{ .JsonMarshal (.EventName .Event) }}, "plan": {{ .JsonMarshal .Plan.Id }}, "repo": {{ .JsonMarshal .Repo.Id }}, "snapshotId": {{ .JsonMarshal .SnapshotId }}, "error": {{ .JsonMarshal .Error }}}
JSON
```

**Hook 2, after the backup**

**Event:** `CONDITION_SNAPSHOT_END`  
**Error Behavior:** `ON_ERROR_IGNORE`

```bash v-pre
curl -fsS -m 300 -X POST -H 'Content-Type: application/json' --data-binary @- http://NODE-RED-HOST:1880/backrest-hook/end <<'JSON'
{"event": {{ .JsonMarshal (.EventName .Event) }}, "plan": {{ .JsonMarshal .Plan.Id }}, "repo": {{ .JsonMarshal .Repo.Id }}, "snapshotId": {{ .JsonMarshal .SnapshotId }}, "error": {{ .JsonMarshal .Error }}}
JSON
```

Node-RED receives the plan and repository IDs in `msg.payload.plan` and `msg.payload.repo`. After the backup, `msg.payload.snapshotId` has the new snapshot, or `msg.payload.error` the reason it failed. The end hook runs for failed backups too, so the service is always started again, but not when the start hook cancelled the backup.

A few things to keep in mind:

- The backup waits for Node-RED's answer for at most 5 minutes (`-m 300`), so keep the flows short.
- If Node-RED is down or Backrest can't reach it, the start hook fails and the backup is cancelled. To have Backrest try again later instead, set Hook 1's Error Behavior to `ON_ERROR_RETRY_10MINUTES`. To back up anyway, use `ON_ERROR_IGNORE`, but then Node-RED can't cancel backups either.
- Anyone who can reach these Node-RED URLs can trigger the flows. If Node-RED is reachable from untrusted networks, protect its HTTP endpoints (Node-RED's `httpNodeAuth` setting) and add the login to the curl commands with `-u USER:PASSWORD`.

## Send Backup Alerts to a Matrix Room

This recipe posts to a [Matrix](https://matrix.org) room when a backup fails or finishes with warnings, or when a repository check or prune fails. It builds on the [first recipe](#get-notified-when-a-backup-fails): the same **Operation Events** node, set to also report check and prune runs, followed by a function that formats the message and the [Matrix nodes](https://flows.nodered.org/node/node-red-contrib-matrix-chat) to post it. Install `node-red-contrib-matrix-chat` from **Manage palette** before importing.

![Send backup alerts to a Matrix room](/screenshots/node-red-matrix-alerts.png)

**Flow to import:** [matrix-alerts.json](/node-red-flows/matrix-alerts.json)

After importing, set up the Backrest server as described in [Setup](#setup). Then double-click the **Send Message** node, add your Matrix server (the homeserver URL and the login of the account that posts the alerts) and enter the ID of the room to post in, such as `!AbCdEf:example.org`. That account has to be in the room. Deploy, and the next failure shows up like this:

```text
❌ Backup failed

- Plan: documents
- Repository: nas

failed to backup: path /data/documents does not exist
```

Backups with warnings also list the new snapshot, how many files were new or changed, how much data was added and how long the backup took. Backups cancelled from the Backrest UI aren't reported. If Node-RED loses its connection to Backrest, the operations that failed in the meantime are posted once it reconnects.

To post somewhere else, replace the Send Message node with the node for your chat or notification service. It receives the alert as Markdown in `msg.payload`. For email, use the next recipe instead.

## Send Backup Alerts by Email

This recipe sends the same alerts as the Matrix recipe by email: when a backup fails or finishes with warnings, or when a repository check or prune fails. It uses the email node from [node-red-node-email](https://flows.nodered.org/node/node-red-node-email), which you can install from **Manage palette**.

![Send backup alerts by email](/screenshots/node-red-email-alerts.png)

**Flow to import:** [email-alerts.json](/node-red-flows/email-alerts.json)

After importing, set up the Backrest server as described in [Setup](#setup). Then double-click the **Send email** node and fill in:

- **To:** the address to send the alerts to.
- **Server** and **Port:** your mail provider's SMTP server, such as `smtp.gmail.com` and port `465`. Leave **Use secure connection** ticked for port 465, and untick it for port 587.
- **Userid** and **Password:** the login for the SMTP server. The mail is sent from this address. Many providers, Gmail among them, need an app password here instead of your normal one.

The subject says what happened, such as `Backrest: Backup of database failed` or `Backrest: Repository check of nas failed`. The mail lists the plan, repository and error, and for backups with warnings also the snapshot, how many files were new or changed, how much data was added and how long the backup took. Mail apps that don't show HTML get the same details as plain text.

## Home Assistant

With the [Home Assistant nodes](https://flows.nodered.org/node/node-red-contrib-home-assistant-websocket) for Node-RED, the recipes above can drive Home Assistant entities: send the Backup Status output to a sensor node to show each plan's state on a dashboard, or wire a button entity to the Run Backup node to start a backup from Home Assistant.

## More Ideas

- Attach the backup's log to a failure alert: the **Get Logs** node takes the result of the Run Backup node (or an operation from Operation Events) and streams that operation's log.
- Use the **API** node for anything the other nodes don't cover, such as listing snapshots, browsing their files, restoring, or running a restic command. When you pick an endpoint, the node shows an example of its input.
