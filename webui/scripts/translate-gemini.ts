
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
const MODEL_NAME = "gemini-2.5-flash"; // Updated per previous edit
const BATCH_SIZE = 64;

// Ensure API key is present
if (!process.env.GEMINI_API_KEY) {
  console.error("Error: GEMINI_API_KEY environment variable is not set.");
  process.exit(1);
}

const genAI = new GoogleGenerativeAI(process.env.GEMINI_API_KEY);

// Schema for simple translation
const translationSchema = {
  type: SchemaType.ARRAY,
  items: {
    type: SchemaType.OBJECT,
    properties: {
      id: { type: SchemaType.STRING },
      translation: { type: SchemaType.STRING },
    },
  },
} as any;

// Schema for reprocessing/review
const reprocessSchema = {
  type: SchemaType.ARRAY,
  items: {
    type: SchemaType.OBJECT,
    properties: {
      id: { type: SchemaType.STRING },
      ok: { type: SchemaType.BOOLEAN },
      newTranslation: { type: SchemaType.STRING, nullable: true },
      explanation: { type: SchemaType.STRING, nullable: true },
    },
    required: ["id", "ok"],
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
  2. Do not add explanations.
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

function parsePattern(text: string, allowedVariables?: Set<string>): any[] {
  // Split by variable pattern {varName}
  const parts = text.split(/({[^}]+})/g);
  return parts.filter(p => p !== "").map(p => {
    // If it looks like a variable {name}
    if (p.startsWith("{") && p.endsWith("}")) {
      const varName = p.slice(1, -1);
      // Only treat as expression if it's in the allowed list (or if no list provided, for safety/fallback)
      // Ideally we always provide the list.
      if (!allowedVariables || allowedVariables.has(varName)) {
        return {
          type: "expression",
          arg: { type: "variable-reference", name: varName }
        };
      }
    }
    // Otherwise it's plain text
    return { type: "text", value: p };
  });
}

async function translateBatch(
  batch: { id: string; text: string }[],
  targetLang: string
): Promise<{ id: string; translation: string }[]> {
  const payload = batch.map((msg) => ({
    id: msg.id,
    text: msg.text,
  }));

  const prompt = `
    You are a professional translator for "Backrest", a web-accessible backup solution built on top of restic.
    
    ${BACKREST_CONTEXT}
    
    Translate the following texts to ${targetLang}.
    
    ${SHARED_RULES}
    5. Return a JSON array where each object contains the 'id' (from input) and the 'translation'.
    
    Input JSON:
    ${JSON.stringify(payload, null, 2)}
  `;

  try {
      return await callGemini(prompt, translationSchema);
  } catch (e) {
      return [];
  }
}

async function reprocessBatch(
  batch: { id: string; english: string; current: string }[],
  targetLang: string
): Promise<{ id: string; ok: boolean; newTranslation?: string; explanation?: string }[]> {
  const payload = batch.map((msg) => ({
    id: msg.id,
    english: msg.english,
    current: msg.current,
  }));

  const prompt = `
    You are a professional translator for "Backrest" (a backup tool web UI).
    Review the following translations for target language: ${targetLang}.

    ${BACKREST_CONTEXT}

    For each item:
    1. accurate: Is the 'current' translation accurate and does it use correct terminology? (bool "ok")
    2. fit: Does it preserve variables (e.g. {name})?
    
    If "ok" is false:
    - Provide a better "newTranslation".
    - Provide a brief "explanation" (in English) of why the old one was incorrect or suboptimal.
    
    ${SHARED_RULES}

    Input JSON:
    ${JSON.stringify(payload, null, 2)}
  `;

  try {
      return await callGemini(prompt, reprocessSchema);
  } catch (e) {
      return [];
  }
}

function askQuestion(query: string, rl: readline.Interface): Promise<string> {
  return new Promise((resolve) => rl.question(query, resolve));
}

async function* createBufferedIterator<T, R>(
  items: T[],
  batchSize: number,
  processor: (batch: T[]) => Promise<R>,
  concurrency: number = 2
) {
  // Split items into batches
  const batches: T[][] = [];
  for (let i = 0; i < items.length; i += batchSize) {
     batches.push(items.slice(i, i + batchSize));
  }

  const batchPromises: Promise<R>[] = [];

  for (let i = 0; i < batches.length; i++) {
      // Ensure we have a promise for this batch
      if (!batchPromises[i]) {
          console.log(`[Background] Starting batch ${i + 1}/${batches.length}...`);
          batchPromises[i] = processor(batches[i]);
      }
      
      // Ensure we have prefetched up to concurrency limit
      for (let j = 1; j < concurrency + 1; j++) {
           if (i + j < batches.length && !batchPromises[i + j]) {
               console.log(`[Background] Pre-fetching batch ${i + j + 1}/${batches.length}...`);
               batchPromises[i + j] = processor(batches[i + j]);
           }
      }
      
      const result = await batchPromises[i];
      yield { batch: batches[i], result, index: i, total: batches.length };
  }
}

async function main() {
  const isReprocess = process.argv.includes("--reprocess");
  
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
  const sourceLang = settings.sourceLanguageTag;

  if (!sourceLang) {
    console.error("Source language not found in settings.");
    process.exit(1);
  }

  const targetLangs = (settings.languageTags || []).filter((tag) => tag !== sourceLang);
  const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout,
  });

  // Map to store allowed variables for each message ID
  const messageVariables = new Map<string, Set<string>>();

  // Helper to extract variables from a pattern
  const extractVariables = (pattern: any[]) => {
      const vars = new Set<string>();
      for (const node of pattern) {
          if (node.type === "expression" && (node.arg?.type === "variable" || node.arg?.type === "variable-reference")) {
              vars.add(node.arg.name);
          }
      }
      return vars;
  };

  for (const targetLang of targetLangs) {
    console.log(`\nProcessing language: ${targetLang}...`);

    const itemsToProcess: { id: string; text: string; bundle: any; current?: string }[] = [];

    for (const bundle of bundles) {
      const sourceMsg = bundle.messages.find((m: any) => m.locale === sourceLang);
      const targetMsg = bundle.messages.find((m: any) => m.locale === targetLang);

      // Extract source text
      let sourceText = "";
      if (sourceMsg && sourceMsg.variants[0]) {
         // Extract allowed variables from source
         messageVariables.set(bundle.id, extractVariables(sourceMsg.variants[0].pattern));

         sourceText = sourceMsg.variants[0].pattern.map((p: any) => {
            if (p.type === "text") return p.value;
            if (p.type === "expression") return `{${p.arg.name}}`;
            return "";
        }).join("");
      }

      if (!sourceText) continue;

      if (isReprocess) {
        // For reprocess, we look for EXISTING translations
        if (targetMsg) {
             let currentText = "";
             if (targetMsg.variants[0]) {
                currentText = targetMsg.variants[0].pattern.map((p: any) => {
                    if (p.type === "text") return p.value;
                    if (p.type === "expression") return `{${p.arg.name}}`;
                    return "";
                }).join("");
             }
             if (currentText) {
                 itemsToProcess.push({
                     id: bundle.id,
                     text: sourceText,
                     bundle: bundle,
                     current: currentText
                 });
             }
        }
      } else {
        // For standard translate, we look for MISSING translations
        if (!targetMsg) {
            itemsToProcess.push({ 
                id: bundle.id, 
                text: sourceText, 
                bundle: bundle 
            });
        }
      }
    }

    if (itemsToProcess.length === 0) {
      console.log(`No items to process for ${targetLang}.`);
      continue;
    }

    console.log(`Found ${itemsToProcess.length} items for ${targetLang}.`);

    if (isReprocess) {
        // Process with buffered iterator for parallelism
        const iterator = createBufferedIterator(
            itemsToProcess,
            BATCH_SIZE,
            (b) => {
                // Transform for the API
                const reprocessInput = b.map(item => ({ id: item.id, english: item.text, current: item.current! }));
                return reprocessBatch(reprocessInput, targetLang);
            }
        );

        for await (const { batch, result: results, index, total } of iterator) {
            console.log(`\n--- Batch ${index + 1} of ${total} Ready ---`);
            
            for (const res of results) {
               const item = batch.find(b => b.id === res.id);
               if (!item) continue;

               if (res.ok) {
                   // NOOP
               } else if (res.newTranslation) {
                   console.log(`\n----------------------------------------`);
                   console.log(`Key: ${item.id}`);
                   console.log(`English: "${item.text}"`);
                   console.log(`Current: "${item.current}"`);
                   console.log(`Suggestion: "${res.newTranslation}"`);
                   console.log(`Reason: ${res.explanation}`);
                   
                   const answer = await askQuestion("Approve replacement? (y/N): ", rl);
                   if (answer.toLowerCase() === 'y') {
                       console.log(`[${res.id}] Updating...`);

                        const bundle = item.bundle;
                         const updatedMessages = bundle.messages.map((msg: any) => {
                             if (msg.locale === targetLang) {
                                 return {
                                     ...msg,
                                     variants: [{
                                         id: randomUUID(), // New variant ID
                                         messageId: msg.id,
                                         matches: [],
                                         pattern: parsePattern(res.newTranslation!, messageVariables.get(item.id)) 
                                     }]
                                 }
                             }
                             return msg;
                         });
                         
                         await upsertBundleNested(project.db, { ...bundle, messages: updatedMessages });
                   } else {
                       console.log("Skipped.");
                   }
               }
            }
        }
    } else {
        // Standard Translation Logic
        for (let i = 0; i < itemsToProcess.length; i += BATCH_SIZE) {
            const batch = itemsToProcess.slice(i, i + BATCH_SIZE);
            console.log(`Batch ${Math.floor(i / BATCH_SIZE) + 1} / ${Math.ceil(itemsToProcess.length / BATCH_SIZE)}`);
            
            const results = await translateBatch(batch, targetLang);
            console.log("Batch results:");
            let updates = 0;
            for (const item of results) {
                const originalTask = batch.find((b) => b.id === item.id);
                if (originalTask && item.translation) {
                    console.log(` - ${item.id}: "${item.translation}"`);
                    const bundle = originalTask.bundle;
                    
                    const messageId = randomUUID();
                    const variantId = randomUUID();
                    const pattern = parsePattern(item.translation, messageVariables.get(item.id));
                    
                    const newMessage = {
                        id: messageId,
                        bundleId: bundle.id,
                        locale: targetLang,
                        selectors: [],
                        variants: [
                            {
                                id: variantId,
                                messageId: messageId,
                                matches: [],
                                pattern: pattern as any 
                            }
                        ]
                    };
                    
                    const updatedBundle = {
                        ...bundle,
                        messages: [...bundle.messages, newMessage]
                    };

                    await upsertBundleNested(project.db, updatedBundle);
                    updates++;
                }
            }
            console.log(`Updated ${updates} messages.`);
        }
    }

    // Explicitly save the project after processing each language to ensure persistence
    console.log(`Saving progress for ${targetLang}...`);
    await saveProjectToDirectory({
       fs: nodeFs,
       project: project,
       path: path.resolve(process.cwd(), PROJECT_PATH)
    });
  }

  console.log("\nDone.");
  rl.close();
  process.exit(0); // Allow pending file writes to complete
}

main().catch(console.error);
