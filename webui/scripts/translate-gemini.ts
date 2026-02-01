
import "dotenv/config";
import { GoogleGenerativeAI, SchemaType } from "@google/generative-ai";
import { loadProjectFromDirectory, selectBundleNested, upsertBundleNested } from "@inlang/sdk";
import { openRepository } from "@lix-js/client";
import fs from "node:fs";
import path from "node:path";
import { pathToFileURL } from "node:url";
import { randomUUID } from "node:crypto";
import * as readline from "node:readline";

const PROJECT_PATH = "./project.inlang";
const MODEL_NAME = "gemini-2.5-flash"; // Updated per previous edit
const BATCH_SIZE = 32; // Smaller batch size for interactive mode consistency

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
};

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
};

const model = genAI.getGenerativeModel({
  model: MODEL_NAME,
});

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
    
    Context about Backrest:
    - It is a GUI and scheduler for the "restic" backup tool.
    - Key operations: 
      - "Backup": Creates snapshots of data.
      - "Prune": Removes unused data from the repository (repacks).
      - "Forget": Manages snapshot retention policies (e.g., keep last N).
      - "Check": Verifies repository integrity.
    - Features: Cron scheduling, multi-platform (Linux/macOS/Windows), supports various storage backends (S3, B2, Local, SFTP).
    
    Translate the following texts to ${targetLang}.
    
    Rules:
    1. Maintain all variables (e.g. {name}) exactly.
    2. Do not add explanations.
    3. Return a JSON array where each object contains the 'id' (from input) and the 'translation'.
    4. Use terminology consistent with backup software (e.g., "snapshot", "repository", "retention").
    
    Variables in text are enclosed in braces {}.
    
    Input JSON:
    ${JSON.stringify(payload, null, 2)}
  `;

  try {
    const result = await model.generateContent({
      contents: [{ role: "user", parts: [{ text: prompt }] }],
      generationConfig: {
        responseMimeType: "application/json",
        responseSchema: translationSchema,
      },
    });
    return JSON.parse(result.response.text());
  } catch (error: any) {
    console.error("Batch translation failed:", error.message);
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

    For each item:
    1. accurate: Is the 'current' translation accurate and does it use correct terminology? (bool "ok")
    2. fit: Does it preserve variables (e.g. {name})?
    
    If "ok" is false:
    - Provide a better "newTranslation".
    - Provide a brief "explanation" (in English) of why the old one was incorrect or suboptimal.
    
    Context:
    - "Prune" = removing unused data / repacking.
    - "Forget" = removing old snapshots based on policy.
    - "Snapshot" = a backup point in time.

    Input JSON:
    ${JSON.stringify(payload, null, 2)}
  `;

  try {
    const result = await model.generateContent({
      contents: [{ role: "user", parts: [{ text: prompt }] }],
      generationConfig: {
        responseMimeType: "application/json",
        responseSchema: reprocessSchema,
      },
    });
    return JSON.parse(result.response.text());
  } catch (error: any) {
    console.error("Batch reprocessing failed:", error.message);
    return [];
  }
}

function askQuestion(query: string, rl: readline.Interface): Promise<string> {
  return new Promise((resolve) => rl.question(query, resolve));
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

  const targetLangs = settings.languageTags.filter((tag) => tag !== sourceLang);
  const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout,
  });

  for (const targetLang of targetLangs) {
    console.log(`\nProcessing language: ${targetLang}...`);

    const itemsToProcess: { id: string; text: string; bundle: any; current?: string }[] = [];

    for (const bundle of bundles) {
      const sourceMsg = bundle.messages.find((m: any) => m.locale === sourceLang);
      const targetMsg = bundle.messages.find((m: any) => m.locale === targetLang);

      // Extract source text
      let sourceText = "";
      if (sourceMsg && sourceMsg.variants[0]) {
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

    for (let i = 0; i < itemsToProcess.length; i += BATCH_SIZE) {
      const batch = itemsToProcess.slice(i, i + BATCH_SIZE);
      console.log(`Batch ${Math.floor(i / BATCH_SIZE) + 1} / ${Math.ceil(itemsToProcess.length / BATCH_SIZE)}`);

      if (isReprocess) {
          const reprocessInput = batch.map(b => ({ id: b.id, english: b.text, current: b.current! }));
          const results = await reprocessBatch(reprocessInput, targetLang);
          
          for (const res of results) {
              if (res.ok) continue;

              const item = batch.find(b => b.id === res.id);
              if (!item) continue;

              console.log(`\n----------------------------------------`);
              console.log(`Key: ${item.id}`);
              console.log(`English: "${item.text}"`);
              console.log(`Current: "${item.current}"`);
              console.log(`Suggestion: "${res.newTranslation}"`);
              console.log(`Reason: ${res.explanation}`);
              
              const answer = await askQuestion("Approve replacement? (y/N): ", rl);
              if (answer.toLowerCase() === 'y' && res.newTranslation) {
                  // Apply update
                   const bundle = item.bundle;
                   // Find existing message index to update or replace variants?
                   // Simplest is to map over messages and update the one for this locale.
                   const updatedMessages = bundle.messages.map((msg: any) => {
                       if (msg.locale === targetLang) {
                           return {
                               ...msg,
                               variants: [{
                                   id: randomUUID(), // New variant ID
                                   messageId: msg.id,
                                   matches: [],
                                   pattern: [{ type: "text", value: res.newTranslation }] // Simplified pattern
                               }]
                           }
                       }
                       return msg;
                   });

                   await upsertBundleNested(project.db, { ...bundle, messages: updatedMessages });
                   console.log("Updated.");
              } else {
                  console.log("Skipped.");
              }
          }

      } else {
        // Standard Translation Logic
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
                const pattern = [{ type: "text", value: item.translation }];
                
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
  }

  console.log("\nDone.");
  rl.close();
  // process.exit(0); // Allow pending file writes to complete
}

main().catch(console.error);
