#!/usr/bin/env node
// Conservative unused-vars preview codemod
// - Very conservative: only touches simple identifier variable declarations and simple identifier function params
// - Conditions to modify a name:
//   * name does NOT already start with `_`
//   * exact word occurrences in file == 1 (only declaration)
//   * file is not in a __tests__ directory and not a test file
// - Action: prefix the identifier with `_` in the declaration/param list
// - Dry-run mode: write a preview file at frontend/codemod_preview.patch and print a summary

const fs = require('fs');
const path = require('path');

const ROOT = path.resolve(__dirname, '../../');
const SRC_DIR = path.join(ROOT, 'src');
const OUTPUT_PATCH = path.join(ROOT, 'codemod_preview_unused_vars.patch');

function walk(dir, fileList = []) {
  const files = fs.readdirSync(dir);
  for (const f of files) {
    const full = path.join(dir, f);
    const stat = fs.statSync(full);
    if (stat.isDirectory()) {
      // skip node_modules and .git
      if (f === 'node_modules' || f === '.git') continue;
      walk(full, fileList);
    } else {
      fileList.push(full);
    }
  }
  return fileList;
}

function isTargetFile(filePath) {
  if (!filePath.startsWith(SRC_DIR)) return false;
  if (/__tests__/.test(filePath)) return false;
  if (/\.test\./.test(filePath)) return false;
  if (/\.spec\./.test(filePath)) return false;
  if (!/\.(ts|tsx|js|jsx)$/.test(filePath)) return false;
  return true;
}

function findSimpleVarDeclarations(content) {
  // matches `const name =`, `let name =`, `var name =` (very conservative)
  const re = /(?:^|[^\w$])(?:const|let|var)\s+([A-Za-z_$][\w$]*)\b/g;
  const matches = [];
  let m;
  while ((m = re.exec(content)) !== null) {
    matches.push({ name: m[1], index: m.index + m[0].indexOf(m[1]) });
  }
  return matches;
}

function _findSimpleFunctionParams(content) {
  // Very conservative param capture for simple param lists: function foo(a, b) or (a, b) =>
  const results = [];
  // function declarations
  const reFunc = /function\s+[A-Za-z_$][\w$]*\s*\(([^)]*)\)/g;
  let m;
  while ((m = reFunc.exec(content)) !== null) {
    const paramList = m[1];
    const startIndex = m.index + m[0].indexOf('(') + 1;
    results.push({ paramList, startIndex });
  }
  // arrow params: (a, b) =>
  const reArrowParens = /\(([^)]*)\)\s*=>/g;
  while ((m = reArrowParens.exec(content)) !== null) {
    const paramList = m[1];
    const startIndex = m.index + m[0].indexOf('(') + 1;
    results.push({ paramList, startIndex });
  }
  // simple single param arrow: a =>
  const reArrowSingle = /(^|[^\w$])([A-Za-z_$][\w$]*)\s*=>/g;
  while ((m = reArrowSingle.exec(content)) !== null) {
    // ensure not part of a larger token by previous char check
    results.push({ paramList: m[2], startIndex: m.index + m[0].indexOf(m[2]) });
  }
  return results;
}

function wordCount(content, name) {
  const re = new RegExp('\\b' + name.replace(/[$\\^\*\+\?\.\(\)\[\]{}|]/g, '\\$&') + '\\b', 'g');
  const m = content.match(re);
  return m ? m.length : 0;
}

function applyPreview() {
  const files = walk(SRC_DIR).filter(isTargetFile);
  const edits = [];

  for (const file of files) {
    let content = fs.readFileSync(file, 'utf8');
    const original = content;
    const fileEdits = [];

    // find simple var declarations
    const decls = findSimpleVarDeclarations(content);
    for (const d of decls) {
      const name = d.name;
      if (name.startsWith('_')) continue;
      const count = wordCount(content, name);
      if (count === 1) {
        // replace first occurrence in the declaration only
        const varRe = new RegExp('((?:const|let|var)\s+)' + name + '\b');
        content = content.replace(varRe, function(_, prefix) {
          return prefix + '_' + name;
        });
        fileEdits.push({ type: 'var-decl-prefix', name, newName: '_' + name });
      }
    }

    // find simple function param lists
    const funcParams = [];
    // Use regexes to extract param lists (conservative)
    const reFuncAll = /function\s+[A-Za-z_$][\w$]*\s*\(([^)]*)\)/g;
    let m;
    while ((m = reFuncAll.exec(content)) !== null) {
      funcParams.push({ list: m[1], index: m.index + m[0].indexOf('(') + 1 });
    }
    const reArrowParensAll = /\(([^)]*)\)\s*=>/g;
    while ((m = reArrowParensAll.exec(content)) !== null) {
      funcParams.push({ list: m[1], index: m.index + m[0].indexOf('(') + 1 });
    }
    const reArrowSingleAll = /(^|[^\w$])([A-Za-z_$][\w$]*)\s*=>/g;
    while ((m = reArrowSingleAll.exec(content)) !== null) {
      funcParams.push({ list: m[2], index: m.index + m[0].indexOf(m[2]) });
    }

    for (const p of funcParams) {
      const list = p.list.trim();
      if (!list) continue;
      // split by commas
      const parts = list.split(',').map(s => s.trim()).filter(Boolean);
      // only process if all parts are simple identifiers (no destructuring, no default)
      const simple = parts.every(part => /^[A-Za-z_$][\w$]*$/.test(part));
      if (!simple) continue;
      // for each param, if occurrences == 1, prefix it
      let newList = parts.slice();
      let changed = false;
      for (let i = 0; i < parts.length; i++) {
        const name = parts[i];
        if (name.startsWith('_')) continue;
        const count = wordCount(content, name);
        if (count === 1) {
          newList[i] = '_' + name;
          changed = true;
          fileEdits.push({ type: 'param-prefix', name, newName: '_' + name });
        }
      }
      if (changed) {
        // replace the first occurrence of the param list (conservative)
        const escaped = list.replace(/[-/\\^$*+?.()|[\]{}]/g, '\\$&');
        const listRe = new RegExp('\\(' + escaped + '\\)\\s*=>');
        if (listRe.test(content)) {
          content = content.replace(listRe, '(' + newList.join(', ') + ') =>');
        } else {
          // try function(...) pattern
          const funcListRe = new RegExp('function\\s+[A-Za-z_$][\\w$]*\\s*\\(' + escaped + '\\)');
          if (funcListRe.test(content)) {
            content = content.replace(funcListRe, (mstr) => mstr.replace('(' + list + ')', '(' + newList.join(', ') + ')'));
          }
        }
      }
    }

    if (fileEdits.length > 0 && content !== original) {
      edits.push({ file, original, content, edits: fileEdits });
    }
  }

  // write a patch preview file with conservative snippet diffs
  const lines = [];
  lines.push('# Codemod preview: prefix simple unused locals/params with _');
  lines.push('# This is a dry-run preview file. No files were modified.');
  lines.push('');
  for (const e of edits) {
    lines.push('File: ' + path.relative(ROOT, e.file));
    lines.push('Edits: ' + e.edits.map(x => x.type + ':' + x.name + '->' + x.newName).join(', '));
    lines.push('--- original (excerpt) ---');
    // include up to 12 lines around each change by finding the changed token
    const diffExcerpt = makeExcerpt(e.original, e.content);
    lines.push(diffExcerpt);
    lines.push('');
  }

  fs.writeFileSync(OUTPUT_PATCH, lines.join('\n'));

  // print concise summary
  console.log('Codemod dry-run complete. Preview written to:', OUTPUT_PATCH);
  console.log('Files with proposed edits:', edits.length);
  for (const e of edits) {
    console.log('-', path.relative(ROOT, e.file), '(', e.edits.length, 'changes)');
  }
  if (edits.length === 0) console.log('No safe edits found by the conservative preview.');
}

function makeExcerpt(orig, modified) {
  // show original and modified lines side-by-side for the first changed region
  const oLines = orig.split(/\r?\n/);
  const mLines = modified.split(/\r?\n/);
  let firstDiff = -1;
  const len = Math.max(oLines.length, mLines.length);
  for (let i = 0; i < len; i++) {
    if (oLines[i] !== mLines[i]) { firstDiff = i; break; }
  }
  if (firstDiff === -1) return '[no visible diff in excerpt]';
  const start = Math.max(0, firstDiff - 3);
  const end = Math.min(len, firstDiff + 6);
  const excerpt = [];
  excerpt.push('@@ context lines ' + (start + 1) + '-' + end + ' @@');
  for (let i = start; i < end; i++) {
    const o = oLines[i] || '';
    const m = mLines[i] || '';
    if (o === m) {
      excerpt.push('  ' + o);
    } else {
      excerpt.push('- ' + o);
      excerpt.push('+ ' + m);
    }
  }
  return excerpt.join('\n');
}

// Run
try {
  applyPreview();
} catch (err) {
  console.error('Error running codemod preview:', err);
  process.exit(2);
}
