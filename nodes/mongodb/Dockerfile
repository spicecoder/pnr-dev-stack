FROM node:16-alpine

WORKDIR /app

# Copy package files
COPY package.json ./
COPY connection_checker.js ./

# Install dependencies
RUN npm install mongodb

# Run the checker
CMD ["node", "connection_checker.js"]