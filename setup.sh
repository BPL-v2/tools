#!/bin/bash
# BPL Tools Setup Script

echo "BPL Tools Setup"
echo "==============="

# Check if .env file exists
if [ ! -f ".env" ]; then
    echo "Creating .env file from template..."
    cp .env.example .env
    echo "✓ Created .env file"
    echo ""
    echo "IMPORTANT: Please edit the .env file and add your actual values:"
    echo "  - BPL_TOKEN: Your BPL API token"
    echo "  - POESESSID: Your Path of Exile session ID"
    echo "  - GUILD_ID: Your guild ID"
    echo "  - PRIVATE_LEAGUE_ID: Your private league ID"
    echo ""
    echo "See the comments in .env for instructions on how to get these values."
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
echo "Remember to configure your .env file before running!"
