FROM node:22-alpine

WORKDIR /app

# Install dependencies first for better layer caching.
COPY package.json ./
RUN npm install

COPY . .

EXPOSE 5173

# Vite dev server with hot reload.
CMD ["npm", "run", "dev"]
