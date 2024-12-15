// nodes/client/app.js
async function checkHealth() {
    const apiStatus = document.getElementById('api-status');
    const mongoStatus = document.getElementById('mongodb-status');
    
    try {
        const response = await fetch('http://localhost:3000/api/health');
        const data = await response.json();
        
        if (data.status === 'ok') {
            apiStatus.textContent = 'API Server: Connected';
            apiStatus.className = 'status success';
            
            if (data.mongodb === 'connected') {
                mongoStatus.textContent = 'MongoDB: Connected';
                mongoStatus.className = 'status success';
            }
        }
    } catch (err) {
        apiStatus.textContent = `API Server: Error - ${err.message}`;
        apiStatus.className = 'status error';
        mongoStatus.textContent = 'MongoDB: Status Unknown';
        mongoStatus.className = 'status error';
    }
}

// Check health immediately and every 5 seconds
checkHealth();
setInterval(checkHealth, 5000);