#!/usr/bin/env node
/**
 * ollama-termux installer for Termux/Android
 *
 * Downloads a pre-built release tarball from GitHub and installs
 * the binary + ggml .so backends to the Termux prefix.
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

  // Install binary
  const extractedBin = path.join(tmpDir, 'bin', 'ollama');
  if (fs.existsSync(extractedBin)) {
    fs.copyFileSync(extractedBin, OLLAMA_BIN);
    fs.chmodSync(OLLAMA_BIN, 0o755);
    log('Installed: ' + OLLAMA_BIN);
  }

  // Install ggml backends
  const extractedLib = path.join(tmpDir, 'lib', 'ollama');
  if (fs.existsSync(extractedLib)) {
    fs.mkdirSync(OLLAMA_LIB, { recursive: true });
    const soFiles = fs.readdirSync(extractedLib).filter(f => f.endsWith('.so'));
    for (const so of soFiles) {
      const src = path.join(extractedLib, so);
      const dst = path.join(OLLAMA_LIB, so);
      backupIfExists(dst);
      fs.copyFileSync(src, dst);
      log('Installed: ' + path.join('lib/ollama', so));
    }
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
  log('  ollama pull qwen3.5');
  log('  ollama pull gemma4');
  log('  ollama launch claude --model qwen3.5');
  log('  ollama launch codex --model gemma4');
}

main().catch((e) => {
  console.error('[ollama-termux] Installation failed:', e.message);
  process.exit(1);
});
