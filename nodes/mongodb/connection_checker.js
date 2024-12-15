// nodes/mongodb/connection_checker.js
const { MongoClient } = require('mongodb');
const fs = require('fs');
const path = require('path');

// Load configuration
const config = JSON.parse(
    fs.readFileSync(path.join(__dirname, '../../config/domain.json'), 'utf8')
);

const { url, options } = config.env.mongodb;
const maxAttempts = 30;  // Will try for 30 seconds
let attempts = 0;

async function checkConnection() {
    try {
        console.log(`Attempting to connect to MongoDB at ${url}`);
        const client = await MongoClient.connect(url, {
            ...options,
            serverSelectionTimeoutMS: 1000, // Wait 1 second before timing out
            useNewUrlParser: true,
            useUnifiedTopology: true
        });
        
        // Verify connection by performing a simple operation
        await client.db().admin().ping();
        await client.close();
        
        // Write success status
        const status = {
            "mongodb_status": {
                "prompt": "Is MongoDB connected?",
                "response": ["yes"],
                "tv": "Y"
            }
        };
        
        fs.writeFileSync(
            path.join(__dirname, '../../runtime/mongodb_status.json'),
            JSON.stringify(status, null, 2)
        );
        
        console.log('Successfully connected to MongoDB');
        process.exit(0);
    } catch (err) {
        attempts++;
        if (attempts >= maxAttempts) {
            console.error(`Failed to connect to MongoDB after ${maxAttempts} attempts`);
            console.error(`Last error: ${err.message}`);
            process.exit(1);
        }
        console.log(`Attempt ${attempts}: MongoDB not ready yet, retrying... (${err.message})`);
        setTimeout(checkConnection, 1000);
    }
}

// Start checking
checkConnection();