# errex webhook -> email relay via useSend or SendGrid (standard library only, no dependencies).
FROM python:3.13-alpine@sha256:bf12ecc0bea3f171d55145dd02c78b5eeb09f30ee2ac9cc2b4c9e4e8f5e30741
ENV PYTHONDONTWRITEBYTECODE=1 PYTHONUNBUFFERED=1
WORKDIR /app
COPY relay.py .
USER nobody
EXPOSE 8080
# busybox wget, not Python: the image ships no .pyc files, so a Python probe recompiles
# urllib/ssl on every run and takes longer than the timeout under the 0.25 CPU limit.
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
	CMD ["wget", "-q", "-T", "2", "-O", "/dev/null", "http://127.0.0.1:8080/healthz"]
CMD ["python", "relay.py"]
