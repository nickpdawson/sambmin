#!/bin/bash
# sambmin-fix.sh — One-shot fix pass for deployment issues
# Run from Sambmin project root

set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROMPT_FILE="${PROJECT_DIR}/RALPH-FIX.md"
LOGFILE="${PROJECT_DIR}/ralph-fix-output.log"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

MAX_ITERATIONS=${1:-10}

if [[ ! -f "${PROMPT_FILE}" ]]; then
    echo -e "${RED}Missing RALPH-FIX.md${NC}"
    exit 1
fi

echo -e "${BLUE}=== Sambmin Fix Pass ===${NC}"
echo -e "Max iterations: ${MAX_ITERATIONS}"
echo -e "Full output: ralph-fix-output.log"
echo ""

ITERATION=0
COMPLETED=false

while [[ ${ITERATION} -lt ${MAX_ITERATIONS} ]]; do
    ITERATION=$((ITERATION + 1))
    ITER_START=$(date '+%Y-%m-%d %H:%M:%S')
    echo -e "${BLUE}━━━ Fix iteration ${ITERATION}/${MAX_ITERATIONS} — ${ITER_START} ━━━${NC}"

    PROMPT_TEXT=$(cat "${PROMPT_FILE}")

    echo "=== Iteration ${ITERATION} — ${ITER_START} ===" >> "${LOGFILE}"
    OUTPUT=$(claude -p "${PROMPT_TEXT}" --output-format text 2>&1) || true
    echo "${OUTPUT}" >> "${LOGFILE}"
    echo "" >> "${LOGFILE}"

    # Show summary
    echo "${OUTPUT}" | tail -40

    # Log git state
    echo -e "${YELLOW}Changed files:${NC}"
    git diff --stat 2>/dev/null || true

    if echo "${OUTPUT}" | grep -qF "FIXES_COMPLETE"; then
        echo ""
        echo -e "${GREEN}━━━ Fix pass complete at iteration ${ITERATION}! ━━━${NC}"
        COMPLETED=true
        break
    fi

    echo -e "${YELLOW}  (continuing to iteration $((ITERATION + 1)))${NC}"
    sleep 3
done

if ${COMPLETED}; then
    echo -e "${GREEN}=== All fixes applied after ${ITERATION} iterations ===${NC}"
    echo -e "${GREEN}    Deploy to Bridger and re-test.${NC}"
else
    echo -e "${YELLOW}=== Hit max iterations — check ralph-fix-output.log ===${NC}"
fi
