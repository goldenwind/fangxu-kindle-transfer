import test from 'node:test';
import assert from 'node:assert/strict';
import { execFileSync, spawnSync } from 'node:child_process';
import { mkdtempSync, readFileSync, readdirSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';

const script = fileURLToPath(new URL('../scripts/package-windows-portable.ps1', import.meta.url));
const config = JSON.parse(readFileSync(new URL('../src-tauri/tauri.conf.json', import.meta.url), 'utf8'));
const skip = process.platform !== 'win32';
const run = (directory) => spawnSync('pwsh', ['-NoProfile', '-File', script, '-ReleaseDir', directory], { encoding: 'utf8' });

test('Windows portable ZIP extracts with its client, sidecar, runtime DLLs and instructions', { skip }, () => {
  const directory = mkdtempSync(join(tmpdir(), 'kindle portable '));
  try {
    writeFileSync(join(directory, 'fangxu-kindle-transfer-desktop.exe'), 'client fixture');
    writeFileSync(join(directory, 'fangxu-kindle-service.exe'), 'service fixture');
    writeFileSync(join(directory, 'WebView2Loader.dll'), 'runtime fixture');
    writeFileSync(join(directory, 'unrelated.pdb'), 'do not distribute');
    for (let attempt = 0; attempt < 2; attempt++) {
      const result = run(directory);
      assert.equal(result.status, 0, result.error?.message || result.stderr);
    }
    const archive = join(directory, 'bundle', 'portable', `fangxu-kindle-transfer_${config.version}_windows-x64-portable.zip`);
    const extracted = join(directory, 'extracted');
    execFileSync('pwsh', ['-NoProfile', '-Command', 'Expand-Archive -LiteralPath $env:KINDLE_TEST_ZIP -DestinationPath $env:KINDLE_TEST_EXTRACT'], {
      env: { ...process.env, KINDLE_TEST_ZIP: archive, KINDLE_TEST_EXTRACT: extracted },
    });
    const app = join(extracted, 'Fangxu-Kindle-Transfer');
    assert.deepEqual(readdirSync(app).sort(), ['Fangxu-Kindle-Transfer.exe', 'LICENSE.txt', 'README.txt', 'WebView2Loader.dll', 'fangxu-kindle-service.exe'].sort());
    assert.equal(readFileSync(join(app, 'Fangxu-Kindle-Transfer.exe'), 'utf8'), 'client fixture');
    assert.equal(readFileSync(join(app, 'fangxu-kindle-service.exe'), 'utf8'), 'service fixture');
    assert.equal(readFileSync(join(app, 'WebView2Loader.dll'), 'utf8'), 'runtime fixture');
    assert.match(readFileSync(join(app, 'README.txt'), 'utf8'), /双击 Fangxu-Kindle-Transfer\.exe/);
  } finally {
    rmSync(directory, { recursive: true, force: true });
  }
});

test('Windows portable packaging rejects builds missing either executable', { skip }, () => {
  for (const existing of ['fangxu-kindle-transfer-desktop.exe', 'fangxu-kindle-service.exe']) {
    const directory = mkdtempSync(join(tmpdir(), 'kindle incomplete '));
    try {
      writeFileSync(join(directory, existing), 'fixture');
      const result = run(directory);
      assert.notEqual(result.status, 0, result.error?.message);
      assert.match(result.stderr, /Missing portable binary/);
      assert.deepEqual(readdirSync(directory), [existing]);
    } finally {
      rmSync(directory, { recursive: true, force: true });
    }
  }
});
