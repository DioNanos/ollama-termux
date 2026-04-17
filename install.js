#!/usr/bin/env node
/**
 * ollama-termux installer
 * Handles installation on Termux/Android ARM64
 */

const fs = require('fs');
const path = require('path');

const TERMUX_PREFIX = '/data/data/com.termux/files/usr';
const OLLAMA_BIN = path.join(TERMUX_PREFIX, 'bin', 'ollama');

function log(msg) {
  console.log(`[ollama-termux] ${msg}`);
}

function isTermux() {
  return fs.existsSync('/data/data/com.termux/files/usr');
}

function main() {
  if (!isTermux()) {
    console.log('[ollama-termux] Not running on Termux. This package is for Termux only.');
    console.log('For installation instructions, see:');
    console.log('https://github.com/DioNanos/ollama-termux#quick-start');
    return;
  }

  log('Installing ollama-termux...');
  log('');
  log('To build from source:');
  log('  cd ~/Dev/60_toolchains/ollama-termux');
  log('  go build -o ollama-termux -ldflags="-s -w" -trimpath .');
  log('');
  log('Then replace the binary:');
  log(`  cp ollama-termux ${OLLAMA_BIN}`);
  log(`  chmod +x ${OLLAMA_BIN}`);
  log('');
  log('For optimized ggml .so files, see BUILDING.md');
}

main();
