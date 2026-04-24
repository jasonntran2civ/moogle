FROM python:3.12-slim
RUN apt-get update && apt-get install -y --no-install-recommends curl && rm -rf /var/lib/apt/lists/*
WORKDIR /app
ENV PYTHONUNBUFFERED=1 PIP_NO_CACHE_DIR=1
COPY pyproject.toml ./
RUN pip install uv && uv pip install --system .
COPY . .
COPY ../prompts/agent_system.md /prompts/agent_system.md
EXPOSE 8081
HEALTHCHECK --interval=30s --timeout=5s CMD curl -fsSL http://localhost:8081/healthz || exit 1
CMD ["python", "main.py"]
