#!/bin/bash
set -e

# Threader Dependency Installer
# This script ensures system dependencies are met before building.

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"

install_linux_deps() {
    echo "Checking for Tesseract and Leptonica dependencies..."
    if command -v apt-get >/dev/null 2>&1; then
        echo "Detected Debian/Ubuntu-based system."
        MISSING_DEPS=()
        dpkg -s libtesseract-dev >/dev/null 2>&1 || MISSING_DEPS+=("libtesseract-dev")
        dpkg -s libleptonica-dev >/dev/null 2>&1 || MISSING_DEPS+=("libleptonica-dev")
        dpkg -s tesseract-ocr >/dev/null 2>&1 || MISSING_DEPS+=("tesseract-ocr")
        dpkg -s pkg-config >/dev/null 2>&1 || MISSING_DEPS+=("pkg-config")
        dpkg -s g++ >/dev/null 2>&1 || MISSING_DEPS+=("g++")

        if [ ${#MISSING_DEPS[@]} -gt 0 ]; then
            echo "Installing missing dependencies: ${MISSING_DEPS[*]}"
            sudo apt-get update
            sudo apt-get install -y "${MISSING_DEPS[@]}"
        else
            echo "All system dependencies are already installed."
        fi
    elif command -v pacman >/dev/null 2>&1; then
        echo "Detected Arch-based system."
        sudo pacman -Sy --needed --noconfirm tesseract tesseract-data-eng leptonica pkg-config gcc
    else
        echo "Unsupported Linux distribution for automatic dependency installation."
        echo "Please manually install tesseract-devel and leptonica-devel packages."
    fi
}

install_darwin_deps() {
    echo "Checking for Tesseract dependencies via Homebrew..."
    if command -v brew >/dev/null 2>&1; then
        brew install tesseract leptonica pkg-config
    else
        echo "Homebrew not found. Please install Homebrew or manually install tesseract and leptonica."
        exit 1
    fi
}

case "$OS" in
    linux) install_linux_deps ;;
    darwin) install_darwin_deps ;;
    *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

echo "Dependencies satisfied. Proceeding with build..."
go build -o threader ./cmd/threader
