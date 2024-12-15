// nodes/api/server.js
const express = require('express');
const { MongoClient } = require('mongodb');
const fs = require('fs');
const path = require('path');

// Load configuration
const config = JSON.parse(
    fs.readFileSync(path.join(__dirname, '../../config/domain.json'), 'utf8')
);

const app = express();
const { mongodb, api } = config.env;

// Add CORS middleware
app.use((req, res, next) => {
    res.header('Access-Control-Allow-Origin', '*');
    res.header('Access-Control-Allow-Methods', 'GET, POST, OPTIONS');
    res.header('Access-Control-Allow-Headers', 'Content-Type');
    if (req.method === 'OPTIONS') {
        return res.sendStatus(200);
    }
    next();
});

async function startServer() {
    try {
        // Connect to MongoDB
        const client = await MongoClient.connect(mongodb.url, {
            ...mongodb.options,
            useNewUrlParser: true,
            useUnifiedTopology: true
        });
        
        console.log('Connected to MongoDB');
        
        // Basic health check endpoint
        app.get('/api/health', async (req, res) => {
            try {
                await client.db().admin().ping();
                res.json({ status: 'ok', mongodb: 'connected' });
            } catch (err) {
                res.status(500).json({ status: 'error', message: err.message });
            }
        });
        
        // Start server
        app.listen(api.port, api.host, () => {
            console.log(`API server running on ${api.host}:${api.port}`);
            
            // Write status to runtime file
            const status = {
                "nodejs_status": {
                    "prompt": "Is NodeJS API ready?",
                    "response": ["yes"],
                    "tv": "Y"
                }
            };
            
            fs.writeFileSync(
                path.join(__dirname, '../../runtime/nodejs_status.json'),
                JSON.stringify(status, null, 2)
            );
        });
        
    } catch (err) {
        console.error('Failed to start server:', err);
        process.exit(1);
    }
}

startServer();