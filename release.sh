#!/bin/bash
# Release script for BPL Tools

set -e

# Check if version is provided
if [ $# -eq 0 ]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 v1.0.0"
    echo ""
    echo "This script will:"
    echo "  1. Create a git tag"
    echo "  2. Push the tag to GitHub"
    echo "  3. Trigger the automated release build"
    exit 1
fi

VERSION=$1

# Validate version format (should start with v)
if [[ ! $VERSION =~ ^v[0-9]+\.[0-9]+\.[0-9]+.*$ ]]; then
    echo "Error: Version should be in format vX.Y.Z (e.g., v1.0.0)"
    exit 1
fi

echo "Creating release $VERSION"
echo "========================="

# Check if we're in a git repository
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    echo "Error: Not in a git repository"
    exit 1
fi

# Check if there are uncommitted changes
if ! git diff-index --quiet HEAD --; then
    echo "Error: You have uncommitted changes. Please commit them first."
    git status --porcelain
    exit 1
fi

# Check if tag already exists
if git tag --list | grep -q "^$VERSION$"; then
    echo "Error: Tag $VERSION already exists"
    exit 1
fi

# Create and push the tag
echo "Creating tag $VERSION..."
git tag -a "$VERSION" -m "Release $VERSION"

echo "Pushing tag to GitHub..."
git push origin "$VERSION"

echo ""
echo "✓ Tag $VERSION created and pushed successfully!"
echo ""
echo "The GitHub Actions workflow will now:"
echo "  - Build binaries for all platforms"
echo "  - Create release packages with .env template"
echo "  - Create a GitHub release with download links"
echo ""
echo "Check the progress at: https://github.com/BPL-v2/tools/actions"
echo "Release will be available at: https://github.com/BPL-v2/tools/releases/tag/$VERSION"
