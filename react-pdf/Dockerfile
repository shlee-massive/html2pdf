FROM node:24-bookworm-slim

# OS 폰트 패키지는 fallback 안전망. 실제 @react-pdf 는 Font.register() 로
# 명시 등록된 TTF 만 사용 (server.mjs 참조). CJK 글리프 누락 방지용으로만 설치.
RUN apt-get update && apt-get install -y --no-install-recommends \
        fonts-noto-cjk \
        fonts-noto \
        fonts-ipaexfont \
        fontconfig \
        ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# 의존성만 먼저 설치 (캐시 효율)
COPY package.json /app/package.json
RUN npm install --omit=dev --no-audit --no-fund

COPY server.mjs /app/server.mjs
COPY invoice-doc.mjs /app/invoice-doc.mjs
COPY money.mjs /app/money.mjs
COPY strings.mjs /app/strings.mjs

ENV NODE_ENV=production
EXPOSE 5002

CMD ["node", "server.mjs"]
