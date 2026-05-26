FROM python:3.12-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
        fonts-noto-cjk \
        fonts-noto \
        fonts-ipaexfont \
        fontconfig \
        libpango-1.0-0 \
        libpangoft2-1.0-0 \
        libharfbuzz0b \
    && rm -rf /var/lib/apt/lists/*

RUN pip install --no-cache-dir weasyprint==62.3 pydyf==0.10.0 flask==3.0.3

WORKDIR /app
COPY server.py /app/server.py

EXPOSE 5000
CMD ["python", "server.py"]
