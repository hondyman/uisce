FROM node:18-alpine

WORKDIR /app

# Copy dependency definitions
COPY resolver/package.json ./resolver/
COPY resolver/tsconfig.json ./resolver/
# Dry-run script has no separate package.json, we'll assume it uses resolver's deps or global
# For robustness, let's copy root package.json if it exists or just rely on resolver since dryrun imports from it conceptually
# Uses resolver package.json for standard dependencies
COPY resolver/package.json ./package.json

# Install dependencies
RUN npm install

# Copy source
COPY ops/dryrun/dryrun.ts ./src/dryrun.ts
# Need ts-node to run ts file
RUN npm install -g ts-node typescript

# Runtime envs
ENV NODE_ENV=production

CMD ["ts-node", "src/dryrun.ts"]
