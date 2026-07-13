# Notifications

Backrest can notify you when backups succeed, fail, or need attention. Notifications are built on the [hooks system](/docs/hooks): a hook pairs one or more trigger *conditions* (e.g. "snapshot succeeded") with an *action* (e.g. "post to Discord"). This guide walks through setting up the most common notification services; the [Hooks reference](/docs/hooks) documents every condition, error policy, and template variable exhaustively.

## How Notifications Work

Hooks can be attached to a **repository** (fires for every plan using that repo, plus repo-level operations like prune and check) or to an individual **plan** (fires only for that plan's operations). You configure them in the same modal used to edit the repo or plan: scroll to the **Hooks** section, add a hook, and pick an action type:

- **Discord**, **Slack**, **Gotify**, **Telegram**, **Healthchecks** — first-class integrations, each needing only a URL/token.
- **Shoutrrr** — a multi-provider gateway covering dozens of services (ntfy, Pushover, email, Matrix, and more).
- **Webhook** and **Command** — generic options for services without a dedicated integration; see the [command hook cookbook](/cookbooks/command-hook-examples).

Every notification body is rendered from a Go template. If you leave the template field empty, Backrest uses <code v-pre>{{ .Summary }}</code>, a sensible default that includes the task name, event, and (for backups) snapshot statistics.

Each hook execution is recorded as an operation in the UI, so you can see when hooks ran and read their logs if delivery fails.

## Choosing Conditions

A good baseline for a notification hook is:

- `CONDITION_ANY_ERROR` — any operation on this plan/repo failed. This is the most useful condition; if you configure only one hook, use this.
- `CONDITION_SNAPSHOT_SUCCESS` (or `CONDITION_SNAPSHOT_END` to also cover failures) — confirmation that backups are running as expected.

::: tip Avoid notification fatigue
Skip `CONDITION_SNAPSHOT_START` for chat notifications; on an hourly schedule it doubles message volume without adding useful information. Start conditions are mainly useful for command hooks (e.g. mounting a drive before backup) and for Healthchecks-style dead man's switches, which measure the time between start and success pings.
:::

`CONDITION_SNAPSHOT_WARNING` is also worth adding: it fires when a backup completes but some files could not be read (a *partial* backup), which `ANY_ERROR` does not cover.

## Walkthrough: Discord

1. In Discord, open the target channel's settings → **Integrations** → **Webhooks** → **New Webhook**, and copy the webhook URL.
2. In Backrest, edit the plan you want notifications for (or the repo, to cover all its plans) and scroll to the **Hooks** section.
3. Add a hook and choose the **Discord** action type.
4. Paste the webhook URL.
5. Select conditions: `CONDITION_ANY_ERROR` and `CONDITION_SNAPSHOT_SUCCESS`.
6. Leave the template empty to use the default summary, then submit the modal to save.

<img src="/screenshots/discord-hook.png" alt="Hooks section of the plan modal with a Discord hook configured" style="max-width: 100%; border-radius: 8px; margin-bottom: 20px;">

Trigger a manual backup with **Backup Now** — you should see a message in your Discord channel when the snapshot completes.

## Quick Recipes for Other Services

### Gotify

Choose the **Gotify** action and provide the base URL of your Gotify server (e.g. `https://gotify.example.com`) and an application token (create one under **Apps** in the Gotify UI). You can optionally set a title template and a Gotify priority level to control how intrusively the notification is delivered.

### Telegram

1. Create a bot by messaging [@BotFather](https://t.me/botfather) and copy the bot token.
2. Start a chat with your bot (or add it to a group), then find the chat ID. For a direct chat, messaging [@userinfobot](https://t.me/userinfobot) is a quick way to get your numeric ID.
3. Choose the **Telegram** action in Backrest and enter the bot token and chat ID.

::: info
Telegram messages are sent with HTML parse mode, so literal `<` and `>` characters in a custom template must be escaped as `&lt;` and `&gt;`.
:::

### ntfy, Pushover, Email, and More via Shoutrrr

The **Shoutrrr** action delivers to any service supported by the [Shoutrrr](https://containrrr.dev/shoutrrr/) notification library, including ntfy, Pushover, SMTP email, Matrix, Pushbullet, and many others. Configuration is a single service URL such as `ntfy://...` or `smtp://...`; see the Shoutrrr documentation for the URL syntax of each service.

### Healthchecks (Dead Man's Switch)

Push notifications cannot alert you when backups stop happening entirely (host offline, Backrest not running). For that case, use a service that expects periodic pings and alerts on their absence. The **Healthchecks** action integrates natively with [Healthchecks.io](https://healthchecks.io/) or a self-hosted instance:

1. Create a check in Healthchecks and copy its ping URL.
2. Add a **Healthchecks** hook with that URL and select `CONDITION_SNAPSHOT_START`, `CONDITION_SNAPSHOT_SUCCESS`, and `CONDITION_SNAPSHOT_ERROR`.

Backrest automatically appends the right endpoint per event (`/start` for start events, `/fail` for errors, the base URL for success) and sends the rendered summary as the ping body, so error details are readable in the Healthchecks dashboard. See the [Hooks reference](/docs/hooks#healthchecks-io-integration) for details.

::: info
The [command hook cookbook](/cookbooks/command-hook-examples) shows an equivalent `curl`-based approach; that is the manual alternative for services without a native hook type.
:::

### Slack

Choose the **Slack** action and paste an [incoming webhook URL](https://api.slack.com/messaging/webhooks). For richly formatted messages using Slack's Block Kit layout, see the [Slack hook cookbook](/cookbooks/slack-hook-build-kit-examples).

## Testing Your Hooks

The simplest test is to click **Backup Now** on a plan with a `CONDITION_SNAPSHOT_SUCCESS` (or `_END`) hook attached. Hook executions appear as operations in the operation list. If a notification does not arrive, open the hook operation there to read its error output; common causes are a mistyped webhook URL or a template syntax error.

## Customizing Messages

Templates use Go template syntax. Commonly used variables:

| Variable | Meaning |
| --- | --- |
| <code v-pre>{{ .Summary }}</code> | The full default message — a good starting point |
| <code v-pre>{{ .Task }}</code> | Name of the task that fired the hook |
| <code v-pre>{{ .Error }}</code> | Error message (empty on success) |
| <code v-pre>{{ .FormatDuration .Duration }}</code> | How long the operation took |
| <code v-pre>{{ .SnapshotStats }}</code> | Backup statistics (files processed, data added, ...) |

For example, a compact success/failure template:

<div v-pre>

```text
{{ if .Error }}❗ Backrest: {{ .Task }} failed: {{ .Error }}
{{ else }}✅ Backrest: {{ .Task }} finished in {{ .FormatDuration .Duration }}
{{ if .SnapshotStats }}Added {{ .FormatSizeBytes .SnapshotStats.DataAdded }} in snapshot {{ .SnapshotId }}{{ end }}
{{ end }}
```
</div>

See the [Hooks reference](/docs/hooks#template-system) for the complete list of variables and helper functions, and the [error handling policies](/docs/hooks#error-handling) that control what happens when a hook itself fails.
