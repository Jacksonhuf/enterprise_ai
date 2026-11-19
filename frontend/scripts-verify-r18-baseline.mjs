#!/usr/bin/env node
/** Compare R18 spot captures against committed PNG baselines (pixel gate, v1.05). */
import { readFileSync, writeFileSync, existsSync } from 'node:fs';
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
const ratioConfigPath = join(baselineRoot, 'max-diff-ratio.json');
const ratioConfig = existsSync(ratioConfigPath)
  ? JSON.parse(readFileSync(ratioConfigPath, 'utf8'))
  : {default: Number(process.env.COGNARA_R18_MAX_DIFF_RATIO || '0.02')};
const manifestPath = join(captureDir, 'manifest.json');
if (!existsSync(manifestPath)) {
  console.error('missing manifest.json in', captureDir);
  process.exit(2);
}
const manifest = JSON.parse(readFileSync(manifestPath, 'utf8'));
let failed = false;
const measurements = [];

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
  if (base.width !== got.width) {
    console.error(`${spot.slug}: width ${got.width} != baseline ${base.width}`);
    failed = true;
    continue;
  }
  const height = Math.min(base.height, got.height);
  if (Math.abs(base.height - got.height) > 64) {
    console.error(`${spot.slug}: height delta ${Math.abs(base.height - got.height)}px > 64`);
    failed = true;
    continue;
  }
  const rowBytes = base.width * 4;
  const baseData = base.data.subarray(0, rowBytes * height);
  const gotData = got.data.subarray(0, rowBytes * height);
  if (base.height !== got.height) {
    console.warn(`${spot.slug}: comparing top ${height}px (${base.width}x${base.height} vs ${got.width}x${got.height})`);
  }
  const diffPixels = pixelmatch(baseData, gotData, null, base.width, height, {
    threshold: 0.2,
    includeAA: false,
  });
  const ratio = diffPixels / (base.width * height);
  const maxRatio = Number(
    process.env.COGNARA_R18_MAX_DIFF_RATIO ||
      ratioConfig[spot.slug] ||
      ratioConfig.default ||
      0.02,
  );
  console.log(`${spot.slug}: diff=${diffPixels} (${(ratio * 100).toFixed(3)}%) max=${maxRatio}`);
  measurements.push({
    slug: spot.slug,
    measured_ratio: Number(ratio.toFixed(6)),
    max_diff_ratio: maxRatio,
    headroom: Number((maxRatio - ratio).toFixed(6)),
  });
  if (ratio > maxRatio) {
    console.error(`${spot.slug}: exceeds max ratio ${maxRatio}`);
    failed = true;
  }
}
const measuredPath = process.env.COGNARA_R18_PIXEL_MEASURED;
if (measuredPath) {
  writeFileSync(
    measuredPath,
    `${JSON.stringify({gate: 'r18-pixel-baseline', spots: measurements}, null, 2)}\n`,
  );
}
if (failed) process.exit(1);
console.log(`R18 baseline gate OK`);