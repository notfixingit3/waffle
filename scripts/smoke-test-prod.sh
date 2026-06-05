#!/usr/bin/env bash
set -euo pipefail

# Project Syrup Production Smoke Test Script
# Tests core HTTP flows end-to-end against any running instance.
# Requires environment variables: BASE_URL, ADMIN_USER, ADMIN_PASS

# ── Validate required environment variables ──────────────────────────────
if [[ -z "${BASE_URL:-}" || -z "${ADMIN_USER:-}" || -z "${ADMIN_PASS:-}" ]]; then
    echo -e "\033[0;31mERROR: Missing required environment variables\033[0m"
    echo ""
    echo "The following environment variables must be set:"
    echo "  BASE_URL    - The base URL of the running application (e.g., http://localhost:8383)"
    echo "  ADMIN_USER  - Admin username for authentication"
    echo "  ADMIN_PASS  - Admin password for authentication"
    echo ""
    echo "Example:"
    echo "  BASE_URL=http://localhost:8383 ADMIN_USER=admin ADMIN_PASS=syrup ./scripts/smoke-test-prod.sh"
    exit 1
fi

PASS=0
FAIL=0

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

WAFFLE_ID=""
TOKEN=""

pass() {
    echo -e "${GREEN}PASS${NC}: $1"
    ((PASS++)) || true
}

fail() {
    echo -e "${RED}FAIL${NC}: $1"
    ((FAIL++)) || true
}

# ── Cleanup: attempt to delete test waffle via API ──────────────────────
cleanup_test_waffle() {
    if [[ -n "$WAFFLE_ID" && "$WAFFLE_ID" != "null" && -n "$TOKEN" && "$TOKEN" != "null" ]]; then
        DELETE_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "${BASE_URL}/api/admin/waffles/${WAFFLE_ID}" \
            -H "Authorization: Bearer ${TOKEN}" \
            -H "Content-Type: application/json" \
            -d "{\"current_password\":\"${ADMIN_PASS}\"}" 2>/dev/null || true)
        if [[ "$DELETE_STATUS" == "200" || "$DELETE_STATUS" == "204" ]]; then
            echo -e "${GREEN}Test waffle deleted successfully${NC}"
        else
            echo -e "${YELLOW}Warning: Test waffle created but not cleaned up (DELETE status: $DELETE_STATUS). Manual cleanup required.${NC}"
        fi
    fi
}

# ── Check dependencies ──────────────────────────────────────────────────
echo "========================================"
echo "Project Syrup Production Smoke Test"
echo "BASE_URL: $BASE_URL"
echo "========================================"

echo -e "\n${YELLOW}[1/9] Checking dependencies...${NC}"

if ! command -v curl &>/dev/null; then
    echo -e "${RED}Missing required command: curl${NC}"
    exit 1
fi

JQ_AVAILABLE=false
if command -v jq &>/dev/null; then
    JQ_AVAILABLE=true
    echo "jq is available (enhanced JSON parsing enabled)"
else
    echo "jq not available (falling back to grep-based JSON parsing)"
fi

# ── Helper: extract JSON field (uses jq if available, else grep) ───────
json_extract() {
    local field="$1"
    if $JQ_AVAILABLE; then
        jq -r "${field} // empty"
    else
        grep -oP '"'"${field}"'"\s*:\s*"\K[^"]+' || true
    fi
}

# Helper: extract JSON array/object raw value by key
grep_json_value() {
    local key="$1"
    grep -oP '"'"$key"'"\s*:\s*"\K[^"]+' || true
}

# ── 2. Test health endpoint ────────────────────────────────────────────
echo -e "\n${YELLOW}[2/9] Testing health endpoint...${NC}"
HEALTH_RESPONSE=$(curl -sf "${BASE_URL}/health" 2>/dev/null || true)
if [[ "$HEALTH_RESPONSE" == *'"status":"ok"'* ]]; then
    pass "GET /health returns 200 + status:ok"
else
    fail "GET /health did not return expected JSON (got: $HEALTH_RESPONSE)"
    echo -e "${RED}Aborting: target does not appear to be healthy.${NC}"
    exit 1
fi

# ── 3. Test admin login ────────────────────────────────────────────────
echo -e "\n${YELLOW}[3/9] Testing admin login...${NC}"
LOGIN_RESPONSE=$(curl -sf -X POST "${BASE_URL}/api/admin/login" \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"${ADMIN_USER}\",\"password\":\"${ADMIN_PASS}\"}" 2>/dev/null || true)

if $JQ_AVAILABLE; then
    TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.token // empty' 2>/dev/null || true)
else
    TOKEN=$(echo "$LOGIN_RESPONSE" | grep_json_value "token")
fi

if [[ -n "$TOKEN" && "$TOKEN" != "null" ]]; then
    pass "POST /api/admin/login returns 200 + token"
else
    fail "POST /api/admin/login did not return token (response: $LOGIN_RESPONSE)"
    echo -e "${RED}Aborting: cannot proceed without auth token.${NC}"
    exit 1
fi

# ── 4. Test create waffle ──────────────────────────────────────────────
echo -e "\n${YELLOW}[4/9] Testing create waffle...${NC}"
CREATE_RESPONSE=$(curl -sf -L -X POST "${BASE_URL}/api/admin/waffles/" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer ${TOKEN}" \
    -d '{"title":"Smoke Test Waffle","total_spots":10,"spot_price":5}' 2>/dev/null || true)

CREATE_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -L -X POST "${BASE_URL}/api/admin/waffles/" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer ${TOKEN}" \
    -d '{"title":"Smoke Test Waffle","total_spots":10,"spot_price":5}' 2>/dev/null || true)

if $JQ_AVAILABLE; then
    WAFFLE_ID=$(echo "$CREATE_RESPONSE" | jq -r '.id // empty' 2>/dev/null || true)
else
    WAFFLE_ID=$(echo "$CREATE_RESPONSE" | grep_json_value "id")
fi

if [[ "$CREATE_STATUS" == "201" && -n "$WAFFLE_ID" && "$WAFFLE_ID" != "null" ]]; then
    pass "POST /api/admin/waffles returns 201 + waffle id"
else
    fail "POST /api/admin/waffles did not return waffle id (status: $CREATE_STATUS, response: $CREATE_RESPONSE)"
fi

# ── 5. Test public waffle list ─────────────────────────────────────────
echo -e "\n${YELLOW}[5/9] Testing public waffle list...${NC}"
LIST_RESPONSE=$(curl -sf "${BASE_URL}/api/waffles/" 2>/dev/null || true)
if [[ "$LIST_RESPONSE" == *'"waffles"'* ]]; then
    pass "GET /api/waffles returns 200 + waffles array"
else
    fail "GET /api/waffles did not return expected JSON (got: $LIST_RESPONSE)"
fi

# ── 6. Test claim spot (need waffle id and spot number) ────────────────
echo -e "\n${YELLOW}[6/9] Testing claim spot...${NC}"
if [[ -n "$WAFFLE_ID" && "$WAFFLE_ID" != "null" ]]; then
    CLAIM_RESPONSE=$(curl -sf -L -X POST "${BASE_URL}/api/claims/" \
        -H "Content-Type: application/json" \
        -d "{\"waffle_id\":\"${WAFFLE_ID}\",\"spots\":[1],\"instagram_handle\":\"smoke_test_user\"}" 2>/dev/null || true)
    if [[ "$CLAIM_RESPONSE" == *'"message"'* ]]; then
        pass "POST /api/claims returns 200 + success message"
    else
        fail "POST /api/claims did not return expected response (got: $CLAIM_RESPONSE)"
    fi
else
    fail "Skipping claim spot — no waffle id from previous step"
fi

# ── 7. Test mark paid (need spot id) ───────────────────────────────────
echo -e "\n${YELLOW}[7/9] Testing mark spot paid...${NC}"
if [[ -n "$WAFFLE_ID" && "$WAFFLE_ID" != "null" ]]; then
    WAFFLE_SLUG=""
    if $JQ_AVAILABLE; then
        WAFFLE_SLUG=$(curl -sf "${BASE_URL}/api/waffles/" 2>/dev/null | jq -r '.waffles[] | select(.id=="'"$WAFFLE_ID"'") | .slug // empty' || true)
    else
        WAFFLE_SLUG=$(curl -sf "${BASE_URL}/api/waffles/" 2>/dev/null | grep -oP '"slug"\s*:\s*"\K[^"]+' | head -n1 || true)
    fi
    SPOTS_RESPONSE=$(curl -sf "${BASE_URL}/api/waffles/${WAFFLE_SLUG}/spots" 2>/dev/null || true)
    SPOT_ID=""
    if $JQ_AVAILABLE; then
        SPOT_ID=$(echo "$SPOTS_RESPONSE" | jq -r '.spots[] | select(.status=="pending") | .id' 2>/dev/null | head -n1 || true)
    else
        SPOT_ID=$(echo "$SPOTS_RESPONSE" | grep -oP '"id"\s*:\s*"\K[^"]+' 2>/dev/null | head -n1 || true)
    fi

    if [[ -n "$SPOT_ID" && "$SPOT_ID" != "null" ]]; then
        PAY_RESPONSE=$(curl -sf -X POST "${BASE_URL}/api/admin/spots/${SPOT_ID}/pay" \
            -H "Authorization: Bearer ${TOKEN}" 2>/dev/null || true)
        if [[ "$PAY_RESPONSE" == *'"message"'* ]]; then
            pass "POST /api/admin/spots/:id/pay returns 200 + success message"
        else
            fail "POST /api/admin/spots/:id/pay did not return expected response (got: $PAY_RESPONSE)"
        fi
    else
        fail "Could not find a spot id to mark paid"
    fi
else
    fail "Skipping mark paid — no waffle id from previous step"
fi

# ── 8. Test audit export ───────────────────────────────────────────────
echo -e "\n${YELLOW}[8/9] Testing audit export...${NC}"
AUDIT_RESPONSE=$(curl -sf "${BASE_URL}/api/admin/audit/export" \
    -H "Authorization: Bearer ${TOKEN}" 2>/dev/null || true)
if [[ "$AUDIT_RESPONSE" == *"id,admin_id,action,target_type,target_id,details,ip_address,created_at"* ]]; then
    pass "GET /api/admin/audit/export returns 200 + CSV headers"
else
    fail "GET /api/admin/audit/export did not return CSV headers (got: ${AUDIT_RESPONSE:0:200})"
fi

# ── 9. Cleanup test waffle ─────────────────────────────────────────────
echo -e "\n${YELLOW}[9/9] Cleaning up test waffle...${NC}"
cleanup_test_waffle

# ── Summary ────────────────────────────────────────────────────────────
echo -e "\n========================================"
echo -e "${GREEN}Passed: ${PASS}${NC}"
echo -e "${RED}Failed: ${FAIL}${NC}"
echo "========================================"

if [[ $FAIL -gt 0 ]]; then
    echo -e "${RED}SMOKE TEST FAILED${NC}"
    exit 1
else
    echo -e "${GREEN}SMOKE TEST PASSED${NC}"
    exit 0
fi
