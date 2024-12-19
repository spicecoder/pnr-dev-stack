// nodes/api/server.js
const express = require('express');
const { MongoClient } = require('mongodb');
const fs = require('fs');
const path = require('path');

const app = express();
const port = 3000;

// MongoDB connection URL (using container name)
const mongoUrl = "mongodb://pnr_mongodb:27017";

app.get('/', (req, res) => {
    res.json({ status: 'API Server Running' });
});

async function startServer() {
    try {
        // Test MongoDB connection
        const client = await MongoClient.connect(mongoUrl);
        await client.db().admin().ping();
        await client.close();

        // Start Express server
        app.listen(port, '0.0.0.0', () => {
            console.log(`API Server listening on port ${port}`);
            
            // Write API ready status
            const status = {
                "api_ready": {
                    "prompt": "Is API server ready?",
                    "response": ["yes"],
                    "tv": "Y"
                }
            };
            
            fs.writeFileSync(
                '/runtime/api_status.json',
                JSON.stringify(status, null, 2)
            );
        });
    } catch (err) {
        console.error('Failed to start server:', err);
        process.exit(1);
    }
}

startServer();