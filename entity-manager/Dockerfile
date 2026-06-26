FROM node:20-alpine

WORKDIR /app/entity-manager

# Copy dependencies
COPY entity-manager/package*.json ./
COPY entity-manager/tsconfig.json ./

# Copy shared internal packages to where imports expect them (../../internal -> ./internal)
COPY internal/pkg ./internal/pkg

# Install deps
RUN npm install && apk add --no-cache curl

# Copy source
COPY entity-manager/src ./src

# Create a build-specific tsconfig to allow compiling ../internal
# We set rootDir to . so it can include both src and internal
# This changes output structure to dist/src and dist/internal
RUN echo '{ \
  "extends": "./tsconfig.json", \
  "compilerOptions": { \
    "rootDir": ".", \
    "outDir": "./dist", \
    "strict": false, \
    "noImplicitAny": false, \
    "skipLibCheck": true \
  }, \
  "include": [ \
    "src/**/*", \
    "internal/**/*" \
  ] \
}' > tsconfig.build.json

# Build using custom config
RUN npx tsc -p tsconfig.build.json

EXPOSE 8080

# Update start command because dist structure changed
CMD ["node", "dist/src/server.js"]
