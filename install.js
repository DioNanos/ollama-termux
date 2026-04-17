#!/usr/bin/env node
/**
 * ollama-termux installer for Termux/Android
 *
 * This script:
 * 1. Clones ollama-termux if not present
 * 2. Builds the Go binary
 * 3. Installs to Termux prefix
 */

const { execSync, spawn } = require('child_process');
const fs = require('fs');
const path = require('path');

const TERMUX_PREFIX = '/data/data/com.termux/files/usr';
const OLLAMA_BIN = path.join(TERMUX_PREFIX, 'bin', 'ollama');
const OLLAMA_LIB = path.join(TERMUX_PREFIX, 'lib', 'ollama');
const INSTALL_DIR = path.join(process.env.HOME || '/data/data/com.termux/files/home', 'ollama-termux');

function log(msg) {
  console.log(`[ollama-termux] ${msg}`);
}

function isTermux() {
  return fs.existsSync('/data/data/com.termux/files/usr');
}

function isGitInstalled() {
  try {
    execSync('git --version', { stdio: 'ignore' });
    return true;
  } catch {
    return false;
  }
}

function isGoInstalled() {
  try {
    execSync('go version', { stdio: 'ignore' });
    return true;
  } catch {
    return false;
  }
}

function run(cmd, args = []) {
  log(`Running: ${cmd} ${args.join(' ')}`);
  try {
    const result = execSync(`${cmd} ${args.join(' ')}`, {
      cwd: INSTALL_DIR,
      stdio: 'inherit',
      timeout: 600000 // 10 min for build
    });
    return true;
  } catch (e) {
    log(`Error running: ${cmd}`);
    return false;
  }
}

function main() {
  if (!isTermux()) {
    console.log('[ollama-termux] This package is for Termux only.');
    console.log('');
    console.log('For manual installation:');
    console.log('  git clone https://github.com/DioNanos/ollama-termux.git');
    console.log('  cd ollama-termux');
    console.log('  go build -o ollama-termux -ldflags="-s -w" -trimpath .');
    console.log('  cp ollama-termux /data/data/com.termux/files/usr/bin/');
    console.log('  chmod +x /data/data/com.termux/files/usr/bin/ollama');
    console.log('');
    console.log('See: https://github.com/DioNanos/ollama-termux#quick-start');
    return;
  }

  log('Installing ollama-termux on Termux...');
  log('');

  // Check prerequisites
  if (!isGitInstalled()) {
    log('git not found. Installing...');
    execSync('pkg install git -y', { stdio: 'inherit' });
  }

  if (!isGoInstalled()) {
    log('golang not found. Installing...');
    execSync('pkg install golang -y', { stdio: 'inherit' });
  }

  // Clone or update repo
  if (!fs.existsSync(INSTALL_DIR)) {
    log(`Cloning ollama-termux to ${INSTALL_DIR}...`);
    try {
      execSync(`git clone https://github.com/DioNanos/ollama-termux.git "${INSTALL_DIR}"`, {
        stdio: 'inherit',
        timeout: 60000
      });
    } catch (e) {
      log('Failed to clone repository');
      return;
    }
  } else {
    log('Updating existing installation...');
    try {
      execSync('git pull origin main', { cwd: INSTALL_DIR, stdio: 'inherit' });
    } catch (e) {
      log('Note: Could not pull updates (may be on detached HEAD)');
    }
  }

  // Build
  log('');
  log('Building ollama-termux (this may take a while)...');
  const built = run('go', ['build', '-o', 'ollama-termux', '-ldflags=-s -w', '-trimpath', '.']);

  if (!built) {
    log('Build failed. Try manually:');
    log(`  cd ${INSTALL_DIR}`);
    log('  go build -o ollama-termux -ldflags="-s -w" -trimpath .');
    return;
  }

  // Backup existing
  if (fs.existsSync(OLLAMA_BIN)) {
    const backup = `${OLLAMA_BIN}.backup`;
    log(`Backing up existing ollama to ${backup}`);
    fs.copyFileSync(OLLAMA_BIN, backup);
  }

  // Install
  log('Installing binary...');
  fs.copyFileSync(path.join(INSTALL_DIR, 'ollama-termux'), OLLAMA_BIN);
  fs.chmodSync(OLLAMA_BIN, 0o755);

  log('');
  log('✓ ollama-termux installed successfully!');
  log('');
  log('To use:');
  log('  ollama serve &');
  log('  ollama pull <model>');
  log('');
  log('For optimized ggml .so files, see BUILDING.md');
}

main();
