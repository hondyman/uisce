#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const semver = require('semver');

const SCHEMA_PATH = path.join(__dirname, '..', 'schemas', 'upgrade-artifacts-data.schema.json');
const BUMP_TYPE = process.env.BUMP_TYPE || 'patch';
const CHANGE_DESC = process.env.CHANGE_DESC || 'Schema updated';

function bumpSchemaVersion() {
  // Read current schema
  const schemaContent = fs.readFileSync(SCHEMA_PATH, 'utf8');
  const schema = JSON.parse(schemaContent);

  // Validate current version
  if (!semver.valid(schema.schema_version)) {
    throw new Error(`Invalid current schema_version: ${schema.schema_version}`);
  }

  // Bump version
  const newVersion = semver.inc(schema.schema_version, BUMP_TYPE);
  if (!newVersion) {
    throw new Error(`Failed to bump version ${schema.schema_version} with type ${BUMP_TYPE}`);
  }

  // Update schema_version
  schema.schema_version = newVersion;

  // Initialize changelog if it doesn't exist
  if (!schema.changelog) {
    schema.changelog = [];
  }

  // Add new changelog entry
  const newEntry = {
    version: newVersion,
    date: new Date().toISOString(),
    description: CHANGE_DESC
  };

  // Prepend to changelog (most recent first)
  schema.changelog.unshift(newEntry);

  // Write updated schema
  fs.writeFileSync(SCHEMA_PATH, JSON.stringify(schema, null, 2) + '\n');

  console.log(`✅ Bumped schema_version from ${schema.schema_version} to ${newVersion}`);
  console.log(`📝 Added changelog entry: ${CHANGE_DESC}`);

  return newVersion;
}

// Run if called directly
if (require.main === module) {
  try {
    bumpSchemaVersion();
  } catch (error) {
    console.error('❌ Error:', error.message);
    process.exit(1);
  }
}

module.exports = { bumpSchemaVersion };
