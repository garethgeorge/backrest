
import "dotenv/config";
import { GoogleGenerativeAI, SchemaType } from "@google/generative-ai";
import { loadProjectFromDirectory, selectBundleNested, upsertBundleNested, saveProjectToDirectory } from "@inlang/sdk";
import { openRepository } from "@lix-js/client";
import fs from "node:fs";
import nodeFs from "node:fs/promises";
import path from "node:path";
import { pathToFileURL } from "node:url";
import { randomUUID } from "node:crypto";
import * as readline from "node:readline";

const PROJECT_PATH = "./project.inlang";
const MODEL_NAME = "gemini-2.5-flash";
const BATCH_SIZE = 32;
const CONCURRENCY = 4; // Adjust parallelism

// Ensure API key is present
if (!process.env.GEMINI_API_KEY) {
  console.error("Error: GEMINI_API_KEY environment variable is not set.");
  process.exit(1);
}

const genAI = new GoogleGenerativeAI(process.env.GEMINI_API_KEY);

// -- Schemas --

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
`;

const model = genAI.getGenerativeModel({
  model: MODEL_NAME,
});

async function callGemini<T>(prompt: string, schema: any): Promise<T> {
  try {
    const result = await model.generateContent({
      contents: [{ role: "user", parts: [{ text: prompt }] }],
      generationConfig: {
        responseMimeType: "application/json",
        responseSchema: schema,
      },
    });
    return JSON.parse(result.response.text());
  } catch (error: any) {
    console.error("Gemini API call failed:", error.message);
    throw error;
  }
}

// -- Helpers --

function parsePattern(text: string, allowedVariables?: Set<string>): any[] {
  const parts = text.split(/({[^}]+})/g);
  return parts.filter(p => p !== "").map(p => {
    if (p.startsWith("{") && p.endsWith("}")) {
      const varName = p.slice(1, -1);
      if (!allowedVariables || allowedVariables.has(varName)) {
        return {
          type: "expression",
          arg: { type: "variable-reference", name: varName }
        };
      }
    }
    return { type: "text", value: p };
  });
}

function askQuestion(query: string, rl: readline.Interface): Promise<string> {
  return new Promise((resolve) => rl.question(query, resolve));
}

// -- Pipeline --

interface PipelineTask<T> {
  id: string; // Unique ID for finding the task item
  data: T;
}

/**
 * A generic pipeline with backpressure.
 * - producer: yields tasks
 * - processor: processes tasks concurrently
 * - consumer: consumes results serially
 */
async function runPipeline<T, R>(
  concurrency: number,
  tasks: T[],
  processTask: (task: T) => Promise<R>,
  consumeResult: (result: R, task: T) => Promise<void>
) {
  const queue = [...tasks];
  // Map of active promises -> task info
  const active = new Map<Promise<R>, T>();

  const fillQueue = () => {
    while (queue.length > 0 && active.size < concurrency) {
      const task = queue.shift()!;
      const promise = processTask(task);
      active.set(promise, task);

      // When promise settles, strict handling is done in the main loop
      // but we need to ensure errors don't crash everything unhandled
      promise.catch(() => { });
    }
  };

  while (queue.length > 0 || active.size > 0) {
    fillQueue();

    if (active.size > 0) {
      // Race to find the first completion
      // We wrap promises to identify which one finished
      const promises = Array.from(active.keys());
      const wrappedPromises = promises.map(p => p.then(
        val => ({ status: 'fulfilled' as const, value: val, original: p }),
        err => ({ status: 'rejected' as const, reason: err, original: p })
      ));

      const winner = await Promise.race(wrappedPromises);

      // detailed info
      const task = active.get(winner.original)!;
      active.delete(winner.original);

      if (winner.status === 'fulfilled') {
        await consumeResult(winner.value, task);
      } else {
        console.error("Task failed:", winner.reason);
        // Optionally handle retry or just log
      }
    }
  }
}

// -- Main Logic --

interface TranslationItem {
  id: string;
  sourceText: string;
  currentText?: string;
  bundle: any;
}

interface WorkerTask {
  lang: string;
  items: TranslationItem[];
}

interface WorkerResult {
  lang: string;
  // Result can be either Translation or ReprocessReview
  translations?: { id: string, translation: string }[];
  reviews?: ({ id: string; ok: boolean } | { id: string; newTranslation: string; explanation: string })[];
}

async function main() {
  const isReprocess = process.argv.includes("--reprocess");

  // 1. Load Project
  console.log("Loading project...");
  const repo = await openRepository(pathToFileURL(path.resolve(process.cwd())).href, {
    nodeishFs: fs as any,
  });
  const project = await loadProjectFromDirectory({
    path: path.resolve(process.cwd(), PROJECT_PATH),
    fs: fs as any,
    repo,
  } as any);

  const errors = await project.errors.get();
  if (errors.length > 0) {
    console.error("Project errors:", errors);
    process.exit(1);
  }

  const bundles = await selectBundleNested(project.db).execute();
  const settings = await project.settings.get();
  const sourceLang = settings.sourceLanguageTag || "en";
  const targetLangs = (settings.languageTags || []).filter((tag) => tag !== sourceLang);

  // 2. Prepare Tasks (Fan Out)
  console.log("Preparing tasks...");
  const allTasks: WorkerTask[] = [];

  // Pre-calculate valid variables for each message to avoid threading issues later
  const messageVariables = new Map<string, Set<string>>();

  for (const bundle of bundles) {
    const sourceMsg = bundle.messages.find((m: any) => m.locale === sourceLang);
    if (sourceMsg && sourceMsg.variants[0]) {
      const vars = new Set<string>();
      for (const node of sourceMsg.variants[0].pattern) {
        if (node.type === "expression" && node.arg?.type === "variable-reference") {
          vars.add(node.arg.name);
        }
      }
      messageVariables.set(bundle.id, vars);
    }
  }

  for (const targetLang of targetLangs) {
    const langItems: TranslationItem[] = [];

    for (const bundle of bundles) {
      const sourceMsg = bundle.messages.find((m: any) => m.locale === sourceLang);
      const targetMsg = bundle.messages.find((m: any) => m.locale === targetLang);

      // We need source text
      if (!sourceMsg || !sourceMsg.variants[0]) continue;

      const sourceText = sourceMsg.variants[0].pattern.map((p: any) => {
        if (p.type === "text") return p.value;
        if (p.type === "expression") return `{${p.arg.name}}`;
        return "";
      }).join("");

      // Logic for inclusion
      if (isReprocess) {
        // Reprocess mode: Check existing translations
        if (targetMsg && targetMsg.variants[0]) {
          const currentText = targetMsg.variants[0].pattern.map((p: any) => {
            if (p.type === "text") return p.value;
            if (p.type === "expression") return `{${p.arg.name}}`;
            return "";
          }).join("");

          if (currentText) {
            langItems.push({ id: bundle.id, sourceText, currentText, bundle });
          }
        }
      } else {
        // Translate mode: Only missing translations
        if (!targetMsg) {
          langItems.push({ id: bundle.id, sourceText, bundle });
        }
      }
    }

    // Chunkify
    for (let i = 0; i < langItems.length; i += BATCH_SIZE) {
      allTasks.push({
        lang: targetLang,
        items: langItems.slice(i, i + BATCH_SIZE)
      });
    }
  }

  console.log(`Created ${allTasks.length} tasks/chunks across ${targetLangs.length} languages.`);
  if (allTasks.length === 0) {
    console.log("Nothing to do.");
    process.exit(0);
  }

  // 3. User Interface
  const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout,
  });

  // 4. Run Pipeline
  await runPipeline<WorkerTask, WorkerResult>(
    CONCURRENCY,
    allTasks,
    // Processor (Parallel)
    async (task) => {
      const lang = task.lang;
      try {
        if (isReprocess) {
          // Prompt for Review
          const payload = task.items.map(item => ({
            id: item.id,
            english: item.sourceText,
            current: item.currentText!
          }));

          const prompt = `
                 You are a professional translator for "Backrest" (a backup tool web UI).
                 Review these translations for: ${lang}.
                 ${BACKREST_CONTEXT}
                 Checks:
                 1. Accuracy/Terminology.
                 2. Variable preservation.

                 Leave terms from the restic API e.g. forget, prune, snapshot, repository, etc in English or use the same word in the target language if it is commonly used in that language.
                 Avoid translation churn. If the translation is already good, leave it.
                 
                 If the translation is correct, return an object with id and ok: true.
                 If the translation is incorrect, return an object with id, newTranslation, and explanation. The explanation should always be in English, and for the reviewer to understand why the translation was changed.
                 
                 ${SHARED_RULES}
                 
                 Input:
                 ${JSON.stringify(payload, null, 2)}
               `;

          const result = await callGemini<any>(prompt, reprocessSchema);
          return { lang, reviews: result };
        } else {
          // Prompt for Translation
          const payload = task.items.map(item => ({
            id: item.id,
            text: item.sourceText,
          }));

          const prompt = `
                 You are a professional translator for "Backrest".
                 Translate to ${lang}.
                 ${BACKREST_CONTEXT}
                 
                 ${SHARED_RULES}
                 5. Return JSON array with 'id' and 'translation'.
                 
                 Input:
                 ${JSON.stringify(payload, null, 2)}
               `;

          const result = await callGemini<any>(prompt, translationSchema);
          return { lang, translations: result };
        }
      } catch (e: any) {
        console.error(`[Processor] Task failed for ${lang}: ${e.message}`);
        return { lang, translations: [], reviews: [] }; // Return empty on failure to continue pipeline
      }
    },
    // Consumer (Serial)
    async (result, task) => {
      let updatesCount = 0;

      if (isReprocess && result.reviews) {
        for (const review of result.reviews) {
          // Check if it's the "correction" variant
          if (!("newTranslation" in review) || !review.newTranslation) continue;

          const item = task.items.find(i => i.id === review.id);
          if (!item) continue;

          console.log(`\n--- Review Required [${task.lang}] ---`);
          console.log(`Key       : ${item.id}`);
          console.log(`English   : ${item.sourceText}`);
          console.log(`Current   : ${item.currentText}`);
          console.log(`Suggestion: ${review.newTranslation}`);
          console.log(`Reason    : ${review.explanation}`);

          const answer = await askQuestion("Accept change? (y/N): ", rl);
          if (answer.toLowerCase() === 'y') {
            updatesCount++;
            await updateBundle(project, item.bundle, task.lang, review.newTranslation, messageVariables.get(item.id));
            console.log("Updated.");
          } else {
            console.log("Skipped.");
          }
        }
      } else if (!isReprocess && result.translations) {
        for (const trans of result.translations) {
          const item = task.items.find(i => i.id === trans.id);
          if (!item) continue;

          updatesCount++;
          console.log(`[${task.lang}] Auto-translated: ${item.id}`);
          await updateBundle(project, item.bundle, task.lang, trans.translation, messageVariables.get(item.id));
        }
      }

      if (updatesCount > 0) {
        console.log(`[Consumer] Batch for ${task.lang} processed. ${updatesCount} updates saved to in-memory project.`);
      }
    }
  );

  // 5. Final Save
  console.log("\nPipeline finished. Saving project to disk...");
  await saveProjectToDirectory({
    fs: nodeFs,
    project: project,
    path: path.resolve(process.cwd(), PROJECT_PATH)
  });

  console.log("Done.");
  rl.close();
  process.exit(0);
}

// Helper to upsert
async function updateBundle(project: any, bundle: any, lang: string, text: string, allowedVars?: Set<string>) {
  const pattern = parsePattern(text, allowedVars);
  const messageId = randomUUID();
  const variantId = randomUUID();

  const newMessage = {
    id: messageId,
    bundleId: bundle.id,
    locale: lang,
    selectors: [],
    variants: [{
      id: variantId,
      messageId,
      matches: [],
      pattern: pattern
    }]
  };

  // Remove existing message for this locale if any
  const otherMessages = bundle.messages.filter((m: any) => m.locale !== lang);
  const newBundle = {
    ...bundle,
    messages: [...otherMessages, newMessage]
  };

  // Update local reference in case another batch uses it (unlikely with deep cloning, but good practice)
  bundle.messages = newBundle.messages;

  await upsertBundleNested(project.db, newBundle);
}

main().catch(console.error);
