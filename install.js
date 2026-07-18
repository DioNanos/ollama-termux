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
    const follow = (u, redirects) => {
      if (redirects > 5) return reject(new Error('too many redirects'));
      let parsed;
      try {
        parsed = new URL(u);
      } catch (e) {
        return reject(new Error(`invalid download URL: ${e.message}`));
      }
      if (parsed.protocol !== 'https:') {
        return reject(new Error(`refusing non-HTTPS download URL: ${parsed.href}`));
      }

      const req = https.get(parsed, { headers: { 'User-Agent': 'ollama-termux-installer' } }, (res) => {
        if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
          const next = new URL(res.headers.location, parsed).href;
          res.resume();
          follow(next, redirects + 1);
          return;
        }
        if (res.statusCode !== 200) {
          res.resume();
          reject(new Error(`HTTP ${res.statusCode} for ${u}`));
          return;
        }
        res.setTimeout(60_000, () => res.destroy(new Error(`download stalled for ${u}`)));
        resolve(res);
      });
      req.setTimeout(30_000, () => req.destroy(new Error(`connection timed out for ${u}`)));
      req.on('error', reject);
    };
    follow(url, 0);
  });
}

async function downloadAndVerify(url, dest, expectedSha256) {
  if (!/^[0-9a-f]{64}$/.test(expectedSha256)) {
    throw new Error('a valid SHA256 checksum is required');
  }

  const tmpDest = dest + '.tmp';
  try {
    const res = await fetchUrl(url);
    const fileStream = fs.createWriteStream(tmpDest, { flags: 'wx', mode: 0o600 });
    const hash = crypto.createHash('sha256');
    res.on('data', (chunk) => hash.update(chunk));
    res.pipe(fileStream);

    await new Promise((resolve, reject) => {
      fileStream.on('finish', resolve);
      fileStream.on('error', reject);
      res.on('aborted', () => reject(new Error(`download aborted for ${url}`)));
      res.on('error', reject);
    });

    const actualSha = hash.digest('hex');
    if (actualSha !== expectedSha256) {
      throw new Error(`SHA256 mismatch: expected ${expectedSha256}, got ${actualSha}`);
    }

    fs.renameSync(tmpDest, dest);
    return actualSha;
  } catch (e) {
    fs.rmSync(tmpDest, { force: true });
    throw e;
  }
}

function parseChecksum(text, expectedFilename) {
  const match = /^([0-9a-fA-F]{64})\s+\*?([^\r\n]+)\s*$/.exec(text);
  if (!match) {
    throw new Error('invalid SHA256 checksum file');
  }
  if (match[2] !== expectedFilename) {
    throw new Error(`checksum names ${match[2]}, expected ${expectedFilename}`);
  }
  return match[1].toLowerCase();
}

function normalizedArchiveEntry(raw) {
  const entry = raw.replace(/^\.\/+/, '').replace(/\/+$/, '');
  if (!entry || entry === '.') return '';
  if (entry.includes('\0') || path.posix.isAbsolute(entry)) {
    throw new Error(`unsafe archive path: ${raw}`);
  }
  const normalized = path.posix.normalize(entry);
  if (normalized === '..' || normalized.startsWith('../')) {
    throw new Error(`archive path escapes extraction root: ${raw}`);
  }
  return normalized;
}

function validateArchiveListing(listing) {
  const entries = new Set();
  for (const raw of listing.split(/\r?\n/)) {
    if (!raw) continue;
    const entry = normalizedArchiveEntry(raw);
    if (!entry) continue;
    const allowed = entry === 'install.js' || entry === 'bin' || entry.startsWith('bin/') ||
      entry === 'lib' || entry === 'lib/ollama' || entry.startsWith('lib/ollama/');
    if (!allowed) {
      throw new Error(`unexpected archive path: ${raw}`);
    }
    if (entries.has(entry)) {
      throw new Error(`duplicate archive path: ${raw}`);
    }
    entries.add(entry);
  }

  for (const required of ['bin/ollama', 'lib/ollama/llama-server', 'lib/ollama/libggml-base.so']) {
    if (!entries.has(required)) {
      throw new Error(`release archive is missing ${required}`);
    }
  }
  if (![...entries].some((entry) => /^lib\/ollama\/libggml-cpu.*\.so$/.test(entry))) {
    throw new Error('release archive is missing a ggml CPU backend');
  }
}

function validateArchiveTypes(verboseListing) {
  for (const line of verboseListing.split(/\r?\n/)) {
    if (!line) continue;
    if (line[0] !== '-' && line[0] !== 'd') {
      throw new Error(`link or special archive entry rejected: ${line}`);
    }
    const mode = line.slice(0, 10);
    if (/[sStT]/.test(mode)) {
      throw new Error(`privileged archive mode rejected: ${line}`);
    }
  }
}

function validateExtractedTree(root) {
  const visit = (dir) => {
    for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
      const target = path.join(dir, entry.name);
      const stat = fs.lstatSync(target);
      if (
        stat.isSymbolicLink() ||
        (!stat.isDirectory() && !stat.isFile()) ||
        (stat.isFile() && stat.nlink > 1)
      ) {
        throw new Error(`unsafe extracted entry: ${path.relative(root, target)}`);
      }
      if ((stat.mode & 0o7000) !== 0) {
        throw new Error(`privileged extracted mode: ${path.relative(root, target)}`);
      }
      if (stat.isDirectory()) visit(target);
    }
  };
  visit(root);

  for (const required of ['bin/ollama', 'lib/ollama/llama-server', 'lib/ollama/libggml-base.so']) {
    const target = path.join(root, required);
    if (!fs.statSync(target).isFile()) {
      throw new Error(`release payload is missing ${required}`);
    }
  }
  for (const executable of ['bin/ollama', 'lib/ollama/llama-server']) {
    if ((fs.statSync(path.join(root, executable)).mode & 0o111) === 0) {
      throw new Error(`release payload is not executable: ${executable}`);
    }
  }
  const cpuBackends = fs.readdirSync(path.join(root, 'lib', 'ollama'))
    .filter((name) => /^libggml-cpu.*\.so$/.test(name));
  if (cpuBackends.length === 0) {
    throw new Error('release payload is missing a ggml CPU backend');
  }
}

function backupIfExists(filePath) {
  if (fs.existsSync(filePath)) {
    const backup = filePath + '.orig';
    log(`Backing up ${path.basename(filePath)} to ${path.basename(backup)}`);
    fs.copyFileSync(filePath, backup);
  }
}

function writeOllamaWrapper(target = OLLAMA_BIN) {
  const script = `#!/data/data/com.termux/files/usr/bin/sh
PREFIX="\${PREFIX:-/data/data/com.termux/files/usr}"
OLLAMA_REAL_BIN="$PREFIX/lib/ollama/ollama"
export LD_LIBRARY_PATH="/system/lib64:$PREFIX/lib/ollama:$PREFIX/lib/ollama/vulkan:$PREFIX/lib\${LD_LIBRARY_PATH:+:$LD_LIBRARY_PATH}"
exec "$OLLAMA_REAL_BIN" "$@"
`;
  fs.writeFileSync(target, script, { flags: 'wx', mode: 0o755 });
  fs.chmodSync(target, 0o755);
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
    if (!entry.isFile()) {
      throw new Error(`refusing non-regular release entry: ${rel}`);
    }

    backupIfExists(dst);
    fs.copyFileSync(src, dst);
    // Preserve the source mode: lib/ollama now ships the llama-server
    // executable alongside the .so libraries.
    fs.chmodSync(dst, fs.statSync(src).mode & 0o777);
    log('Installed: ' + path.join('lib/ollama', rel));
  }
}

function activateStagedInstall(stagedLib, stagedWrapper, liveLib, liveWrapper) {
  const suffix = `${process.pid}-${crypto.randomBytes(6).toString('hex')}`;
  const backupLib = `${liveLib}.backup-${suffix}`;
  let liveMoved = false;
  let stagedActivated = false;

  try {
    if (fs.existsSync(liveLib)) {
      fs.renameSync(liveLib, backupLib);
      liveMoved = true;
    }
    fs.renameSync(stagedLib, liveLib);
    stagedActivated = true;

    // Commit point: the wrapper is replaced only after the complete runtime
    // directory is active. No fallible installation step follows this rename.
    fs.renameSync(stagedWrapper, liveWrapper);
  } catch (e) {
    const rollbackErrors = [];
    try {
      if (stagedActivated && fs.existsSync(liveLib)) {
        fs.rmSync(liveLib, { recursive: true, force: true });
      }
    } catch (rollbackError) {
      rollbackErrors.push(`remove staged runtime: ${rollbackError.message}`);
    }
    try {
      if (liveMoved && fs.existsSync(backupLib)) {
        fs.renameSync(backupLib, liveLib);
      }
    } catch (rollbackError) {
      rollbackErrors.push(`restore previous runtime: ${rollbackError.message}`);
    }
    fs.rmSync(stagedWrapper, { force: true });

    if (rollbackErrors.length > 0) {
      throw new Error(`${e.message}; rollback failed: ${rollbackErrors.join('; ')}`);
    }
    throw e;
  }

  if (liveMoved) {
    try {
      fs.rmSync(backupLib, { recursive: true, force: true });
    } catch (e) {
      log(`Warning: unable to remove previous runtime backup ${backupLib}: ${e.message}`);
    }
  }
}

function installPayloadAtomically(extractedBin, extractedLib) {
  const libParent = path.dirname(OLLAMA_LIB);
  fs.mkdirSync(libParent, { recursive: true });
  fs.mkdirSync(path.dirname(OLLAMA_BIN), { recursive: true });
  const stageRoot = fs.mkdtempSync(path.join(libParent, '.ollama-install-'));
  const stagedLib = path.join(stageRoot, 'ollama');
  const stagedWrapper = path.join(stageRoot, 'ollama-wrapper');

  try {
    copyTree(extractedLib, stagedLib);

    // Keep the same one-generation backup behavior as previous releases.
    if (fs.existsSync(OLLAMA_REAL_BIN)) {
      const previousRealBin = path.join(stagedLib, 'ollama.orig');
      fs.copyFileSync(OLLAMA_REAL_BIN, previousRealBin);
      fs.chmodSync(previousRealBin, fs.statSync(OLLAMA_REAL_BIN).mode & 0o777);
    }

    fs.copyFileSync(extractedBin, path.join(stagedLib, 'ollama'));
    fs.chmodSync(path.join(stagedLib, 'ollama'), 0o755);
    writeOllamaWrapper(stagedWrapper);

    // Build and validate the entire replacement before touching live paths.
    if (!fs.statSync(path.join(stagedLib, 'llama-server')).isFile()) {
      throw new Error('staged runtime is missing llama-server');
    }
    backupIfExists(OLLAMA_BIN);
    activateStagedInstall(stagedLib, stagedWrapper, OLLAMA_LIB, OLLAMA_BIN);
  } finally {
    fs.rmSync(stageRoot, { recursive: true, force: true });
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
  fs.mkdirSync(tmpBase, { recursive: true });
  const tmpDir = fs.mkdtempSync(path.join(tmpBase, 'ollama-termux-install-'));

  const tarballPath = path.join(tmpDir, tarballName);

  // Download from GitHub releases
  const tarballUrl = `https://github.com/${GITHUB_REPO}/releases/download/v${VERSION}/${tarballName}`;
  const sha256Url = tarballUrl + '.sha256';

  log(`Downloading ${tarballName}...`);
  try {
    const shaRes = await fetchUrl(sha256Url);
    const shaText = await new Promise((resolve, reject) => {
      let data = '';
      shaRes.on('data', (chunk) => data += chunk);
      shaRes.on('end', () => resolve(data));
      shaRes.on('error', reject);
    });
    const expectedSha = parseChecksum(shaText, tarballName);
    log(`Expected SHA256: ${expectedSha.substring(0, 16)}...`);
    await downloadAndVerify(tarballUrl, tarballPath, expectedSha);
    log('Checksum verified');

    const listing = execFileSync('tar', ['-tzf', tarballPath], { encoding: 'utf8' });
    validateArchiveListing(listing);
    const verboseListing = execFileSync('tar', ['-tvzf', tarballPath], { encoding: 'utf8' });
    validateArchiveTypes(verboseListing);

    log('Extracting...');
    execFileSync('tar', ['-xzf', tarballPath, '-C', tmpDir], { stdio: 'pipe' });
    validateExtractedTree(tmpDir);

    // Validate the complete payload before modifying an existing installation.
    const extractedBin = path.join(tmpDir, 'bin', 'ollama');
    const extractedLib = path.join(tmpDir, 'lib', 'ollama');

    // Stage the binary, server and all backends together, then activate the
    // complete runtime with a same-filesystem directory rename and rollback.
    installPayloadAtomically(extractedBin, extractedLib);
    log('Installed: ' + OLLAMA_REAL_BIN);
    log('Installed wrapper: ' + OLLAMA_BIN);

    log('');
    log('ollama-termux installed successfully!');
  } catch (e) {
    log(`Installation aborted: ${e.message}`);
    log('The matching GitHub Release and mandatory checksum must both exist.');
    throw e;
  } finally {
    fs.rmSync(tmpDir, { recursive: true, force: true });
  }

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

if (require.main === module) {
  main().catch((e) => {
    console.error('[ollama-termux] Installation failed:', e.message);
    process.exit(1);
  });
}

module.exports = {
  activateStagedInstall,
  normalizedArchiveEntry,
  parseChecksum,
  validateArchiveListing,
  validateArchiveTypes,
  validateExtractedTree,
};
