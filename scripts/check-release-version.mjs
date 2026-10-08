import { readFileSync } from 'node:fs';
import assert from 'node:assert/strict';

const read = path => readFileSync(new URL(`../${path}`, import.meta.url), 'utf8');
const packageInfo = JSON.parse(read('package.json'));
const lock = JSON.parse(read('package-lock.json'));
const config = JSON.parse(read('src-tauri/tauri.conf.json'));
const cargoVersion = read('src-tauri/Cargo.toml').match(/\[package\][\s\S]*?\nversion = "([^"]+)"/)?.[1];
const cargoLockVersion = read('src-tauri/Cargo.lock').match(/name = "fangxu-kindle-transfer-desktop"\nversion = "([^"]+)"/)?.[1];
for (const [source, version] of Object.entries({
  'package-lock.json': lock.version,
  'package-lock.json root package': lock.packages[''].version,
  'tauri.conf.json': config.version,
  'Cargo.toml': cargoVersion,
  'Cargo.lock': cargoLockVersion,
})) assert.equal(version, packageInfo.version, `${source} must match package.json`);
const tag = process.argv[2];
if (tag) assert.equal(tag, `v${packageInfo.version}`, 'Release tag must match the application version');
console.log(`Release version verified: v${packageInfo.version}`);
