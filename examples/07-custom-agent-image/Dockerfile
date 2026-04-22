FROM ghcr.io/komputer-ai/komputer-agent:latest

USER root

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
      postgresql-client \
      awscli \
      ffmpeg && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

RUN pip install --no-cache-dir pandas numpy boto3 psycopg2-binary

USER komputer
