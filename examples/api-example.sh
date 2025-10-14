#!/bin/bash

# NOT7 Server Mode - API and Deploy Folder Examples
# This script demonstrates both ways to deploy agents in server mode

echo "NOT7 Agent Runtime - Server Mode Examples"
echo "=========================================="
echo ""
echo "Make sure the server is running in another terminal:"
echo "  $ not7 run"
echo ""

# Check if server is running
echo "1. Testing server health..."
curl -s http://localhost:8080/health | jq . || echo "   ❌ Server not running. Start with: not7 run"
echo ""

# Method 1: HTTP API
echo "2. Deploying agent via HTTP API..."
curl -X POST http://localhost:8080/api/v1/agents/run \
  -H "Content-Type: application/json" \
  -d @examples/poem-generator.json \
  | jq .
echo ""

# Method 2: Deploy Folder
echo "3. Deploying agent via deploy folder..."
echo "   Copying agent spec to deploy/"
cp examples/poem-generator.json deploy/poem-from-folder.json
echo "   ✓ File copied. Server will automatically pick it up."
echo ""

sleep 3

echo "4. Check execution logs:"
echo "   $ ls -lt logs/ | head -5"
ls -lt logs/ | head -5
echo ""
echo "   $ tail -20 logs/agent-*.log | tail -10"
tail -20 logs/agent-*.log 2>/dev/null | tail -10 || echo "   (No logs yet)"
echo ""

echo "Done! Both deployment methods demonstrated."
echo ""
echo "To watch logs in real-time:"
echo "  $ tail -f logs/agent-*.log"
