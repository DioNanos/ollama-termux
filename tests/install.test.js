const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const test = require('node:test');

const {
  activateStagedInstall,
  normalizedArchiveEntry,
  parseChecksum,
  validateArchiveListing,
  validateArchiveTypes,
  validateExtractedTree,
} = require('../install.js');

const archiveListing = [
  './',
  './bin/',
  './bin/ollama',
  './lib/',
  './lib/ollama/',
  './lib/ollama/llama-server',
  './lib/ollama/libggml-base.so',
  './lib/ollama/libggml-cpu-android_armv8_2.so',
  './lib/ollama/vulkan/',
  './lib/ollama/vulkan/libggml-vulkan.so',
  './install.js',
].join('\n');

test('parseChecksum requires an exact SHA256 and artifact name', () => {
  const sha = 'a'.repeat(64);
  assert.equal(parseChecksum(`${sha}  artifact.tar.gz\n`, 'artifact.tar.gz'), sha);
  assert.throws(
    () => parseChecksum(`${sha}  other.tar.gz\n`, 'artifact.tar.gz'),
    /checksum names other\.tar\.gz/,
  );
  assert.throws(() => parseChecksum('not-a-checksum', 'artifact.tar.gz'), /invalid SHA256/);
});

test('archive listing accepts only the expected payload roots', () => {
  assert.doesNotThrow(() => validateArchiveListing(archiveListing));
  assert.throws(
    () => validateArchiveListing(`${archiveListing}\n./../outside`),
    /escapes extraction root/,
  );
  assert.throws(
    () => validateArchiveListing(`${archiveListing}\n./private/internal.txt`),
    /unexpected archive path/,
  );
  assert.throws(
    () => validateArchiveListing(archiveListing.replace('./bin/ollama\n', '')),
    /missing bin\/ollama/,
  );
});

test('archive path normalization rejects absolute and traversal paths', () => {
  assert.equal(normalizedArchiveEntry('./lib/ollama/llama-server'), 'lib/ollama/llama-server');
  assert.throws(() => normalizedArchiveEntry('/etc/passwd'), /unsafe archive path/);
  assert.throws(() => normalizedArchiveEntry('./../../etc/passwd'), /escapes extraction root/);
});

test('archive metadata rejects links, special entries and privileged modes', () => {
  const safe = [
    'drwxr-xr-x user/group 0 2026-07-18 12:00 ./bin/',
    '-rwxr-xr-x user/group 1 2026-07-18 12:00 ./bin/ollama',
  ].join('\n');
  assert.doesNotThrow(() => validateArchiveTypes(safe));
  assert.throws(
    () => validateArchiveTypes('lrwxrwxrwx user/group 0 2026-07-18 12:00 ./bin/ollama -> /tmp/x'),
    /link or special archive entry/,
  );
  assert.throws(
    () => validateArchiveTypes('-rwsr-xr-x user/group 1 2026-07-18 12:00 ./bin/ollama'),
    /privileged archive mode/,
  );
});

test('extracted payload rejects links and special entries', () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'ollama-termux-installer-test-'));
  try {
    fs.mkdirSync(path.join(root, 'bin'), { recursive: true });
    fs.mkdirSync(path.join(root, 'lib', 'ollama'), { recursive: true });
    fs.writeFileSync(path.join(root, 'bin', 'ollama'), 'binary');
    fs.writeFileSync(path.join(root, 'lib', 'ollama', 'llama-server'), 'binary');
    fs.chmodSync(path.join(root, 'bin', 'ollama'), 0o755);
    fs.chmodSync(path.join(root, 'lib', 'ollama', 'llama-server'), 0o755);
    fs.writeFileSync(path.join(root, 'lib', 'ollama', 'libggml-base.so'), 'library');
    fs.writeFileSync(path.join(root, 'lib', 'ollama', 'libggml-cpu-armv8.so'), 'library');

    assert.doesNotThrow(() => validateExtractedTree(root));
    fs.symlinkSync('/etc/passwd', path.join(root, 'lib', 'ollama', 'escape.so'));
    assert.throws(() => validateExtractedTree(root), /unsafe extracted entry/);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test('runtime activation is atomic and rolls back if wrapper replacement fails', () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'ollama-termux-activation-test-'));
  try {
    const liveLib = path.join(root, 'live-lib');
    const liveWrapper = path.join(root, 'ollama');
    const stagedLib = path.join(root, 'staged-lib');
    const stagedWrapper = path.join(root, 'staged-wrapper');
    fs.mkdirSync(liveLib);
    fs.mkdirSync(stagedLib);
    fs.writeFileSync(path.join(liveLib, 'runtime'), 'old');
    fs.writeFileSync(path.join(stagedLib, 'runtime'), 'new');
    fs.writeFileSync(liveWrapper, 'old wrapper');
    fs.writeFileSync(stagedWrapper, 'new wrapper');

    activateStagedInstall(stagedLib, stagedWrapper, liveLib, liveWrapper);
    assert.equal(fs.readFileSync(path.join(liveLib, 'runtime'), 'utf8'), 'new');
    assert.equal(fs.readFileSync(liveWrapper, 'utf8'), 'new wrapper');
    assert.equal(fs.readdirSync(root).some((name) => name.startsWith('live-lib.backup-')), false);

    const rollbackLive = path.join(root, 'rollback-live');
    const rollbackStage = path.join(root, 'rollback-stage');
    const rollbackWrapper = path.join(root, 'rollback-wrapper');
    const invalidWrapperTarget = path.join(root, 'wrapper-directory');
    fs.mkdirSync(rollbackLive);
    fs.mkdirSync(rollbackStage);
    fs.mkdirSync(invalidWrapperTarget);
    fs.writeFileSync(path.join(rollbackLive, 'runtime'), 'preserved');
    fs.writeFileSync(path.join(rollbackStage, 'runtime'), 'rejected');
    fs.writeFileSync(rollbackWrapper, 'wrapper');

    assert.throws(
      () => activateStagedInstall(
        rollbackStage,
        rollbackWrapper,
        rollbackLive,
        invalidWrapperTarget,
      ),
      /EISDIR|ENOTDIR|directory/i,
    );
    assert.equal(fs.readFileSync(path.join(rollbackLive, 'runtime'), 'utf8'), 'preserved');
    assert.equal(fs.existsSync(path.join(rollbackLive, 'rejected')), false);
    assert.equal(
      fs.readdirSync(root).some((name) => name.startsWith('rollback-live.backup-')),
      false,
    );
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});
