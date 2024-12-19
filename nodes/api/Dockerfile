# nodes/api/Dockerfile
FROM node:16-alpine

WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm install

# Copy source code
COPY server.js ./

# Copy config directory at runtime (mounted)
# COPY config directory will be mounted via the bind mount

# Expose API port
EXPOSE 3000

# Start the API server
CMD ["node", "server.js"]