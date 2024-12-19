// connection_checker.js
const { MongoClient } = require('mongodb');
const fs = require('fs');
const path = require('path');

// Use container name as hostname for MongoDB
const mongoUrl = "mongodb://pnr_mongodb:27017";  // Using container name instead of localhost
console.log(`Using MongoDB URL: ${mongoUrl}`);

async function checkConnection() {
    try {
        console.log(`Attempting to connect to MongoDB at ${mongoUrl}`);
        const client = await MongoClient.connect(mongoUrl, {
            serverSelectionTimeoutMS: 1000,
            useNewUrlParser: true,
            useUnifiedTopology: true
        });
        
        await client.db().admin().ping();
        await client.close();
        
        // Write success status
        const status = {
            "mongodb_connected": {
                "prompt": "Is MongoDB connected?",
                "response": ["yes"],
                "tv": "Y"
            }
        };
        
        fs.writeFileSync(
            '/runtime/mongodb_status.json',
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

checkConnection();