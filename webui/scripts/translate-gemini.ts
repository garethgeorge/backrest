import 'dotenv/config';
import { GoogleGenerativeAI, SchemaType } from '@google/generative-ai';
import {
  loadProjectFromDirectory,
  selectBundleNested,
  upsertBundleNested,
  saveProjectToDirectory,
} from '@inlang/sdk';
import { openRepository } from '@lix-js/client';
import fs from 'node:fs';
import nodeFs from 'node:fs/promises';
import path from 'node:path';
import { pathToFileURL } from 'node:url';
import { randomUUID } from 'node:crypto';
import * as readline from 'node:readline';

// ─── Config ──────────────────────────────────────────────────────────────────

const PROJECT_PATH = './project.inlang';
const MODEL_NAME = 'gemini-3.5-flash';
const BATCH_SIZE = 32;
// Max simultaneous Gemini requests, shared across all languages. Override with CONCURRENCY=N.
const CONCURRENCY = Number(process.env.CONCURRENCY) || 4;
const MAX_RETRIES = 5; // Per-batch retries on transient API failures.

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

/** One batch of items for a single language, tagged with its position for logging. */
interface LangBatch {
  lang: string;
  batch: TranslationItem[];
  batchIndex: number;
  batchCount: number;
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
    required: ['id', 'translation'],
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
        required: ['id', 'ok'],
      },
      {
        type: SchemaType.OBJECT,
        properties: {
          id: { type: SchemaType.STRING },
          newTranslation: { type: SchemaType.STRING },
          explanation: { type: SchemaType.STRING },
        },
        required: ['id', 'newTranslation', 'explanation'],
      },
    ],
  },
} as any;

// ─── Helpers ──────────────────────────────────────────────────────────────────

/** Split an array into fixed-size chunks. */
function chunk<T>(items: T[], size: number): T[][] {
  const out: T[][] = [];
  for (let i = 0; i < items.length; i += size) out.push(items.slice(i, i + size));
  return out;
}

/**
 * Build a flat, cross-language list of batches so the buffered iterator can keep
 * the API saturated across every target language at once (rather than draining
 * one language before starting the next). `getItems` selects the items per lang.
 */
function buildWork(
  langs: string[],
  getItems: (lang: string) => TranslationItem[],
  onEmpty: (lang: string) => void,
): LangBatch[] {
  const work: LangBatch[] = [];
  for (const lang of langs) {
    const items = getItems(lang);
    if (items.length === 0) {
      onEmpty(lang);
      continue;
    }
    const batches = chunk(items, BATCH_SIZE);
    batches.forEach((batch, batchIndex) =>
      work.push({ lang, batch, batchIndex, batchCount: batches.length }),
    );
  }
  return work;
}

/** Convert an inlang pattern array to a flat string (for display and prompts). */
function patternToText(pattern: any[]): string {
  return pattern
    .map((p) => {
      if (p.type === 'text') return p.value;
      if (p.type === 'expression') return `{${p.arg.name}}`;
      return '';
    })
    .join('');
}

/** Parse a flat string (possibly containing {var} placeholders) back to an inlang pattern. */
function parsePattern(text: string, allowedVariables?: Set<string>): any[] {
  return text
    .split(/({[^}]+})/g)
    .filter((p) => p !== '')
    .map((p) => {
      if (p.startsWith('{') && p.endsWith('}')) {
        const varName = p.slice(1, -1);
        if (!allowedVariables || allowedVariables.has(varName)) {
          return { type: 'expression', arg: { type: 'variable-reference', name: varName } };
        }
      }
      return { type: 'text', value: p };
    });
}

/**
 * A buffered async iterator: applies `fn` to each item in `source` with up to
 * `concurrency` in-flight calls at once, yielding results in source order.
 *
 * Crucially, launching is driven by task *completions* (via the `.finally`
 * callback below), not by the consumer pulling the next value. That means the
 * pipeline keeps `concurrency` requests in flight and accumulates an unbounded
 * queue of finished results even while the consumer is parked — e.g. blocked on
 * `readline` waiting for the human to approve a review. Leave it for an hour and
 * everything will have been fetched and queued, ready to approve.
 */
async function* createBufferedIterator<T, R>(
  source: T[],
  concurrency: number,
  fn: (item: T, index: number) => Promise<R>,
): AsyncGenerator<{ item: T; result: R; index: number }> {
  // Finished results waiting to be yielded, keyed by index (the queue).
  const resolved = new Map<number, R>();
  const inflight = new Set<number>();
  let nextToLaunch = 0;
  let nextToYield = 0;

  // A one-shot signal used to wake the consumer when it is waiting for the next
  // in-order result. Refreshed each time it fires.
  let wake: (() => void) | null = null;
  const signalWake = () => {
    const w = wake;
    wake = null;
    w?.();
  };

  // Keep `concurrency` requests in flight. Re-invoked from each task's
  // completion callback, so it runs in the background regardless of how slowly
  // (or whether) the consumer is pulling results.
  const pump = () => {
    while (nextToLaunch < source.length && inflight.size < concurrency) {
      const i = nextToLaunch++;
      inflight.add(i);
      Promise.resolve(fn(source[i], i))
        .then((result) => resolved.set(i, result))
        .finally(() => {
          inflight.delete(i);
          pump();
          signalWake();
        });
    }
  };

  pump();

  while (nextToYield < source.length) {
    if (!resolved.has(nextToYield)) {
      await new Promise<void>((r) => (wake = r));
      continue;
    }
    const result = resolved.get(nextToYield)!;
    resolved.delete(nextToYield);
    yield { item: source[nextToYield], result, index: nextToYield };
    nextToYield++;
  }
}

// ─── GeminiClient ─────────────────────────────────────────────────────────────

class GeminiClient {
  private model: ReturnType<GoogleGenerativeAI['getGenerativeModel']>;

  constructor(apiKey: string) {
    const genAI = new GoogleGenerativeAI(apiKey);
    this.model = genAI.getGenerativeModel({ model: MODEL_NAME });
  }

  private async call<T>(prompt: string, schema: any): Promise<T> {
    let lastErr: any;
    for (let attempt = 0; attempt <= MAX_RETRIES; attempt++) {
      try {
        const result = await this.model.generateContent({
          contents: [{ role: 'user', parts: [{ text: prompt }] }],
          generationConfig: {
            responseMimeType: 'application/json',
            responseSchema: schema,
          },
        });
        return JSON.parse(result.response.text()) as T;
      } catch (err: any) {
        lastErr = err;
        if (attempt === MAX_RETRIES) break;
        // Exponential backoff with jitter to ride out 429s / transient 5xx.
        const delay = Math.min(30_000, 1_000 * 2 ** attempt) * (0.5 + Math.random());
        await new Promise((r) => setTimeout(r, delay));
      }
    }
    throw lastErr;
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

class TranslationProject {
  private project: any | null = null;
  private _bundles: any[] = [];
  private _sourceLang = 'en';
  private _targetLangs: string[] = [];
  private _bundleVars = new Map<string, Set<string>>();
  private _dirty = false;
  // Serializes DB writes (updateBundle) against disk saves so a timer-driven
  // save never reads the DB mid-write.
  private _lock: Promise<void> = Promise.resolve();

  private withLock<T>(fn: () => Promise<T>): Promise<T> {
    const run = this._lock.then(fn, fn);
    this._lock = run.then(
      () => undefined,
      () => undefined,
    );
    return run;
  }

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
      console.error('Project errors:', errors);
      process.exit(1);
    }

    const settings = await this.project.settings.get();
    this._sourceLang = settings.sourceLanguageTag || 'en';
    this._targetLangs = (settings.languageTags || []).filter(
      (tag: string) => tag !== this._sourceLang,
    );
    this._bundles = await selectBundleNested(this.project.db).execute();

    // Precompute variable sets for each bundle from the source message
    for (const bundle of this._bundles) {
      const sourceMsg = bundle.messages.find((m: any) => m.locale === this._sourceLang);
      if (sourceMsg?.variants[0]) {
        const vars = new Set<string>();
        for (const node of sourceMsg.variants[0].pattern) {
          if (node.type === 'expression' && node.arg?.type === 'variable-reference') {
            vars.add(node.arg.name);
          }
        }
        this._bundleVars.set(bundle.id, vars);
      }
    }
  }

  get sourceLang(): string {
    return this._sourceLang;
  }
  get targetLangs(): string[] {
    return this._targetLangs;
  }

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

    await this.withLock(() => upsertBundleNested(this.project!.db, newBundle));
    this._dirty = true;
  }

  async save(projectPath: string): Promise<void> {
    await this.withLock(() =>
      saveProjectToDirectory({
        fs: nodeFs,
        project: this.project!,
        path: path.resolve(process.cwd(), projectPath),
      }),
    );
  }

  /** Saves to disk only if there are unsaved edits. Returns true if it saved. */
  async saveIfDirty(projectPath: string): Promise<boolean> {
    if (!this._dirty) return false;
    this._dirty = false;
    try {
      await this.save(projectPath);
      return true;
    } catch (err) {
      this._dirty = true; // leave dirty so the next tick retries
      throw err;
    }
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
    explanation: string,
  ): Promise<boolean> {
    console.log(`\n┌─ Review Required [${lang}] ${'─'.repeat(40 - lang.length)}`);
    console.log(`│  Key       : ${item.id}`);
    console.log(`│  English   : ${item.sourceText}`);
    console.log(`│  Current   : ${item.currentText}`);
    console.log(`│  Suggestion: ${suggestion}`);
    console.log(`│  Reason    : ${explanation}`);
    console.log(`└${'─'.repeat(48)}`);

    const answer = await this.ask('  Accept change? (y/N): ');
    return answer.trim().toLowerCase() === 'y';
  }

  close(): void {
    this.rl.close();
  }
}

// ─── Main ─────────────────────────────────────────────────────────────────────

async function runTranslate(gemini: GeminiClient, project: TranslationProject): Promise<void> {
  let totalUpdates = 0;

  const work = buildWork(
    project.targetLangs,
    (lang) => project.getMissingItems(lang),
    (lang) => console.error(`[${lang}] Nothing to translate. Skipping.`),
  );
  if (work.length === 0) {
    console.log('Nothing to translate.');
    return;
  }
  console.error(
    `${work.length} batches to translate across ${project.targetLangs.length} languages.`,
  );

  // One buffered iterator over every (lang, batch) pair keeps the API saturated globally.
  for await (const { item: w, result } of createBufferedIterator(work, CONCURRENCY, async (w) => {
    process.stderr.write(`[${w.lang}] Fetching batch ${w.batchIndex + 1}/${w.batchCount}...\n`);
    try {
      return await gemini.translateBatch(w.lang, w.batch);
    } catch (err: any) {
      console.error(
        `\n[${w.lang}] ERROR: batch ${w.batchIndex + 1}/${w.batchCount} failed after retries: ${err.message}`,
      );
      return [] as TranslationResult[];
    }
  })) {
    for (const trans of result) {
      const item = w.batch.find((i) => i.id === trans.id);
      if (!item) {
        console.error(`[${w.lang}] Warning: Gemini returned unknown id '${trans.id}', skipping.`);
        continue;
      }
      await project.updateBundle(item.bundle, w.lang, trans.translation);
      console.log(`[${w.lang}] ✓ ${item.id}`);
      totalUpdates++;
    }
  }

  console.log(`\nTranslation complete. ${totalUpdates} strings translated.`);
}

async function runReprocess(
  gemini: GeminiClient,
  project: TranslationProject,
  reviewer: UserReviewer,
): Promise<void> {
  let totalAccepted = 0;
  let totalSkipped = 0;

  const work = buildWork(
    project.targetLangs,
    (lang) => project.getExistingItems(lang),
    (lang) => console.error(`[${lang}] No existing translations found. Skipping.`),
  );
  if (work.length === 0) {
    console.log('No existing translations to review.');
    return;
  }
  console.error(`Reviewing ${work.length} batches across ${project.targetLangs.length} languages.`);

  // One global buffered iterator keeps CONCURRENCY reviews in flight across all
  // languages while the user serially approves each completed batch — so the
  // queue stays full no matter how long the user takes.
  for await (const { item: w, result } of createBufferedIterator(work, CONCURRENCY, async (w) => {
    process.stderr.write(
      `[${w.lang}] Fetching review for batch ${w.batchIndex + 1}/${w.batchCount}...\n`,
    );
    try {
      return await gemini.reviewBatch(w.lang, w.batch);
    } catch (err: any) {
      console.error(
        `\n[${w.lang}] ERROR: batch ${w.batchIndex + 1}/${w.batchCount} failed after retries: ${err.message}`,
      );
      return [] as ReviewResult[];
    }
  })) {
    const suggestions = result.filter(
      (r): r is { id: string; newTranslation: string; explanation: string } =>
        'newTranslation' in r && !!r.newTranslation,
    );

    if (suggestions.length === 0) {
      process.stderr.write(
        `[${w.lang}] Batch ${w.batchIndex + 1}/${w.batchCount}: all translations OK.\n`,
      );
      continue;
    }

    process.stderr.write(
      `[${w.lang}] Batch ${w.batchIndex + 1}/${w.batchCount}: ${suggestions.length} suggestion(s) to review.\n`,
    );

    for (const review of suggestions) {
      const item = w.batch.find((i) => i.id === review.id);
      if (!item) {
        console.error(`[${w.lang}] Warning: Gemini returned unknown id '${review.id}', skipping.`);
        continue;
      }

      const accepted = await reviewer.promptReview(
        w.lang,
        item,
        review.newTranslation,
        review.explanation,
      );
      if (accepted) {
        await project.updateBundle(item.bundle, w.lang, review.newTranslation);
        console.log('  → Updated.');
        totalAccepted++;
      } else {
        console.log('  → Skipped.');
        totalSkipped++;
      }
    }
  }

  console.log(`\nReview complete. ${totalAccepted} accepted, ${totalSkipped} skipped.`);
}

async function main() {
  if (!process.env.GEMINI_API_KEY) {
    console.error('Error: GEMINI_API_KEY environment variable is not set.');
    process.exit(1);
  }

  const isReprocess = process.argv.includes('--reprocess');
  const mode = isReprocess ? 'reprocess' : 'translate';

  const gemini = new GeminiClient(process.env.GEMINI_API_KEY);
  const project = new TranslationProject();

  console.error(`Mode: ${mode}`);
  console.error('Loading project...');
  await project.load(PROJECT_PATH);

  const targetLangs = project.targetLangs;
  if (targetLangs.length === 0) {
    console.log('No target languages configured. Nothing to do.');
    process.exit(0);
  }

  console.error(`Source: ${project.sourceLang} → Targets: ${targetLangs.join(', ')}`);

  // Periodically flush new edits to disk so an interrupted long run isn't lost.
  const autosave = setInterval(() => {
    project
      .saveIfDirty(PROJECT_PATH)
      .then((saved) => saved && console.error('[autosave] saved.'))
      .catch((err) => console.error(`[autosave] failed: ${err.message}`));
  }, 60_000);
  autosave.unref();

  try {
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
  } finally {
    clearInterval(autosave);
  }

  console.error('Saving project to disk...');
  await project.save(PROJECT_PATH);
  console.error('Done.');
  process.exit(0);
}

main().catch((err) => {
  console.error('Fatal error:', err);
  process.exit(1);
});
