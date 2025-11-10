FROM python:3.9-slim
WORKDIR /app

# The context is the root, so use the full path
COPY backend/ai-service/requirements.txt .

RUN pip install --no-cache-dir -r requirements.txt

# Copy the ai-service source code
COPY backend/ai-service/ .

# This port is 8000, but it will conflict with api-gateway.
# Let's change it to 9008 as per your main.py
CMD ["python3", "main.py"]