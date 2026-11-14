#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const guardUtilsPath = path.join(__dirname, 'node_modules/@sveltejs/kit/src/exports/vite/utils.js');

if (!fs.existsSync(guardUtilsPath)) {
  console.log('SvelteKit utils not found, skipping patch');
  process.exit(0);
}

let content = fs.readFileSync(guardUtilsPath, 'utf8');

const oldPattern = "export const server_only_directory_pattern = /\\/server\\//;";
const newPattern = "export const server_only_directory_pattern = /\\/src\\/server\\//;";

if (content.includes(oldPattern)) {
  content = content.replace(oldPattern, newPattern);
  fs.writeFileSync(guardUtilsPath, content, 'utf8');
  console.log('Patched SvelteKit server_only_directory_pattern to only match /src/server/');
} else if (content.includes(newPattern)) {
  console.log('SvelteKit already patched');
} else {
  console.log('Could not find pattern to patch in SvelteKit');
}
