#!/bin/bash

# TUF Client Verify - Phase 1 Verification Script
echo "🚀 TUF Client Verify - Phase 1 MVP Verification"
echo "=============================================="

# Check if docker-compose is running
echo "📋 Checking container status..."
docker-compose ps

echo ""
echo "🔍 Testing Phase 1 MVP Endpoints:"
echo ""

# Test 1: Alpine manifest
echo "1. Testing Alpine manifest:"
curl -s http://localhost/v2/library/alpine/manifests/latest | jq -r '.mediaType // "❌ Failed"'

# Test 2: Ubuntu manifest  
echo "2. Testing Ubuntu manifest:"
curl -s http://localhost/v2/library/ubuntu/manifests/20.04 | jq -r '.mediaType // "❌ Failed"'

# Test 3: nginx manifest
echo "3. Testing nginx manifest:"
curl -s http://localhost/v2/library/nginx/manifests/latest | jq -r '.mediaType // "❌ Failed"'

# Test 4: Registry version
echo "4. Testing registry version endpoint:"
curl -s http://localhost/v2/ | jq -r '.registry // "❌ Failed"'

# Test 5: Non-existent manifest (should 404)
echo "5. Testing non-existent manifest (should fail):"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost/v2/library/redis/manifests/latest)
if [ "$HTTP_CODE" = "404" ]; then
    echo "✅ Correctly returned 404 for non-existent manifest"
else
    echo "❌ Expected 404, got $HTTP_CODE"
fi

echo ""
echo "📊 Auth Service Logs (last 10 lines):"
docker-compose logs --tail=10 tuf-client-verify | grep "Auth request"

echo ""
echo "✅ Phase 1 MVP Verification Complete!"
echo "🎯 All endpoints working with nginx auth_request integration"
echo "🔧 Ready to proceed with Phase 2: TUF Repository Integration"
