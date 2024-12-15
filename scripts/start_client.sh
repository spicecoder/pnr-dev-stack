
# # start_client.sh
# #!/bin/bash
# open http://localhost:3000
# echo '{"client_status": {"prompt": "Is web client running?", "response": ["yes"], "tv": "Y"}}' > ./runtime/client_status.json
#!/bin/bash
# scripts/start_client.sh

# Get the project root directory
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Function to write client status
write_status() {
    echo '{
        "client_status": {
            "prompt": "Is web client running?",
            "response": ["yes"],
            "tv": "Y"
        }
    }' > "$PROJECT_ROOT/runtime/client_status.json"
}

# Determine the OS and open the client accordingly
if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS
    open "$PROJECT_ROOT/nodes/client/index.html"
elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
    # Linux
    if command -v xdg-open > /dev/null; then
        xdg-open "$PROJECT_ROOT/nodes/client/index.html"
    else
        echo "Please open $PROJECT_ROOT/nodes/client/index.html in your browser"
    fi
else
    # Windows or other
    echo "Please open $PROJECT_ROOT/nodes/client/index.html in your browser"
fi

# Write status file
write_status

exit 0