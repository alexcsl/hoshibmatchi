FROM python:3.9-slim
WORKDIR /app

# Install system dependencies for building Python packages
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

# The context is the root, so use the full path
COPY backend/ai-service/requirements.txt .

# Install Python dependencies with increased timeout
RUN pip install --no-cache-dir --upgrade pip && \
    pip install --no-cache-dir -r requirements.txt

# Copy the ai-service source code
COPY backend/ai-service/ .

# Expose port 9008
EXPOSE 9008

CMD ["python3", "main.py"]