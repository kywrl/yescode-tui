#!/bin/bash

# Install Git hooks for this project
# 安装 Git hooks 到本地仓库

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
HOOKS_DIR="$PROJECT_ROOT/.git/hooks"

echo "🔧 Installing Git hooks..."

# Check if .git directory exists
if [ ! -d "$PROJECT_ROOT/.git" ]; then
    echo "❌ Error: .git directory not found"
    echo "   Please run this script from within a git repository"
    exit 1
fi

# Install pre-commit hook
if [ -f "$HOOKS_DIR/pre-commit" ]; then
    echo "⚠️  pre-commit hook already exists"
    read -p "   Overwrite? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "   Skipping pre-commit hook installation"
        exit 0
    fi
fi

cp "$SCRIPT_DIR/pre-commit" "$HOOKS_DIR/pre-commit"
chmod +x "$HOOKS_DIR/pre-commit"

echo "✅ Git hooks installed successfully!"
echo ""
echo "📝 Installed hooks:"
echo "   - pre-commit: Runs go fmt, go vet, and go test before each commit"
echo ""
echo "💡 To bypass hooks temporarily, use: git commit --no-verify"
