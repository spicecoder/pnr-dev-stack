#!/bin/bash
# mongod --dbpath ./data/db --logpath ./data/mongod.log --fork
echo '{"mongodb_status": {"prompt": "Is MongoDB running?", "response": ["yes"], "tv": "Y"}}' > ./runtime/mongodb_status.json
