FROM node:22-alpine AS build

WORKDIR /src/frontend
COPY frontend/package.json ./
RUN npm install

COPY frontend ./
RUN npm run build

FROM node:22-alpine AS runtime

ENV NODE_ENV=production
WORKDIR /app

COPY --from=build /src/frontend/package.json ./package.json
COPY --from=build /src/frontend/node_modules ./node_modules
COPY --from=build /src/frontend/.next ./.next
COPY --from=build /src/frontend/public ./public

EXPOSE 3000
CMD ["npm", "run", "start"]
