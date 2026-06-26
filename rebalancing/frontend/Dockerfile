# Multi-stage build for React frontend
FROM node:18-alpine AS builder

WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm ci --legacy-peer-deps

# Build arguments for environment
ARG VITE_API_URL=http://localhost:8090
ARG VITE_GRAPHQL_URL=http://localhost:8080/v1/graphql

# Copy source code
COPY . .

# Build the application
RUN npm run build

# Production stage - serve with lightweight server
FROM node:18-alpine

WORKDIR /app

# Install serve to run the built app
RUN npm install -g serve

# Copy built assets from builder
COPY --from=builder /app/dist ./dist

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:3000/ || exit 1

# Expose port
EXPOSE 3000

# Set environment
ENV NODE_ENV=production

# Start the server
CMD ["serve", "-s", "dist", "-l", "3000"]