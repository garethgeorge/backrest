
## What is this folder?

This is an [unpacked (git-friendly)](https://inlang.com/docs/unpacked-project) inlang project.

## At a glance

Purpose:
- This folder stores inlang project configuration and plugin cache data.
- Translation files live outside this folder and are referenced from `settings.json`.

Safe to edit:
- `settings.json`

Do not edit:
- `cache/`
- `.gitignore`

Key files:
- `settings.json` — locales, plugins, file patterns (source of truth)
- `cache/` — plugin caches (safe to delete)
- `.gitignore` — generated

```
*.inlang/
├── settings.json    # Locales, plugins, and file patterns (source of truth)
├── cache/           # Plugin caches (gitignored)
└── .gitignore       # Ignores everything except settings.json
```

Translation files (like `messages/en.json`) live **outside** this folder and are referenced via plugins in `settings.json`.

## What is inlang?

[Inlang](https://inlang.com) is an open file format for building custom localization (i18n) tooling. It provides:

- **CRUD API** — Read and write translations programmatically via SQL
- **Plugin system** — Import/export any format (JSON, XLIFF, etc.)
- **Version control** — Built-in version control via [lix](https://lix.dev)

```
┌──────────┐        ┌───────────┐         ┌────────────┐
│ i18n lib │        │Translation│         │   CI/CD    │
│          │        │   Tool    │         │ Automation │
└────┬─────┘        └─────┬─────┘         └─────┬──────┘
     │                    │                     │
     └─────────┐          │          ┌──────────┘
               ▼          ▼          ▼
           ┌──────────────────────────────────┐
           │          *.inlang file           │
           └──────────────────────────────────┘
```

## Quick start

```bash
npm install @inlang/sdk
```

```ts
import { loadProjectFromDirectory, saveProjectToDirectory } from "@inlang/sdk";

const project = await loadProjectFromDirectory({ path: "./project.inlang" });
// Query messages with SQLite + [Kysely](https://kysely.dev/) under the hood.
const messages = await project.db.selectFrom("message").selectAll().execute();

// Use project.db to update messages.
await saveProjectToDirectory({ path: "./project.inlang", project });
```

## Ideas for custom tooling

- Translation health dashboard (missing/empty/stale messages)
- Locale coverage report in CI
- Auto-PR for new keys with placeholders
- Migration tool between file formats via plugins
- Glossary/term consistency checker

## Data model ([docs](https://inlang.com/docs/data-model))

```
bundle (a concept, e.g., "welcome_header")
  └── message (per locale, e.g., "en", "de")
        └── variant (plural forms, gender, etc.)
```

- **bundle**: Groups messages by ID (e.g., `welcome_header`)
- **message**: A translation for a specific locale
- **variant**: Handles pluralization/selectors (most messages have one variant)

## Common tasks

- List bundles: `project.db.selectFrom("bundle").selectAll().execute()`
- List messages for locale: `project.db.selectFrom("message").where("locale", "=", "en").selectAll().execute()`
- Find missing translations: compare message counts across locales
- Update a message: `project.db.updateTable("message").set({ ... }).where("id", "=", "...").execute()`

## Links

- [SDK documentation](https://inlang.com/docs)
- [inlang.com](https://inlang.com)
- [List of plugins](https://inlang.com/c/plugins)
- [List of tools](https://inlang.com/c/tools)
