FROM python:3.12-slim
RUN apt-get update && apt-get install -y --no-install-recommends curl build-essential && rm -rf /var/lib/apt/lists/*
WORKDIR /app
ENV PYTHONUNBUFFERED=1 PIP_NO_CACHE_DIR=1
COPY pyproject.toml ./
RUN pip install uv && uv pip install --system .
COPY . .
EXPOSE 50052
ENTRYPOINT ["python", "main.py"]
