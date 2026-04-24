# NVIDIA CUDA base for vLLM. CPU fallback path works on any host.
FROM nvidia/cuda:12.4.1-runtime-ubuntu22.04 AS base
RUN apt-get update && apt-get install -y --no-install-recommends \
        python3.12 python3.12-venv python3-pip git && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app
ENV PYTHONUNBUFFERED=1
ENV PIP_NO_CACHE_DIR=1

COPY pyproject.toml ./
RUN pip install uv && uv pip install --system .

COPY . .

EXPOSE 50051 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=60s --retries=3 \
    CMD curl -fsSL http://localhost:8080/healthz || exit 1

ENTRYPOINT ["python3.12", "main.py"]
