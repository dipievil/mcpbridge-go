#!/bin/bash
# Quick test to verify SSE headers are correct

echo "Testing root endpoint / for SSE headers..."
curl -i http://localhost:3000/ 2>/dev/null | head -15

echo -e "\n\nTesting /sse endpoint for SSE headers..."
curl -i http://localhost:3000/sse 2>/dev/null | head -15

echo -e "\n\nTesting /health endpoint for JSON headers..."
curl -i http://localhost:3000/health 2>/dev/null | head -15
