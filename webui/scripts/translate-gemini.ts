import "dotenv/config";
import { GoogleGenerativeAI, SchemaType } from "@google/generative-ai";
import {
  loadProjectFromDirectory,
  selectBundleNested,
  upsertBundleNested,
  saveProjectToDirectory,
} from "@inlang/sdk";
import { openRepository } from "@lix-js/client";
import fs from "node:fs";
import nodeFs from "node:fs/promises";
import path from "node:path";
import { pathToFileURL } from "node:url";
import { randomUUID } from "node:crypto";
import * as readline from "node:readline";

// ─── Config ──────────────────────────────────────────────────────────────────

const PROJECT_PATH = "./project.inlang";
const MODEL_NAME = "gemini-2.5-flash";
const BATCH_SIZE = 32;
const CONCURRENCY = 4; // Max simultaneous Gemini requests per language

const BACKREST_CONTEXT = `
  Context about Backrest:
  - It is a GUI and scheduler for the "restic" backup tool.
  - Key operations:
    - "Backup": Creates snapshots of data.
    - "Prune": Removes unused data from the repository (repacks).
    - "Forget": Manages snapshot retention policies (e.g., keep last N).
    - "Check": Verifies repository integrity.
    - "Snapshot": A backup point in time.
  - Scheduling:
    - Repo: a restic repository, includes its passwords, environment variables and flags.
    - Plan: a scheduled backup job, includes its schedule, repository, and any extra flags.
  - Features: Cron scheduling, multi-platform (Linux/macOS/Windows), supports various storage backends (S3, B2, Local, SFTP).
`;

const SHARED_RULES = `
  Rules:
  1. Maintain all variables (e.g. {name}) exactly.
  2. Do not add explanations to the translation text.
  3. Use terminology consistent with backup software (e.g., "snapshot", "repository", "retention").
  4. Variables in text are enclosed in braces {}, copy them exactly. Do not add new escape characters (but keep any existing ones).
  Leave terms from the restic API (e.g. forget, prune, snapshot, repository) in English or use the same word in the target language if it is commonly used there.
`;

// ─── Types ────────────────────────────────────────────────────────────────────

interface TranslationItem {
  id: string;
  sourceText: string;
  currentText?: string; // only set in reprocess mode
  bundle: any;
}

type TranslationResult = { id: string; translation: string };

type ReviewResult =
  | { id: string; ok: true }
  | { id: string; ok?: false; newTranslation: string; explanation: string };

// ─── Schemas ──────────────────────────────────────────────────────────────────

const translationSchema = {
  type: SchemaType.ARRAY,
  items: {
    type: SchemaType.OBJECT,
    properties: {
      id: { type: SchemaType.STRING },
      translation: { type: SchemaType.STRING },
    },
    required: ["id", "translation"],
  },
} as any;

const reprocessSchema = {
  type: SchemaType.ARRAY,
  items: {
    oneOf: [
      {
        type: SchemaType.OBJECT,
        properties: {
          id: { type: SchemaType.STRING },
          ok: { type: SchemaType.BOOLEAN },
        },
        required: ["id", "ok"],
      },
      {
        type: SchemaType.OBJECT,
        properties: {
          id: { type: SchemaType.STRING },
          newTranslation: { type: SchemaType.STRING },
          explanation: { type: SchemaType.STRING },
        },
        required: ["id", "newTranslation", "explanation"],
      },
    ],
  },
} as any;

// ─── Helpers ──────────────────────────────────────────────────────────────────

/** Convert an inlang pattern array to a flat string (for display and prompts). */
function patternToText(pattern: any[]): string {
  return pattern
    .map((p) => {
      if (p.type === "text") return p.value;
      if (p.type === "expression") return `{${p.arg.name}}`;
      return "";
    })
    .join("");
}

/** Parse a flat string (possibly containing {var} placeholders) back to an inlang pattern. */
function parsePattern(text: string, allowedVariables?: Set<string>): any[] {
  return text
    .split(/({[^}]+})/g)
    .filter((p) => p !== "")
    .map((p) => {
      if (p.startsWith("{") && p.endsWith("}")) {
        const varName = p.slice(1, -1);
        if (!allowedVariables || allowedVariables.has(varName)) {
          return { type: "expression", arg: { type: "variable-reference", name: varName } };
        }
      }
      return { type: "text", value: p };
    });
}

/**
 * A buffered async iterator: applies `fn` to each item in `source` with up to
 * `concurrency` in-flight calls at once, yielding results in source order.
 *
 * This is the core primitive enabling "fetch ahead while user reviews".
 */
async function* createBufferedIterator<T, R>(
  source: T[],
  concurrency: number,
  fn: (item: T, index: number) => Promise<R>
): AsyncGenerator<{ item: T; result: R; index: number }> {
  // Resolved results waiting to be yielded, keyed by index
  const resolved = new Map<number, R>();
  // Active promises, keyed by index
  const inflight = new Map<number, Promise<void>>();
  let nextToLaunch = 0;
  let nextToYield = 0;

  const launch = (i: number) => {
    const item = source[i];
    const p = fn(item, i).then((result) => {
      resolved.set(i, result);
      inflight.delete(i);
    });
    inflight.set(i, p);
  };

  while (nextToYield < source.length) {
    // Fill up to concurrency
    while (nextToLaunch < source.length && inflight.size < concurrency) {
      launch(nextToLaunch++);
    }

    // Wait until the next result in order is ready
    if (!resolved.has(nextToYield)) {
      // Wait for any inflight to finish, then check again
      await Promise.race(inflight.values());
    }

    if (resolved.has(nextToYield)) {
      const item = source[nextToYield];
      const result = resolved.get(nextToYield)!;
      resolved.delete(nextToYield);
      yield { item, result, index: nextToYield };
      nextToYield++;
    }
  }
}

// ─── GeminiClient ─────────────────────────────────────────────────────────────

class GeminiClient {
  private model: ReturnType<GoogleGenerativeAI["getGenerativeModel"]>;

  constructor(apiKey: string) {
    const genAI = new GoogleGenerativeAI(apiKey);
    this.model = genAI.getGenerativeModel({ model: MODEL_NAME });
  }

  private async call<T>(prompt: string, schema: any): Promise<T> {
    const result = await this.model.generateContent({
      contents: [{ role: "user", parts: [{ text: prompt }] }],
      generationConfig: {
        responseMimeType: "application/json",
        responseSchema: schema,
      },
    });
    return JSON.parse(result.response.text()) as T;
  }

  private translatePrompt(lang: string, items: TranslationItem[]): string {
    const payload = items.map((item) => ({ id: item.id, text: item.sourceText }));
    return `
      You are a professional translator for "Backrest".
      Translate the following strings to ${lang}.
      ${BACKREST_CONTEXT}
      ${SHARED_RULES}
      5. Return a JSON array with 'id' and 'translation' fields.

      Input:
      ${JSON.stringify(payload, null, 2)}
    `;
  }

  private reviewPrompt(lang: string, items: TranslationItem[]): string {
    const payload = items.map((item) => ({
      id: item.id,
      english: item.sourceText,
      current: item.currentText!,
    }));
    return `
      You are a professional translator for "Backrest" (a backup tool web UI).
      Review these ${lang} translations for accuracy and consistency.
      ${BACKREST_CONTEXT}
      Checks:
      1. Accuracy / terminology.
      2. Variable preservation.
      Avoid translation churn: if a translation is already good, leave it.
      ${SHARED_RULES}

      If a translation is correct, return: { "id": "...", "ok": true }
      If it needs changing, return: { "id": "...", "newTranslation": "...", "explanation": "..." }
      The explanation must be in English and explain the reasoning for the reviewer.

      Input:
      ${JSON.stringify(payload, null, 2)}
    `;
  }

  async translateBatch(lang: string, items: TranslationItem[]): Promise<TranslationResult[]> {
    return this.call<TranslationResult[]>(this.translatePrompt(lang, items), translationSchema);
  }

  async reviewBatch(lang: string, items: TranslationItem[]): Promise<ReviewResult[]> {
    return this.call<ReviewResult[]>(this.reviewPrompt(lang, items), reprocessSchema);
  }
}

// ─── TranslationProject ───────────────────────────────────────────────────────

interface BundleInfo {
  bundle: any;
  /** Variable names used in the English source. */
  allowedVars: Set<string>;
}

class TranslationProject {
  private project: any | null = null;
  private _bundles: any[] = [];
  private _sourceLang = "en";
  private _targetLangs: string[] = [];
  private _bundleVars = new Map<string, Set<string>>();

  async load(projectPath: string): Promise<void> {
    const repo = await openRepository(pathToFileURL(path.resolve(process.cwd())).href, {
      nodeishFs: fs as any,
    });
    this.project = await loadProjectFromDirectory({
      path: path.resolve(process.cwd(), projectPath),
      fs: fs as any,
      repo,
    } as any);

    const errors = await this.project.errors.get();
    if (errors.length > 0) {
      console.error("Project errors:", errors);
      process.exit(1);
    }

    const settings = await this.project.settings.get();
    this._sourceLang = settings.sourceLanguageTag || "en";
    this._targetLangs = (settings.languageTags || []).filter(
      (tag: string) => tag !== this._sourceLang
    );
    this._bundles = await selectBundleNested(this.project.db).execute();

    // Precompute variable sets for each bundle from the source message
    for (const bundle of this._bundles) {
      const sourceMsg = bundle.messages.find((m: any) => m.locale === this._sourceLang);
      if (sourceMsg?.variants[0]) {
        const vars = new Set<string>();
        for (const node of sourceMsg.variants[0].pattern) {
          if (node.type === "expression" && node.arg?.type === "variable-reference") {
            vars.add(node.arg.name);
          }
        }
        this._bundleVars.set(bundle.id, vars);
      }
    }
  }

  get sourceLang(): string { return this._sourceLang; }
  get targetLangs(): string[] { return this._targetLangs; }

  /**
   * Returns items that need translation (no target message exists yet).
   */
  getMissingItems(targetLang: string): TranslationItem[] {
    const items: TranslationItem[] = [];
    for (const bundle of this._bundles) {
      const sourceMsg = bundle.messages.find((m: any) => m.locale === this._sourceLang);
      if (!sourceMsg?.variants[0]) continue;
      const targetMsg = bundle.messages.find((m: any) => m.locale === targetLang);
      if (targetMsg) continue;

      items.push({
        id: bundle.id,
        sourceText: patternToText(sourceMsg.variants[0].pattern),
        bundle,
      });
    }
    return items;
  }

  /**
   * Returns items that have an existing translation (for review/reprocess mode).
   */
  getExistingItems(targetLang: string): TranslationItem[] {
    const items: TranslationItem[] = [];
    for (const bundle of this._bundles) {
      const sourceMsg = bundle.messages.find((m: any) => m.locale === this._sourceLang);
      if (!sourceMsg?.variants[0]) continue;
      const targetMsg = bundle.messages.find((m: any) => m.locale === targetLang);
      if (!targetMsg?.variants[0]) continue;

      const currentText = patternToText(targetMsg.variants[0].pattern);
      if (!currentText) continue;

      items.push({
        id: bundle.id,
        sourceText: patternToText(sourceMsg.variants[0].pattern),
        currentText,
        bundle,
      });
    }
    return items;
  }

  allowedVars(bundleId: string): Set<string> | undefined {
    return this._bundleVars.get(bundleId);
  }

  async updateBundle(bundle: any, lang: string, text: string): Promise<void> {
    const allowedVars = this._bundleVars.get(bundle.id);
    const pattern = parsePattern(text, allowedVars);
    const messageId = randomUUID();
    const variantId = randomUUID();

    const newMessage = {
      id: messageId,
      bundleId: bundle.id,
      locale: lang,
      selectors: [],
      variants: [{ id: variantId, messageId, matches: [], pattern }],
    };

    const newBundle = {
      ...bundle,
      messages: [...bundle.messages.filter((m: any) => m.locale !== lang), newMessage],
    };

    // Mutate the local reference so subsequent passes see the updated message
    bundle.messages = newBundle.messages;

    await upsertBundleNested(this.project!.db, newBundle);
  }

  async save(projectPath: string): Promise<void> {
    await saveProjectToDirectory({
      fs: nodeFs,
      project: this.project!,
      path: path.resolve(process.cwd(), projectPath),
    });
  }
}

// ─── UserReviewer ─────────────────────────────────────────────────────────────

class UserReviewer {
  private rl: readline.Interface;

  constructor() {
    this.rl = readline.createInterface({ input: process.stdin, output: process.stdout });
  }

  private ask(query: string): Promise<string> {
    return new Promise((resolve) => this.rl.question(query, resolve));
  }

  async promptReview(
    lang: string,
    item: TranslationItem,
    suggestion: string,
    explanation: string
  ): Promise<boolean> {
    console.log(`\n┌─ Review Required [${lang}] ${"─".repeat(40 - lang.length)}`);
    console.log(`│  Key       : ${item.id}`);
    console.log(`│  English   : ${item.sourceText}`);
    console.log(`│  Current   : ${item.currentText}`);
    console.log(`│  Suggestion: ${suggestion}`);
    console.log(`│  Reason    : ${explanation}`);
    console.log(`└${"─".repeat(48)}`);

    const answer = await this.ask("  Accept change? (y/N): ");
    return answer.trim().toLowerCase() === "y";
  }

  close(): void {
    this.rl.close();
  }
}

// ─── Main ─────────────────────────────────────────────────────────────────────

async function runTranslate(
  gemini: GeminiClient,
  project: TranslationProject
): Promise<void> {
  let totalUpdates = 0;

  for (const lang of project.targetLangs) {
    const allItems = project.getMissingItems(lang);
    if (allItems.length === 0) {
      console.error(`[${lang}] Nothing to translate. Skipping.`);
      continue;
    }

    // Chunk into batches
    const batches: TranslationItem[][] = [];
    for (let i = 0; i < allItems.length; i += BATCH_SIZE) {
      batches.push(allItems.slice(i, i + BATCH_SIZE));
    }

    console.error(`[${lang}] ${allItems.length} strings to translate across ${batches.length} batches.`);
    let batchNum = 0;

    // Process batches with bounded concurrency, yielding results in order
    for await (const { item: batch, result, index } of createBufferedIterator(
      batches,
      CONCURRENCY,
      async (batch, batchIndex) => {
        process.stderr.write(`[${lang}] Fetching batch ${batchIndex + 1}/${batches.length}...\n`);
        try {
          return await gemini.translateBatch(lang, batch);
        } catch (err: any) {
          console.error(`\n[${lang}] ERROR: Gemini call failed for batch ${batchIndex + 1}/${batches.length}: ${err.message}`);
          return [] as TranslationResult[];
        }
      }
    )) {
      batchNum++;
      for (const trans of result) {
        const item = batch.find((i) => i.id === trans.id);
        if (!item) {
          console.error(`[${lang}] Warning: Gemini returned unknown id '${trans.id}', skipping.`);
          continue;
        }
        await project.updateBundle(item.bundle, lang, trans.translation);
        console.log(`[${lang}] ✓ ${item.id}`);
        totalUpdates++;
      }
      process.stderr.write(`[${lang}] Batch ${batchNum}/${batches.length} applied.\n`);
    }
  }

  console.log(`\nTranslation complete. ${totalUpdates} strings translated.`);
}

async function runReprocess(
  gemini: GeminiClient,
  project: TranslationProject,
  reviewer: UserReviewer
): Promise<void> {
  let totalAccepted = 0;
  let totalSkipped = 0;

  for (const lang of project.targetLangs) {
    const allItems = project.getExistingItems(lang);
    if (allItems.length === 0) {
      console.error(`[${lang}] No existing translations found. Skipping.`);
      continue;
    }

    const batches: TranslationItem[][] = [];
    for (let i = 0; i < allItems.length; i += BATCH_SIZE) {
      batches.push(allItems.slice(i, i + BATCH_SIZE));
    }

    console.error(`\n[${lang}] Reviewing ${allItems.length} strings across ${batches.length} batches.`);

    // The buffered iterator keeps CONCURRENCY Gemini calls in flight while the
    // user serially reviews each completed batch — true pipelining.
    for await (const { item: batch, result, index } of createBufferedIterator(
      batches,
      CONCURRENCY,
      async (batch, batchIndex) => {
        process.stderr.write(`[${lang}] Fetching review for batch ${batchIndex + 1}/${batches.length}...\n`);
        try {
          return await gemini.reviewBatch(lang, batch);
        } catch (err: any) {
          console.error(`\n[${lang}] ERROR: Gemini call failed for batch ${batchIndex + 1}/${batches.length}: ${err.message}`);
          return [] as ReviewResult[];
        }
      }
    )) {
      const suggestions = result.filter(
        (r): r is { id: string; newTranslation: string; explanation: string } =>
          "newTranslation" in r && !!r.newTranslation
      );

      if (suggestions.length === 0) {
        process.stderr.write(`[${lang}] Batch ${index + 1}/${batches.length}: all translations OK.\n`);
        continue;
      }

      process.stderr.write(
        `[${lang}] Batch ${index + 1}/${batches.length}: ${suggestions.length} suggestion(s) to review.\n`
      );

      for (const review of suggestions) {
        const item = batch.find((i) => i.id === review.id);
        if (!item) {
          console.error(`[${lang}] Warning: Gemini returned unknown id '${review.id}', skipping.`);
          continue;
        }

        const accepted = await reviewer.promptReview(lang, item, review.newTranslation, review.explanation);
        if (accepted) {
          await project.updateBundle(item.bundle, lang, review.newTranslation);
          console.log("  → Updated.");
          totalAccepted++;
        } else {
          console.log("  → Skipped.");
          totalSkipped++;
        }
      }
    }
  }

  console.log(`\nReview complete. ${totalAccepted} accepted, ${totalSkipped} skipped.`);
}

async function main() {
  if (!process.env.GEMINI_API_KEY) {
    console.error("Error: GEMINI_API_KEY environment variable is not set.");
    process.exit(1);
  }

  const isReprocess = process.argv.includes("--reprocess");
  const mode = isReprocess ? "reprocess" : "translate";

  const gemini = new GeminiClient(process.env.GEMINI_API_KEY);
  const project = new TranslationProject();

  console.error(`Mode: ${mode}`);
  console.error("Loading project...");
  await project.load(PROJECT_PATH);

  const targetLangs = project.targetLangs;
  if (targetLangs.length === 0) {
    console.log("No target languages configured. Nothing to do.");
    process.exit(0);
  }

  console.error(`Source: ${project.sourceLang} → Targets: ${targetLangs.join(", ")}`);

  if (isReprocess) {
    const reviewer = new UserReviewer();
    try {
      await runReprocess(gemini, project, reviewer);
    } finally {
      reviewer.close();
    }
  } else {
    await runTranslate(gemini, project);
  }

  console.error("Saving project to disk...");
  await project.save(PROJECT_PATH);
  console.error("Done.");
  process.exit(0);
}

main().catch((err) => {
  console.error("Fatal error:", err);
  process.exit(1);
});
