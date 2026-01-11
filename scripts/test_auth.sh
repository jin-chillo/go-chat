#!/bin/bash

# GoChat Auth API Test Script
# Usage: ./scripts/test_auth.sh

set -e

BASE_URL="${BASE_URL:-http://localhost:8080}"
API_URL="$BASE_URL/api/v1"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counters
PASSED=0
FAILED=0

print_header() {
    echo ""
    echo "============================================"
    echo -e "${YELLOW}$1${NC}"
    echo "============================================"
}

print_result() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓ PASSED${NC}: $2"
        ((PASSED++))
    else
        echo -e "${RED}✗ FAILED${NC}: $2"
        ((FAILED++))
    fi
}

# Generate unique email for testing
TIMESTAMP=$(date +%s)
TEST_EMAIL="test${TIMESTAMP}@example.com"
TEST_PASSWORD="password123"
TEST_NICKNAME="TestUser${TIMESTAMP}"

echo "GoChat Auth API Test"
echo "Base URL: $BASE_URL"
echo "Test Email: $TEST_EMAIL"

# ============================================
# 1. Health Check
# ============================================
print_header "1. Health Check"

RESPONSE=$(curl -s -w "\n%{http_code}" "$BASE_URL/health")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | sed '$d')

if [ "$HTTP_CODE" = "200" ]; then
    print_result 0 "Health check endpoint"
    echo "   Response: $BODY"
else
    print_result 1 "Health check endpoint (HTTP $HTTP_CODE)"
fi

# ============================================
# 2. Register API
# ============================================
print_header "2. Register API"

# 2.1 Successful registration
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/auth/register" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$TEST_EMAIL\",\"password\":\"$TEST_PASSWORD\",\"nickname\":\"$TEST_NICKNAME\"}")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | sed '$d')

if [ "$HTTP_CODE" = "201" ]; then
    print_result 0 "User registration"
    USER_ID=$(echo "$BODY" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
    echo "   User ID: $USER_ID"
else
    print_result 1 "User registration (HTTP $HTTP_CODE)"
    echo "   Response: $BODY"
fi

# 2.2 Duplicate email
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/auth/register" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$TEST_EMAIL\",\"password\":\"$TEST_PASSWORD\",\"nickname\":\"DuplicateUser\"}")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | sed '$d')

if [ "$HTTP_CODE" = "409" ]; then
    print_result 0 "Duplicate email rejection"
else
    print_result 1 "Duplicate email rejection (expected 409, got $HTTP_CODE)"
fi

# 2.3 Validation - short password
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/auth/register" \
    -H "Content-Type: application/json" \
    -d '{"email":"short@example.com","password":"short","nickname":"ShortPwd"}')
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)

if [ "$HTTP_CODE" = "400" ]; then
    print_result 0 "Short password validation"
else
    print_result 1 "Short password validation (expected 400, got $HTTP_CODE)"
fi

# 2.4 Validation - invalid email
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/auth/register" \
    -H "Content-Type: application/json" \
    -d '{"email":"invalid-email","password":"password123","nickname":"InvalidEmail"}')
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)

if [ "$HTTP_CODE" = "400" ]; then
    print_result 0 "Invalid email validation"
else
    print_result 1 "Invalid email validation (expected 400, got $HTTP_CODE)"
fi

# ============================================
# 3. Login API
# ============================================
print_header "3. Login API"

# 3.1 Successful login
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$TEST_EMAIL\",\"password\":\"$TEST_PASSWORD\"}")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | sed '$d')

if [ "$HTTP_CODE" = "200" ]; then
    print_result 0 "User login"
    ACCESS_TOKEN=$(echo "$BODY" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
    REFRESH_TOKEN=$(echo "$BODY" | grep -o '"refresh_token":"[^"]*"' | cut -d'"' -f4)
    echo "   Access Token: ${ACCESS_TOKEN:0:50}..."
    echo "   Refresh Token: ${REFRESH_TOKEN:0:20}..."
else
    print_result 1 "User login (HTTP $HTTP_CODE)"
    echo "   Response: $BODY"
fi

# 3.2 Wrong password
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$TEST_EMAIL\",\"password\":\"wrongpassword\"}")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)

if [ "$HTTP_CODE" = "401" ]; then
    print_result 0 "Wrong password rejection"
else
    print_result 1 "Wrong password rejection (expected 401, got $HTTP_CODE)"
fi

# 3.3 Non-existent user
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"email":"nonexistent@example.com","password":"password123"}')
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)

if [ "$HTTP_CODE" = "401" ]; then
    print_result 0 "Non-existent user rejection"
else
    print_result 1 "Non-existent user rejection (expected 401, got $HTTP_CODE)"
fi

# ============================================
# 4. Token Refresh API
# ============================================
print_header "4. Token Refresh API"

if [ -n "$REFRESH_TOKEN" ]; then
    # 4.1 Successful refresh
    RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/auth/refresh" \
        -H "Content-Type: application/json" \
        -d "{\"refresh_token\":\"$REFRESH_TOKEN\"}")
    HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
    BODY=$(echo "$RESPONSE" | sed '$d')

    if [ "$HTTP_CODE" = "200" ]; then
        print_result 0 "Token refresh"
        NEW_ACCESS_TOKEN=$(echo "$BODY" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
        echo "   New Access Token: ${NEW_ACCESS_TOKEN:0:50}..."
        ACCESS_TOKEN="$NEW_ACCESS_TOKEN"
    else
        print_result 1 "Token refresh (HTTP $HTTP_CODE)"
        echo "   Response: $BODY"
    fi

    # 4.2 Invalid refresh token
    RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/auth/refresh" \
        -H "Content-Type: application/json" \
        -d '{"refresh_token":"invalid-refresh-token"}')
    HTTP_CODE=$(echo "$RESPONSE" | tail -n1)

    if [ "$HTTP_CODE" = "401" ]; then
        print_result 0 "Invalid refresh token rejection"
    else
        print_result 1 "Invalid refresh token rejection (expected 401, got $HTTP_CODE)"
    fi
else
    print_result 1 "Token refresh (no refresh token from login)"
fi

# ============================================
# 5. Logout API
# ============================================
print_header "5. Logout API"

if [ -n "$ACCESS_TOKEN" ]; then
    # 5.1 Successful logout
    RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/auth/logout" \
        -H "Authorization: Bearer $ACCESS_TOKEN")
    HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
    BODY=$(echo "$RESPONSE" | sed '$d')

    if [ "$HTTP_CODE" = "200" ]; then
        print_result 0 "User logout"
        echo "   Response: $BODY"
    else
        print_result 1 "User logout (HTTP $HTTP_CODE)"
        echo "   Response: $BODY"
    fi

    # 5.2 Use blacklisted token
    sleep 1
    RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/auth/logout" \
        -H "Authorization: Bearer $ACCESS_TOKEN")
    HTTP_CODE=$(echo "$RESPONSE" | tail -n1)

    if [ "$HTTP_CODE" = "401" ]; then
        print_result 0 "Blacklisted token rejection"
    else
        print_result 1 "Blacklisted token rejection (expected 401, got $HTTP_CODE)"
    fi
else
    print_result 1 "Logout test (no access token)"
fi

# 5.3 Missing authorization header
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/auth/logout")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)

if [ "$HTTP_CODE" = "401" ]; then
    print_result 0 "Missing auth header rejection"
else
    print_result 1 "Missing auth header rejection (expected 401, got $HTTP_CODE)"
fi

# ============================================
# 6. Rate Limiting Test
# ============================================
print_header "6. Rate Limiting Test"

echo "Testing register rate limit (3 req/min)..."
RATE_LIMITED=0
for i in {1..5}; do
    RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/auth/register" \
        -H "Content-Type: application/json" \
        -d "{\"email\":\"ratelimit${i}${TIMESTAMP}@example.com\",\"password\":\"password123\",\"nickname\":\"RateTest${i}\"}")
    HTTP_CODE=$(echo "$RESPONSE" | tail -n1)

    if [ "$HTTP_CODE" = "429" ]; then
        RATE_LIMITED=1
        echo "   Request $i: Rate limited (429)"
        break
    else
        echo "   Request $i: HTTP $HTTP_CODE"
    fi
done

if [ "$RATE_LIMITED" = "1" ]; then
    print_result 0 "Register rate limiting"
else
    print_result 1 "Register rate limiting (expected 429 after 3 requests)"
fi

# ============================================
# Summary
# ============================================
print_header "Test Summary"
TOTAL=$((PASSED + FAILED))
echo -e "Total: $TOTAL tests"
echo -e "${GREEN}Passed: $PASSED${NC}"
echo -e "${RED}Failed: $FAILED${NC}"

if [ "$FAILED" -eq 0 ]; then
    echo ""
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo ""
    echo -e "${RED}Some tests failed.${NC}"
    exit 1
fi
