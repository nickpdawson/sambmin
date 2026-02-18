#!/bin/bash
# sambmin-ralph.sh — Ralph Wiggum loop runner for Sambmin
# 
# Usage:
#   ./sambmin-ralph.sh                    # Run full loop (M13-M20)
#   ./sambmin-ralph.sh --milestone 13     # Run single milestone only
#   ./sambmin-ralph.sh --dry-run          # Show what would run
#
# Prerequisites:
#   - Claude Code CLI installed and authenticated (`claude` command available)
#   - Run from Sambmin project root directory
#
# Files expected in project root:
#   - CLAUDE.md, plan.md, sambmin-prd.json, RALPH-PROMPT.md

set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROMPT_FILE="${PROJECT_DIR}/RALPH-PROMPT.md"
PROGRESS_FILE="${PROJECT_DIR}/ralph-progress.md"
PRD_FILE="${PROJECT_DIR}/sambmin-prd.json"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Defaults
MAX_ITERATIONS=50
MILESTONE=""
DRY_RUN=false

# Parse args
while [[ $# -gt 0 ]]; do
    case $1 in
        --milestone|-m)
            MILESTONE="$2"
            shift 2
            ;;
        --max-iterations|-n)
            MAX_ITERATIONS="$2"
            shift 2
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --help|-h)
            echo "Usage: $0 [--milestone N] [--max-iterations N] [--dry-run]"
            echo ""
            echo "Options:"
            echo "  --milestone, -m N       Run only milestone N (13-20)"
            echo "  --max-iterations, -n N  Max iterations per milestone (default: 50)"
            echo "  --dry-run               Show what would run without executing"
            echo "  --help, -h              Show this help"
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            exit 1
            ;;
    esac
done

# Preflight checks
echo -e "${BLUE}=== Sambmin Ralph Loop ===${NC}"
echo -e "Project dir: ${PROJECT_DIR}"
echo -e "Max iterations per milestone: ${MAX_ITERATIONS}"

for f in CLAUDE.md plan.md sambmin-prd.json RALPH-PROMPT.md; do
    if [[ ! -f "${PROJECT_DIR}/${f}" ]]; then
        echo -e "${RED}Missing required file: ${f}${NC}"
        exit 1
    fi
done
echo -e "${GREEN}All required files present.${NC}"

# Initialize progress file if it doesn't exist
if [[ ! -f "${PROGRESS_FILE}" ]]; then
    cat > "${PROGRESS_FILE}" << EOF
# Sambmin Ralph Progress Log

Started: $(date '+%Y-%m-%d %H:%M')
Project state: M13 in progress
Runner: sambmin-ralph.sh

---
EOF
    echo -e "${YELLOW}Created ralph-progress.md${NC}"
fi

# Check for Claude Code CLI
if ! command -v claude &> /dev/null; then
    echo -e "${RED}Claude Code CLI not found. Install from https://docs.claude.com${NC}"
    exit 1
fi
echo -e "${GREEN}Claude Code CLI found.${NC}"

# Build the prompt
build_prompt() {
    local milestone_filter=""
    if [[ -n "${MILESTONE}" ]]; then
        milestone_filter="Focus ONLY on Milestone M${MILESTONE}. Do not work on other milestones."
    fi

    cat "${PROMPT_FILE}"

    if [[ -n "${milestone_filter}" ]]; then
        echo ""
        echo "## OVERRIDE: Single Milestone Mode"
        echo "${milestone_filter}"
        echo "Output <promise>MILESTONE_COMPLETE</promise> when M${MILESTONE} is done."
    fi
}

# Run it
if [[ "${DRY_RUN}" == true ]]; then
    echo ""
    echo -e "${YELLOW}=== DRY RUN — Would execute: ===${NC}"
    echo ""
    if [[ -n "${MILESTONE}" ]]; then
        echo "  while loop: cat RALPH-PROMPT.md + M${MILESTONE} focus | claude --print"
        echo "    completion check: grep for 'MILESTONE_COMPLETE'"
        echo "    max iterations: ${MAX_ITERATIONS}"
    else
        echo "  while loop: cat RALPH-PROMPT.md | claude --print"
        echo "    completion check: grep for 'SAMBMIN_COMPLETE'"
        echo "    max iterations: ${MAX_ITERATIONS}"
    fi
    echo ""
    echo -e "${BLUE}Prompt preview (first 20 lines):${NC}"
    build_prompt | head -20
    echo "  ..."
    exit 0
fi

# Determine completion promise
if [[ -n "${MILESTONE}" ]]; then
    PROMISE="MILESTONE_COMPLETE"
    echo -e "${BLUE}Running M${MILESTONE} only (max ${MAX_ITERATIONS} iterations)${NC}"
else
    PROMISE="SAMBMIN_COMPLETE"
    echo -e "${BLUE}Running full loop M13-M20 (max ${MAX_ITERATIONS} iterations)${NC}"
fi

echo ""
echo -e "${YELLOW}Starting Ralph loop at $(date '+%Y-%m-%d %H:%M:%S')${NC}"
echo -e "${YELLOW}Progress logged to: ralph-progress.md${NC}"
echo -e "${YELLOW}Press Ctrl+C to abort${NC}"
echo ""

# Log start to progress file
cat >> "${PROGRESS_FILE}" << EOF

---
## Ralph Loop Started — $(date '+%Y-%m-%d %H:%M')
**Mode:** $([ -n "${MILESTONE}" ] && echo "Single milestone M${MILESTONE}" || echo "Full run M13-M20")
**Max iterations:** ${MAX_ITERATIONS}

EOF

# Run the Ralph loop — pure bash Ralph Wiggum technique
# Uses claude -p (prompt flag) which gives full project context (file read/write, tools)
# NOT --print which is headless/no-tools mode
ITERATION=0
COMPLETED=false
LOGFILE="${PROJECT_DIR}/ralph-loop-output.log"

echo -e "${YELLOW}Full output logged to: ralph-loop-output.log${NC}"

while [[ ${ITERATION} -lt ${MAX_ITERATIONS} ]]; do
    ITERATION=$((ITERATION + 1))
    ITER_START=$(date '+%Y-%m-%d %H:%M:%S')
    echo ""
    echo -e "${BLUE}━━━ Iteration ${ITERATION}/${MAX_ITERATIONS} — ${ITER_START} ━━━${NC}"

    # Build the prompt text
    PROMPT_TEXT=$(build_prompt)

    # Use -p flag for single-prompt mode WITH project context (tool use, file access)
    # --output-format text gives us clean text output
    echo "=== Iteration ${ITERATION} — ${ITER_START} ===" >> "${LOGFILE}"
    OUTPUT=$(claude -p "${PROMPT_TEXT}" --output-format text 2>&1) || true
    echo "${OUTPUT}" >> "${LOGFILE}"
    echo "" >> "${LOGFILE}"

    ITER_END=$(date '+%Y-%m-%d %H:%M:%S')

    # Show summary (last 30 lines)
    echo "${OUTPUT}" | tail -30

    # Log iteration to progress file (the script does this since Claude may not)
    CHANGES=$(git diff --stat 2>/dev/null || echo "no git repo")
    NEW_FILES=$(git ls-files --others --exclude-standard 2>/dev/null | head -10 || echo "none")
    TEST_SNIPPET=$(echo "${OUTPUT}" | grep -i -E "(PASS|FAIL|ok |---)" | tail -5 || echo "no test output found")

    cat >> "${PROGRESS_FILE}" << LOGEOF

---
## Loop ${ITERATION} — ${ITER_START}
**Milestone:** (see output)
**Duration:** ${ITER_START} to ${ITER_END}
**Git changes:**
\`\`\`
${CHANGES}
\`\`\`
**New files:** ${NEW_FILES}
**Test output (snippet):**
\`\`\`
${TEST_SNIPPET}
\`\`\`
**Completion promise found:** $(echo "${OUTPUT}" | grep -qF "${PROMISE}" && echo "YES" || echo "no")
LOGEOF

    # Check for completion promise in output
    if echo "${OUTPUT}" | grep -qF "${PROMISE}"; then
        echo ""
        echo -e "${GREEN}━━━ Completion promise '${PROMISE}' detected at iteration ${ITERATION}! ━━━${NC}"
        COMPLETED=true
        break
    fi

    echo -e "${YELLOW}  (no completion promise — continuing to iteration $((ITERATION + 1)))${NC}"

    # Brief pause between iterations
    sleep 3
done

# Log completion
cat >> "${PROGRESS_FILE}" << EOF

---
## Ralph Loop Ended — $(date '+%Y-%m-%d %H:%M')
**Iterations:** ${ITERATION}/${MAX_ITERATIONS}
**Completed:** ${COMPLETED}
**Reason:** $(${COMPLETED} && echo "Completion promise matched" || echo "Max iterations reached")

EOF

if ${COMPLETED}; then
    echo -e "${GREEN}=== Ralph loop completed successfully after ${ITERATION} iterations ===${NC}"
else
    echo -e "${YELLOW}=== Ralph loop hit max iterations (${MAX_ITERATIONS}) — check ralph-progress.md ===${NC}"
    echo -e "${YELLOW}    Run again to continue where it left off.${NC}"
fi
