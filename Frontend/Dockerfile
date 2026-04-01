FROM node:20-alpine AS build
WORKDIR /app

ARG VITE_BACKEND_BASE_URL=http://localhost:8180
ARG VITE_API_FLAVOR=go
ENV VITE_BACKEND_BASE_URL=$VITE_BACKEND_BASE_URL
ENV VITE_API_FLAVOR=$VITE_API_FLAVOR

COPY package.json package-lock.json* ./
RUN npm ci

COPY . .
RUN npm run build

FROM nginx:1.27-alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html

EXPOSE 80
