# nodes/client/Dockerfile
FROM nginx:alpine

# Copy static files
COPY index.html /usr/share/nginx/html/
COPY app.js /usr/share/nginx/html/

# Nginx runs on port 80 by default
EXPOSE 80

# Nginx's default CMD will start the server