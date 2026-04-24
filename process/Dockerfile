FROM python:3.12-slim AS base
RUN apt-get update && apt-get install -y --no-install-recommends \
        build-essential libxml2-dev libxslt1-dev curl && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app
ENV PYTHONUNBUFFERED=1 PIP_NO_CACHE_DIR=1
COPY pyproject.toml ./
RUN pip install uv && uv pip install --system .

COPY . .
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s CMD curl -fsSL http://localhost:8080/healthz || exit 1
ENTRYPOINT ["python", "main.py"]
