# frontend/hoshi-vue/Dockerfile

# Stage 1: Build the Vue application (This stage remains the same)
FROM node:lts-alpine as build-stage
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
RUN npm run build

# Stage 2: Serve the static files with Nginx
FROM nginx:stable-alpine as production-stage

# Copy the built files from the build stage
COPY --from=build-stage /app/dist /usr/share/nginx/html

# ---- THIS IS THE NEW LINE ----
# Remove the default Nginx config and copy our custom one.
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Expose port 80
EXPOSE 80

# Start Nginx
CMD ["nginx", "-g", "daemon off;"]