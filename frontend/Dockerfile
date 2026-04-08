# Build stage
FROM node:18-alpine AS builder
WORKDIR /app

# Copia apenas os arquivos de dependências primeiro (otimiza cache)
COPY package*.json ./
RUN npm install

# Copia TODO o resto (já inclui index.html, vite.config, etc)
# Se o postcss não existir, o Vite usa o padrão interno e não quebra o build
COPY . .

# Build the application
RUN npm run build

# Production stage
FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]