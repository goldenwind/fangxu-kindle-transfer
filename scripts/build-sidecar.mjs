import { execFileSync } from 'node:child_process';
import { mkdirSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { resolve } from 'node:path';

const root = fileURLToPath(new URL('../', import.meta.url));
const host = execFileSync('rustc', ['-vV'], { encoding: 'utf8' }).match(/^host: (.+)$/m)?.[1];
const target = process.env.TAURI_ENV_TARGET_TRIPLE || process.argv[2] || host;
const targets = {
  'aarch64-apple-darwin': ['darwin', 'arm64'],
  'x86_64-apple-darwin': ['darwin', 'amd64'],
  'x86_64-pc-windows-msvc': ['windows', 'amd64'],
  'aarch64-pc-windows-msvc': ['windows', 'arm64'],
  'x86_64-unknown-linux-gnu': ['linux', 'amd64'],
  'aarch64-unknown-linux-gnu': ['linux', 'arm64'],
};
if (!targets[target]) throw new Error(`Unsupported target: ${target}`);
const [GOOS, GOARCH] = targets[target];
mkdirSync(resolve(root, 'src-tauri/binaries'), { recursive: true });
const output = resolve(root, `src-tauri/binaries/fangxu-kindle-service-${target}${GOOS === 'windows' ? '.exe' : ''}`);
execFileSync('go', ['build', '-trimpath', '-ldflags=-s -w', '-o', output, '.'], {
  cwd: root, stdio: 'inherit', env: { ...process.env, GOOS, GOARCH, CGO_ENABLED: '0' },
});
console.log(`Built Kindle service for ${target}`);
