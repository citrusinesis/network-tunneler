#!/bin/bash
set -e

# Preview script: Show what will change without modifying files
# This is a safe way to see the impact before running rename.sh

echo "========================================="
echo "Network Tunneler Rename Preview"
echo "========================================="
echo "Changes that would be made:"
echo "  Agent (in compounds) → Client (AgentListenAddr → ClientListenAddr)"
echo "  agent (standalone)   → client"
echo "  AGENT (standalone)   → CLIENT"
echo "  Implant (in compounds) → Proxy (ImplantRegistry → ProxyRegistry)"
echo "  implant (standalone)   → proxy"
echo "  IMPLANT (standalone)   → PROXY"
echo ""

# Check if we're in the right directory
if [ ! -f "go.mod" ]; then
    echo "ERROR: Must run from project root (network-tunneler/)"
    exit 1
fi

# Find files that will be modified
echo "=== Files to be modified ==="
FILES=$(find . -type f \
    \( -name "*.go" -o -name "*.md" -o -name "*.proto" -o -name "Makefile" \) \
    ! -path "./.git/*" \
    ! -path "./bin/*" \
    ! -path "./.go/*" \
    ! -path "./.direnv/*" \
    ! -path "./.idea/*" \
    ! -path "./.vscode/*" \
    ! -path "*/proto/*.pb.go")

echo "$FILES" | while read -r file; do
    if grep -q '\(agent\|Agent\|AGENT\|implant\|Implant\|IMPLANT\)' "$file" 2>/dev/null; then
        echo "  • $file"
    fi
done

echo ""
echo "=== Directories to be renamed ==="
[ -d "cmd/agent" ] && echo "  • cmd/agent → cmd/client"
[ -d "cmd/implant" ] && echo "  • cmd/implant → cmd/proxy"
[ -d "internal/agent" ] && echo "  • internal/agent → internal/client"
[ -d "internal/implant" ] && echo "  • internal/implant → internal/proxy"

echo ""
echo "=== Sample changes (first 20 matches) ==="
grep -n '\(agent\|Agent\|AGENT\|implant\|Implant\|IMPLANT\)' \
    --include="*.go" \
    --include="*.md" \
    --include="*.proto" \
    --include="Makefile" \
    --exclude-dir=.git \
    --exclude-dir=bin \
    --exclude-dir=.go \
    -r . 2>/dev/null | head -20

echo ""
echo "=== Compound identifier examples ==="
echo "Searching for compound identifiers..."
grep -h 'Agent[A-Z]' --include="*.go" -r . 2>/dev/null | head -5 | sed 's/^/  • /'
grep -h 'Implant[A-Z]' --include="*.go" -r . 2>/dev/null | head -5 | sed 's/^/  • /'

echo ""
echo "=== Statistics ==="
AGENT_COUNT=$(grep -o 'agent\|Agent\|AGENT' \
    --include="*.go" \
    --include="*.md" \
    --include="*.proto" \
    --include="Makefile" \
    --exclude-dir=.git \
    --exclude-dir=bin \
    -r . 2>/dev/null | wc -l)

IMPLANT_COUNT=$(grep -o 'implant\|Implant\|IMPLANT' \
    --include="*.go" \
    --include="*.md" \
    --include="*.proto" \
    --include="Makefile" \
    --exclude-dir=.git \
    --exclude-dir=bin \
    -r . 2>/dev/null | wc -l)

echo "  • 'agent' variants:   $AGENT_COUNT occurrences"
echo "  • 'implant' variants: $IMPLANT_COUNT occurrences"
echo ""
echo "========================================="
echo "To apply changes, run: ./rename.sh"
echo "========================================="
