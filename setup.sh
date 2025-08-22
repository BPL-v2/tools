#!/bin/bash
# BPL Tools Setup Script
# Note: This script is now optional! The BPL Tools application handles 
# environment variable setup automatically.

echo "BPL Tools Setup"
echo "==============="
echo ""
echo "⚠️  NOTE: This setup script is now OPTIONAL!"
echo "   BPL Tools will automatically prompt you for credentials when needed."
echo ""

# Check if .env file exists
if [ ! -f ".env" ]; then
    echo "Creating .env file from template..."
    if [ -f ".env.example" ]; then
        cp .env.example .env
        echo "✓ Created .env file from template"
    else
        echo "Creating basic .env file..."
        cat > .env << 'EOF'
# BPL Tools Environment Variables
# These will be automatically prompted for when needed

# BPL API token for authentication
BPL_TOKEN=

# Path of Exile session ID from browser
POESESSID=

# Guild ID to monitor stash for
GUILD_ID=

# Private league ID to process invites for
PRIVATE_LEAGUE_ID=
EOF
        echo "✓ Created basic .env file"
    fi
    echo ""
    echo "You can manually edit the .env file if you want, but it's not required."
    echo "The application will prompt you for values as needed."
else
    echo "✓ .env file already exists"
fi

# Make the binary executable (for Linux/macOS)
if [ -f "bpl-tools" ]; then
    chmod +x bpl-tools
    echo "✓ Made bpl-tools executable"
elif [ -f "bpl-tools.exe" ]; then
    echo "✓ Windows executable detected"
fi

echo ""
echo "Setup complete! To run BPL Tools:"
if [ -f "bpl-tools" ]; then
    echo "  ./bpl-tools"
elif [ -f "bpl-tools.exe" ]; then
    echo "  bpl-tools.exe"
fi
echo ""
echo "The application will guide you through any required setup automatically!"
