#!/bin/bash
set -e

URL="https://github.com/Dorero/migrator/releases/latest/download/migrator"
DEST="/usr/local/bin/migrator"

echo "Downloading migrator..."
curl -L -o migrator $URL
chmod +x migrator

if [ "$(id -u)" -eq 0 ]; then
    mv migrator $DEST
    echo "Installed migrator to $DEST"
else
    echo "You don't have root permissions. Moving binary to ~/bin"
    mkdir -p ~/bin
    mv migrator ~/bin/
    echo 'export PATH="$HOME/bin:$PATH"' >> ~/.bashrc
    echo "Please run: source ~/.bashrc"
fi
