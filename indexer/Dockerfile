# syntax=docker/dockerfile:1.6

FROM python:3.12-slim

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    WORKSPACE_DIR=/workspace \
    INDEX_DIR=/data/index

WORKDIR /app

COPY requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt

COPY index_repo.py ./

ENTRYPOINT ["python", "/app/index_repo.py"]
