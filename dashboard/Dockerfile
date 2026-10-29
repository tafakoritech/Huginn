# syntax=docker/dockerfile:1

# ---- Stage 1: build ----
FROM node:20-alpine AS build
WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

COPY . .
# Vite inlines VITE_* variables at build time. For container deployments the
# recommended value is a same-origin path ("/api/v1") proxied to the backend
# by nginx, so no backend URL is embedded in the bundle.
ARG VITE_API_BASE_URL=/api/v1
ARG VITE_APP_NAME=Huginn
ARG VITE_DEFAULT_LOCALE=fa
ARG VITE_DEFAULT_THEME=light
ENV VITE_API_BASE_URL=$VITE_API_BASE_URL \
    VITE_APP_NAME=$VITE_APP_NAME \
    VITE_DEFAULT_LOCALE=$VITE_DEFAULT_LOCALE \
    VITE_DEFAULT_THEME=$VITE_DEFAULT_THEME \
    VITE_USE_MOCK_API=false

RUN npm run build

# ---- Stage 2: static server ----
FROM nginx:1.27-alpine
# Backend API proxy (Huginn Go API on :8080). Override via -e BACKEND_UPSTREAM.
# nginx envsubst only substitutes $VAR (no ${VAR:-default} forms), so these
# always carry defaults here.
ENV BACKEND_UPSTREAM=http://backend:8080 \
    DASHBOARD_PORT=80

COPY nginx.conf /etc/nginx/templates/default.conf.template
COPY --from=build /app/dist /usr/share/nginx/html

EXPOSE 80

# nginx's official entrypoint renders the .template with envsubst.
CMD ["nginx", "-g", "daemon off;"]
