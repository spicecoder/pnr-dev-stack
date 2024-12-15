#!/bin/bash
# scripts/check_mongodb.sh

# Get the project root directory (parent of scripts directory)
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Run the connection checker
node "$PROJECT_ROOT/nodes/mongodb/connection_checker.js"

# Exit with the checker's exit code
exit $?