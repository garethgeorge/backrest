
## What is this folder?

This is an [unpacked (git-friendly)](https://inlang.com/docs/unpacked-project) inlang project.

## At a glance

Purpose:
- This folder is the Git-friendly representation of an `.inlang` project.
- The canonical `.inlang` format is a single binary file; this directory is the unpacked version for Git.
- This folder stores project configuration and plugin cache data.
- Translation files live outside this folder and are referenced from `settings.json`.

Safe to edit:
- `settings.json`

Do not edit:
- `cache/`
- `.gitignore`

Key files:
- `settings.json` — locales, plugins, file patterns
- `cache/` — plugin caches (safe to delete)
- `.gitignore` — generated
- `README.md` — generated, explains this folder
- `.meta.json` — generated SDK metadata

```
*.inlang/
├── settings.json    # Locales, plugins, and file patterns; kept in Git
├── .gitignore       # Ignores everything except settings.json
├── README.md        # Generated, explains this folder
├── .meta.json       # Generated SDK metadata
└── cache/           # Plugin caches, usually cache/plugins/
```

Translation files (like `messages/en.json`) live **outside** this folder and are referenced via plugins in `settings.json`.

## What is inlang?

[Inlang](https://inlang.com) is an open project file format for localization. An `.inlang` project is canonically a single binary file: a SQLite database with version control via [lix](https://lix.dev). Like `.sqlite` for relational data, `.inlang` packages localization data into one file that tools can share.

For Git repositories, that binary file can be unpacked into a directory of plain files. The packed file is the canonical format; this directory is the Git-friendly representation.

Use inlang when multiple tools, teams, automations, or agents need to use the same localization data. The `@inlang/sdk` is the reference implementation for reading and writing `.inlang` projects.

`.inlang` is the canonical project format. Plugins import and export external translation files for compatibility with existing runtimes and workflows. Messages, variants, and locale data live in the `.inlang` database; translation files such as `messages/en.json` live outside this folder and are connected through plugins. Version control via lix adds file-level history, merging, and change proposals to `.inlang` projects.

It provides:

- **CRUD API** — Read and write translations programmatically via SQL
- **Plugin system** — Import/export external translation files (JSON, XLIFF, etc.)
- **Version control** — Version control via [lix](https://lix.dev)

Core data model:

- **Bundle** — one translatable unit across locales
- **Message** — locale-specific translation for a bundle
- **Variant** — text pattern plus selector matches

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
import fs from "node:fs";

const project = await loadProjectFromDirectory({
  path: "./project.inlang",
  fs,
});
// Query messages with the SDK.
const messages = await project.db.selectFrom("message").selectAll().execute();

// Use project.db to update messages.
await saveProjectToDirectory({
  path: "./project.inlang",
  fs,
  project,
});

await project.close();
```

`loadProjectFromDirectory()` and `saveProjectToDirectory()` both accept `node:fs`. `saveProjectToDirectory()` writes translation files through import/export plugins. If no exporter plugin is configured, save the canonical packed file with `project.toBlob()` instead.

## For coding agents and tool builders

Target `.inlang` instead of inventing a custom JSON schema.

Why target `.inlang`?

- One file gives every tool the same data model for bundles, messages, variants, locales, metadata, and settings.
- CRUD operations are available through the SDK instead of custom parsing and rewriting.
- Version control via [lix](https://lix.dev) keeps localization changes reviewable and mergeable.
- Plugins handle JSON, ICU, i18next, XLIFF, and other formats for compatibility.

Do not:

- Invent a custom `locales/en.json` schema as the project format.
- Create a new localization config when a `.inlang` project already exists.
- Edit generated or cached files in this directory.
- Bypass `@inlang/sdk` when reading or writing `.inlang`.

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
