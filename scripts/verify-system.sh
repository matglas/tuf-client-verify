#!/bin/bash

# TUF Client Verify - System Verification Script
# This script performs comprehensive verification of the TUF Client Verify system

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
AUTH_SERVICE_URL="http://localhost:8080"
NGINX_URL="http://localhost"

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_dependencies() {
    log_info "Checking dependencies..."
    
    # Check if jq is installed
    if ! command -v jq &> /dev/null; then
        log_error "jq is required but not installed. Please install jq."
        exit 1
    fi
    
    # Check if curl is installed
    if ! command -v curl &> /dev/null; then
        log_error "curl is required but not installed. Please install curl."
        exit 1
    fi
    
    log_success "All dependencies are available"
}

check_services() {
    log_info "Checking if services are running..."
    
    # Check auth service
    if curl -s -f "$AUTH_SERVICE_URL/health" > /dev/null; then
        log_success "Auth service is running (port 8080)"
    else
        log_error "Auth service is not running. Please start with: docker-compose up -d"
        exit 1
    fi
    
    # Check nginx
    if curl -s -f "$NGINX_URL/nginx-health" > /dev/null; then
        log_success "nginx is running (port 80)"
    else
        log_error "nginx is not running. Please start with: docker-compose up -d"
        exit 1
    fi
}

check_tuf_repository() {
    log_info "Checking TUF repository..."
    
    if [ ! -d "testdata/repository" ]; then
        log_error "TUF repository not found. Please run: go run scripts/generate-tuf-repo.go"
        exit 1
    fi
    
    # Check required TUF metadata files
    required_files=("root.json" "targets.json" "snapshot.json" "timestamp.json" "registry-library.json")
    for file in "${required_files[@]}"; do
        if [ ! -f "testdata/repository/$file" ]; then
            log_error "Missing TUF metadata file: $file"
            exit 1
        fi
    done
    
    log_success "TUF repository is complete"
}

test_auth_service_direct() {
    log_info "Testing auth service directly..."
    
    # Test allowed path
    local allowed_response
    allowed_response=$(curl -s -H "X-Original-URI: /v2/library/alpine/manifests/latest" "$AUTH_SERVICE_URL/auth")
    if [ "$allowed_response" = "OK" ]; then
        log_success "✅ Allowed path test passed"
    else
        log_error "❌ Allowed path test failed. Expected 'OK', got: $allowed_response"
        return 1
    fi
    
    # Test denied path
    local denied_response
    denied_response=$(curl -s -H "X-Original-URI: /v2/redis/manifests/latest" "$AUTH_SERVICE_URL/auth")
    if [ "$denied_response" = "Forbidden" ]; then
        log_success "✅ Denied path test passed"
    else
        log_error "❌ Denied path test failed. Expected 'Forbidden', got: $denied_response"
        return 1
    fi
}

test_health_and_debug() {
    log_info "Testing health and debug endpoints..."
    
    # Test health endpoint
    local health_response
    health_response=$(curl -s "$AUTH_SERVICE_URL/health")
    if [ "$health_response" = "healthy" ]; then
        log_success "✅ Health endpoint working"
    else
        log_error "❌ Health endpoint failed. Got: $health_response"
        return 1
    fi
    
    # Test debug endpoint and validate JSON
    local debug_response
    debug_response=$(curl -s "$AUTH_SERVICE_URL/debug")
    if echo "$debug_response" | jq . > /dev/null 2>&1; then
        log_success "✅ Debug endpoint working (valid JSON)"
        
        # Check delegation configuration
        local delegation_count
        delegation_count=$(echo "$debug_response" | jq '.delegations | length')
        if [ "$delegation_count" -gt 0 ]; then
            log_success "✅ TUF delegation configured"
        else
            log_warning "⚠️ No TUF delegations found"
        fi
    else
        log_error "❌ Debug endpoint failed or returned invalid JSON"
        return 1
    fi
}

test_nginx_integration() {
    log_info "Testing nginx integration..."
    
    # Test allowed path through nginx
    local nginx_allowed_status
    nginx_allowed_status=$(curl -s -w "%{http_code}" -o /dev/null "$NGINX_URL/v2/library/nginx/manifests/latest")
    if [ "$nginx_allowed_status" = "200" ]; then
        log_success "✅ nginx allows authorized paths"
    else
        log_error "❌ nginx integration failed for allowed path. Status: $nginx_allowed_status"
        return 1
    fi
    
    # Test denied path through nginx
    local nginx_denied_status
    nginx_denied_status=$(curl -s -w "%{http_code}" -o /dev/null "$NGINX_URL/v2/redis/manifests/latest")
    if [ "$nginx_denied_status" = "403" ]; then
        log_success "✅ nginx denies unauthorized paths"
    else
        log_error "❌ nginx integration failed for denied path. Status: $nginx_denied_status"
        return 1
    fi
}

test_edge_cases() {
    log_info "Testing edge cases..."
    
    # Test different allowed paths
    local ubuntu_response
    ubuntu_response=$(curl -s -H "X-Original-URI: /v2/library/ubuntu/manifests/20.04" "$AUTH_SERVICE_URL/auth")
    if [ "$ubuntu_response" = "OK" ]; then
        log_success "✅ Ubuntu path allowed"
    else
        log_error "❌ Ubuntu path test failed"
        return 1
    fi
    
    # Test similar but denied path
    local library_test_response
    library_test_response=$(curl -s -H "X-Original-URI: /v2/library-test/alpine/manifests/latest" "$AUTH_SERVICE_URL/auth")
    if [ "$library_test_response" = "Forbidden" ]; then
        log_success "✅ Similar but unauthorized path denied"
    else
        log_error "❌ Similar path test failed"
        return 1
    fi
    
    # Test root path
    local root_response
    root_response=$(curl -s -H "X-Original-URI: /v2/" "$AUTH_SERVICE_URL/auth")
    if [ "$root_response" = "Forbidden" ]; then
        log_success "✅ Root path properly denied"
    else
        log_error "❌ Root path test failed"
        return 1
    fi
}

validate_tuf_metadata() {
    log_info "Validating TUF metadata..."
    
    # Check root metadata signatures
    local signature_count
    signature_count=$(cat testdata/repository/root.json | jq '.signatures | length')
    if [ "$signature_count" -gt 0 ]; then
        log_success "✅ Root metadata is signed ($signature_count signatures)"
    else
        log_error "❌ Root metadata is not signed"
        return 1
    fi
    
    # Check delegation configuration
    local delegation_paths
    delegation_paths=$(cat testdata/repository/targets.json | jq -r '.signed.delegations.roles[] | select(.name=="registry-library") | .paths[]')
    if echo "$delegation_paths" | grep -q "/v2/library/\*"; then
        log_success "✅ Delegation configured for /v2/library/*"
    else
        log_error "❌ Delegation not properly configured"
        return 1
    fi
}

print_summary() {
    local total_tests=$1
    local failed_tests=$2
    
    echo ""
    echo "========================================"
    echo "         VERIFICATION SUMMARY"
    echo "========================================"
    
    if [ "$failed_tests" -eq 0 ]; then
        log_success "🎉 ALL $total_tests TESTS PASSED!"
        echo ""
        log_info "System Status:"
        echo "  🎯 TUF Repository: ✅ Complete with signed metadata"
        echo "  🔐 TUF Delegation: ✅ /v2/library/* → registry-library role"
        echo "  🚦 Auth Service: ✅ Correctly allows/denies based on TUF"
        echo "  🌐 nginx Integration: ✅ Proper auth_request handling"
        echo "  🐳 Docker Services: ✅ Running healthy"
        echo ""
        log_success "Phase 2 TUF integration is working correctly!"
    else
        log_error "❌ $failed_tests out of $total_tests tests failed"
        echo ""
        log_error "Please check the errors above and fix the issues."
        exit 1
    fi
}

main() {
    echo "========================================"
    echo "    TUF Client Verify - System Verification"
    echo "========================================"
    echo ""
    
    local failed_tests=0
    local total_tests=6
    
    # Pre-flight checks
    check_dependencies
    check_services
    check_tuf_repository
    
    echo ""
    log_info "Starting verification tests..."
    echo ""
    
    # Run tests
    test_auth_service_direct || ((failed_tests++))
    echo ""
    
    test_health_and_debug || ((failed_tests++))
    echo ""
    
    test_nginx_integration || ((failed_tests++))
    echo ""
    
    test_edge_cases || ((failed_tests++))
    echo ""
    
    validate_tuf_metadata || ((failed_tests++))
    echo ""
    
    print_summary $total_tests $failed_tests
}

# Run main function
main "$@"
