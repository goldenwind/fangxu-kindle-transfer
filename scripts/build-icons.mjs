import { execFileSync } from 'node:child_process';
import { copyFileSync, mkdtempSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { resolve, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = fileURLToPath(new URL('../', import.meta.url));
const output = mkdtempSync(join(tmpdir(), 'fangxu-kindle-icons-'));
try {
  execFileSync(process.execPath, [
    resolve(root, 'node_modules/@tauri-apps/cli/tauri.js'), 'icon',
    resolve(root, 'docs/app-icon.svg'), '--output', output,
  ], { cwd: root, stdio: 'inherit' });
  for (const name of ['32x32.png', '128x128.png', '128x128@2x.png', 'icon.png', 'icon.icns', 'icon.ico']) {
    copyFileSync(join(output, name), resolve(root, 'src-tauri/icons', name));
  }
} finally {
  rmSync(output, { recursive: true, force: true });
}
