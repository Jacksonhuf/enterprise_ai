#!/usr/bin/env node
/** Compare R18 spot captures against committed PNG baselines (pixel gate, v1.05). */
import { readFileSync, existsSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { PNG } from 'pngjs';
import pixelmatch from 'pixelmatch';

const captureDir = process.argv[2];
if (!captureDir) {
  console.error('usage: node scripts-verify-r18-baseline.mjs <capture-dir>');
  process.exit(2);
}
const baselineRoot = join(dirname(fileURLToPath(import.meta.url)), 'r18-baselines');
const manifestPath = join(captureDir, 'manifest.json');
if (!existsSync(manifestPath)) {
  console.error('missing manifest.json in', captureDir);
  process.exit(2);
}
const manifest = JSON.parse(readFileSync(manifestPath, 'utf8'));
const maxRatio = Number(process.env.COGNARA_R18_MAX_DIFF_RATIO || '0.02');
let failed = false;

for (const spot of manifest.spots) {
  const basePath = join(baselineRoot, spot.file);
  const gotPath = join(captureDir, spot.file);
  if (!existsSync(basePath) || !existsSync(gotPath)) {
    console.error('missing file for', spot.slug);
    failed = true;
    continue;
  }
  const base = PNG.sync.read(readFileSync(basePath));
  const got = PNG.sync.read(readFileSync(gotPath));
  if (base.width !== got.width || base.height !== got.height) {
    console.error(`${spot.slug}: size ${got.width}x${got.height} != baseline ${base.width}x${base.height}`);
    failed = true;
    continue;
  }
  const diffPixels = pixelmatch(base.data, got.data, null, base.width, base.height, {
    threshold: 0.2,
    includeAA: false,
  });
  const ratio = diffPixels / (base.width * base.height);
  console.log(`${spot.slug}: diff=${diffPixels} (${(ratio * 100).toFixed(3)}%)`);
  if (ratio > maxRatio) {
    console.error(`${spot.slug}: exceeds max ratio ${maxRatio}`);
    failed = true;
  }
}
if (failed) process.exit(1);
console.log(`R18 baseline gate OK (max ratio ${maxRatio})`);