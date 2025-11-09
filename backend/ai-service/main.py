#!/usr/bin/env python3

import logging
from concurrent import futures
import grpc

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

def serve():
    """Start the AI service gRPC server."""
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    
    # TODO: Add service implementation here
    # pb2_grpc.add_AIServiceServicer_to_server(AIServiceServicer(), server)
    
    listen_addr = '[::]:9008'
    server.add_insecure_port(listen_addr)
    
    logger.info(f"AI service starting on {listen_addr}")
    server.start()
    
    try:
        server.wait_for_termination()
    except KeyboardInterrupt:
        logger.info("Shutting down AI service...")
        server.stop(0)

if __name__ == '__main__':
    serve()