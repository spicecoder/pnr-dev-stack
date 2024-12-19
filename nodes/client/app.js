// nodes/client/app.js
document.addEventListener('DOMContentLoaded', function() {
    fetch('http://pnr_api_server:3000')
        .then(response => response.json())
        .then(data => {
            document.getElementById('status').textContent = data.status;
            
            // Write client ready status
            fetch('/runtime/client_status.json', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    "client_ready": {
                        "prompt": "Is web client ready?",
                        "response": ["yes"],
                        "tv": "Y"
                    }
                })
            });
        })
        .catch(error => {
            document.getElementById('status').textContent = 'Error connecting to API';
            console.error('Error:', error);
        });
});