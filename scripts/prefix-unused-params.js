#!/usr/bin/env node
// Simple codemod: prefix unused function params with '_' across TSX/TS files under frontend/src
// WARNING: heuristics only. Back up before running.
const fs = require('fs');
const path = require('path');
const ts = require('typescript');

function walk(dir, cb) {
  fs.readdirSync(dir, { withFileTypes: true }).forEach(d => {
    const full = path.join(dir, d.name);
    if (d.isDirectory()) walk(full, cb);
    else if (/\.(ts|tsx|js|jsx)$/.test(d.name)) cb(full);
  });
}

function processFile(filePath) {
  const src = fs.readFileSync(filePath, 'utf8');
  const sourceFile = ts.createSourceFile(filePath, src, ts.ScriptTarget.Latest, true);
  const edits = [];

  function visit(node) {
    if ((ts.isFunctionDeclaration(node) || ts.isFunctionExpression(node) || ts.isArrowFunction(node) || ts.isMethodDeclaration(node)) && node.parameters) {
      node.parameters.forEach(param => {
        const name = param.name;
        if (ts.isIdentifier(name)) {
          const paramName = name.text;
          if (!paramName.startsWith('_')) {
            // check usage of paramName in the function body
            const body = node.body;
            if (body) {
              let used = false;
              function findUsage(n) {
                if (used) return;
                if (ts.isIdentifier(n) && n.text === paramName) used = true;
                ts.forEachChild(n, findUsage);
              }
              findUsage(body);
              if (!used) {
                // schedule edit: replace paramName with _ + paramName
                const start = name.getStart(sourceFile);
                const end = name.getEnd();
                edits.push({ start, end, text: '_' + paramName });
              }
            }
          }
        }
      });
    }
    ts.forEachChild(node, visit);
  }
  visit(sourceFile);
  if (edits.length === 0) return false;
  // apply edits from last to first
  edits.sort((a,b)=>b.start - a.start);
  let out = src;
  edits.forEach(e => {
    out = out.slice(0, e.start) + e.text + out.slice(e.end);
  });
  fs.writeFileSync(filePath, out, 'utf8');
  return true;
}

const root = path.join(__dirname, '..', 'frontend', 'src');
let changed = 0;
walk(root, (file) => {
  try {
    if (processFile(file)) changed++;
  } catch (e) {
    console.error('Failed processing', file, e.message);
  }
});
console.log('Files changed:', changed);
