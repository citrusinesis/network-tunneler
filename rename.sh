#!/bin/bash
set -e

# Comprehensive rename script: agent→client, implant→proxy
# This script renames all occurrences in code, documentation, and file paths
# Including compound identifiers like AgentListenAddr → ClientListenAddr

echo "========================================="
echo "Network Tunneler Rename Script"
echo "========================================="
echo "Changes:"
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

# Step 1: Preview changes
echo "Step 1: Finding files to modify..."
FILES=$(find . -type f \
    \( -name "*.go" -o -name "*.md" -o -name "*.proto" -o -name "Makefile" \) \
    ! -path "./.git/*" \
    ! -path "./bin/*" \
    ! -path "./.go/*" \
    ! -path "./.direnv/*" \
    ! -path "./.idea/*" \
    ! -path "./.vscode/*" \
    ! -path "*/proto/*.pb.go")

FILE_COUNT=$(echo "$FILES" | wc -l)
echo "Found $FILE_COUNT files to process"
echo ""

# Step 2: Ask for confirmation
read -p "Continue with replacement? (y/N) " -n 1 -r
echo ""
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Aborted."
    exit 0
fi

# Step 3: Replace in file contents
echo ""
echo "Step 2: Replacing content in files..."

for file in $FILES; do
    # Create backup
    cp "$file" "$file.bak"

    # Use Perl for better regex support
    # Order matters: do compound replacements first, then standalone
    perl -i -pe '
        # Compound identifiers (PascalCase/camelCase) - no word boundaries
        s/Agent/Client/g;
        s/Implant/Proxy/g;

        # Standalone lowercase and uppercase (with word boundaries)
        s/\bagent\b/client/g;
        s/\bAGENT\b/CLIENT/g;
        s/\bimplant\b/proxy/g;
        s/\bIMPLANT\b/PROXY/g;
    ' "$file"

    echo "  ✓ $file"
done

# Step 4: Rename directories
echo ""
echo "Step 3: Renaming directories..."

if [ -d "cmd/agent" ]; then
    git mv cmd/agent cmd/client 2>/dev/null || mv cmd/agent cmd/client
    echo "  ✓ cmd/agent → cmd/client"
fi

if [ -d "cmd/implant" ]; then
    git mv cmd/implant cmd/proxy 2>/dev/null || mv cmd/implant cmd/proxy
    echo "  ✓ cmd/implant → cmd/proxy"
fi

if [ -d "internal/agent" ]; then
    git mv internal/agent internal/client 2>/dev/null || mv internal/agent internal/client
    echo "  ✓ internal/agent → internal/client"
fi

if [ -d "internal/implant" ]; then
    git mv internal/implant internal/proxy 2>/dev/null || mv internal/implant internal/proxy
    echo "  ✓ internal/implant → internal/proxy"
fi

# Step 5: Regenerate protobuf (if proto files were changed)
echo ""
echo "Step 4: Regenerating protobuf files..."
if command -v protoc &> /dev/null; then
    make proto 2>/dev/null || echo "  ⚠ Run 'make proto' manually to regenerate"
else
    echo "  ⚠ protoc not found - run 'make proto' manually"
fi

# Step 6: Update go.mod dependencies
echo ""
echo "Step 5: Updating Go module..."
go mod tidy

echo ""
echo "========================================="
echo "✓ Rename complete!"
echo "========================================="
echo ""
echo "Next steps:"
echo "  1. Review changes:    git diff"
echo "  2. Test build:        make dev-client dev-server dev-proxy"
echo "  3. Remove backups:    find . -name '*.bak' -delete"
echo "  4. Commit:            git add -A && git commit -m 'Rename agent→client, implant→proxy'"
echo ""
echo "To rollback:"
echo "  find . -name '*.bak' -exec sh -c 'mv \"\$1\" \"\${1%.bak}\"' _ {} \\;"
echo ""
