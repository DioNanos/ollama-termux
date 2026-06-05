#!/usr/bin/env node
/**
 * ollama-termux installer for Termux/Android
 *
 * Downloads a pre-built release tarball from GitHub and installs
 * the binary + the llama-server runtime (llama-server + ggml CPU
 * variant libraries) to the Termux prefix.
 * No Go toolchain required on-device.
 */

const { execFileSync } = require('child_process');
const crypto = require('crypto');
const fs = require('fs');
const https = require('https');
const http = require('http');
const os = require('os');
const path = require('path');

const TERMUX_PREFIX = '/data/data/com.termux/files/usr';
const OLLAMA_BIN = path.join(TERMUX_PREFIX, 'bin', 'ollama');
const OLLAMA_LIB = path.join(TERMUX_PREFIX, 'lib', 'ollama');
const OLLAMA_REAL_BIN = path.join(OLLAMA_LIB, 'ollama');
const GITHUB_REPO = 'DioNanos/ollama-termux';
const VERSION = require('./package.json').version;

function log(msg) {
  console.log(`[ollama-termux] ${msg}`);
}

function isTermux() {
  return fs.existsSync(TERMUX_PREFIX);
}

function fetchUrl(url) {
  return new Promise((resolve, reject) => {
    const mod = url.startsWith('https') ? https : http;
    const follow = (u, redirects) => {
      if (redirects > 5) return reject(new Error('too many redirects'));
      mod.get(u, { headers: { 'User-Agent': 'ollama-termux-installer' } }, (res) => {
        if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
          follow(res.headers.location, redirects + 1);
          return;
        }
        if (res.statusCode !== 200) {
          reject(new Error(`HTTP ${res.statusCode} for ${u}`));
          return;
        }
        resolve(res);
      }).on('error', reject);
    };
    follow(url, 0);
  });
}

async function downloadAndVerify(url, dest, expectedSha256) {
  const tmpDest = dest + '.tmp';
  const res = await fetchUrl(url);
  const fileStream = fs.createWriteStream(tmpDest);
  const hash = crypto.createHash('sha256');

  res.pipe(hash);
  res.pipe(fileStream);

  await new Promise((resolve, reject) => {
    fileStream.on('finish', resolve);
    fileStream.on('error', reject);
    res.on('error', reject);
  });

  const actualSha = hash.digest('hex');
  if (expectedSha256 && actualSha !== expectedSha256) {
    fs.unlinkSync(tmpDest);
    throw new Error(`SHA256 mismatch: expected ${expectedSha256}, got ${actualSha}`);
  }

  fs.renameSync(tmpDest, dest);
  return actualSha;
}

function backupIfExists(filePath) {
  if (fs.existsSync(filePath)) {
    const backup = filePath + '.orig';
    log(`Backing up ${path.basename(filePath)} to ${path.basename(backup)}`);
    fs.copyFileSync(filePath, backup);
  }
}

function writeOllamaWrapper() {
  const script = `#!/data/data/com.termux/files/usr/bin/sh
PREFIX="\${PREFIX:-/data/data/com.termux/files/usr}"
OLLAMA_REAL_BIN="$PREFIX/lib/ollama/ollama"
export LD_LIBRARY_PATH="$PREFIX/lib:$PREFIX/lib/ollama:$PREFIX/lib/ollama/vulkan\${LD_LIBRARY_PATH:+:$LD_LIBRARY_PATH}"
exec "$OLLAMA_REAL_BIN" "$@"
`;
  fs.writeFileSync(OLLAMA_BIN, script, { mode: 0o755 });
  fs.chmodSync(OLLAMA_BIN, 0o755);
}

function removeInstalledBackends() {
  if (!fs.existsSync(OLLAMA_LIB)) return;
  for (const entry of fs.readdirSync(OLLAMA_LIB, { withFileTypes: true })) {
    if (entry.name === 'ollama' || entry.name === 'ollama.orig') continue;
    const target = path.join(OLLAMA_LIB, entry.name);
    if (entry.isDirectory() || entry.name.endsWith('.so') || entry.name === 'llama-server') {
      fs.rmSync(target, { recursive: true, force: true });
    }
  }
}

function copyTree(srcDir, dstDir, relDir = '') {
  fs.mkdirSync(dstDir, { recursive: true });
  for (const entry of fs.readdirSync(srcDir, { withFileTypes: true })) {
    const src = path.join(srcDir, entry.name);
    const dst = path.join(dstDir, entry.name);
    const rel = path.join(relDir, entry.name);

    if (entry.isDirectory()) {
      copyTree(src, dst, rel);
      continue;
    }

    backupIfExists(dst);
    fs.copyFileSync(src, dst);
    // Preserve the source mode: lib/ollama now ships the llama-server
    // executable alongside the .so libraries.
    fs.chmodSync(dst, fs.statSync(src).mode & 0o777);
    log('Installed: ' + path.join('lib/ollama', rel));
  }
}

async function main() {
  if (!isTermux()) {
    console.log('[ollama-termux] This installer is for Termux/Android only.');
    console.log('');
    console.log('For manual installation or cross-compilation, see:');
    console.log('  https://github.com/DioNanos/ollama-termux#building');
    return;
  }

  log(`Installing ollama-termux v${VERSION}...`);
  log('');

  const tarballName = `ollama-termux-${VERSION}-android-arm64.tar.gz`;
  const tmpBase = process.env.TMPDIR || os.tmpdir() || path.join(TERMUX_PREFIX, 'tmp');
  const tmpDir = path.join(tmpBase, 'ollama-termux-install');
  fs.mkdirSync(tmpDir, { recursive: true });

  const tarballPath = path.join(tmpDir, tarballName);

  // Download from GitHub releases
  const tarballUrl = `https://github.com/${GITHUB_REPO}/releases/download/v${VERSION}/${tarballName}`;
  const sha256Url = tarballUrl + '.sha256';

  log(`Downloading ${tarballName}...`);
  let expectedSha = null;
  try {
    const shaRes = await fetchUrl(sha256Url);
    const shaText = await new Promise((resolve, reject) => {
      let data = '';
      shaRes.on('data', (chunk) => data += chunk);
      shaRes.on('end', () => resolve(data));
      shaRes.on('error', reject);
    });
    expectedSha = shaText.trim().split(/\s+/)[0];
    log(`Expected SHA256: ${expectedSha.substring(0, 16)}...`);
  } catch {
    log('Note: SHA256 checksum not available, skipping verification');
  }

  try {
    const actualSha = await downloadAndVerify(tarballUrl, tarballPath, expectedSha);
    if (!expectedSha) {
      log(`Downloaded (SHA256: ${actualSha.substring(0, 16)}...)`);
    } else {
      log('Checksum verified');
    }
  } catch (e) {
    log(`Download failed: ${e.message}`);
    log('');
    log('The matching GitHub Release asset may not exist for this version yet.');
    log('Build manually with:');
    log('  pkg install golang');
    log('  git clone https://github.com/DioNanos/ollama-termux.git');
    log('  cd ollama-termux && go build -o ollama -ldflags="-s -w" -trimpath .');
    log('  cp ollama ' + OLLAMA_BIN);
    process.exit(1);
  }

  // Extract tarball
  log('Extracting...');
  execFileSync('tar', ['-xzf', tarballPath, '-C', tmpDir], { stdio: 'pipe' });

  // Backup existing files
  backupIfExists(OLLAMA_BIN);
  fs.mkdirSync(OLLAMA_LIB, { recursive: true });

  // Install the real binary under lib/ollama and expose a Termux-safe wrapper
  // from bin/ollama so clean shells can resolve libc++_shared.so.
  const extractedBin = path.join(tmpDir, 'bin', 'ollama');
  if (fs.existsSync(extractedBin)) {
    backupIfExists(OLLAMA_REAL_BIN);
    fs.copyFileSync(extractedBin, OLLAMA_REAL_BIN);
    fs.chmodSync(OLLAMA_REAL_BIN, 0o755);
    writeOllamaWrapper();
    log('Installed: ' + OLLAMA_REAL_BIN);
    log('Installed wrapper: ' + OLLAMA_BIN);
  }

  // Install ggml backends
  const extractedLib = path.join(tmpDir, 'lib', 'ollama');
  if (fs.existsSync(extractedLib)) {
    removeInstalledBackends();
    copyTree(extractedLib, OLLAMA_LIB);
  }

  // Cleanup
  try {
    fs.rmSync(tmpDir, { recursive: true });
  } catch {}

  log('');
  log('ollama-termux installed successfully!');
  log('');
  log('Quick start:');
  log('  ollama serve &');
  log('  ollama pull qwen3.5:4b');
  log('  ollama pull gemma4:e4b');
  log('  ollama launch codex --model qwen3.5:4b');
  log('  ollama launch codex-vl --model gemma4:e4b');
  log('  ollama launch qwen --model qwen3.5:4b');
  log('  ollama launch pi');
}

main().catch((e) => {
  console.error('[ollama-termux] Installation failed:', e.message);
  process.exit(1);
});
