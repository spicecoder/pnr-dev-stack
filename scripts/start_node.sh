#!/bin/bash
# scripts/start_nodejs.sh

# Get the project root directory (parent of scripts directory)
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Run the Node.js server
node "$PROJECT_ROOT/nodes/api/server.js"

# Exit with the server's exit code
exit $?