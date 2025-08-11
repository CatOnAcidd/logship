# Build
FROM node:20-alpine AS build
WORKDIR /app

# Only package.json is required if no lockfile is committed
COPY package.json ./
RUN npm install

# Now copy the rest and build
COPY index.html vite.config.ts tsconfig.json ./
COPY src ./src
RUN npm run build

# Serve
FROM nginx:1.27-alpine
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80
