#!/bin/bash
# SocialPilot Build Script
# Usage: ./build.sh [linux|windows|all|clean]

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Project info
VERSION=${VERSION:-"1.5.0"}
BUILD_DIR="build"
BINARY_NAME="socialpilot"

echo -e "${BLUE}================================${NC}"
echo -e "${BLUE}  SocialPilot Build Script${NC}"
echo -e "${BLUE}================================${NC}"
echo ""

# Function to build frontend
build_frontend() {
    echo -e "${YELLOW}[1/3] Building Web UI...${NC}"
    if [ -d "webui" ]; then
        cd webui
        if [ ! -d "node_modules" ]; then
            echo -e "${YELLOW}Installing dependencies...${NC}"
            npm install
        fi
        npm run build
        cd ..
        echo -e "${GREEN}✓ Web UI built successfully${NC}"
    else
        echo -e "${RED}Error: webui directory not found${NC}"
        exit 1
    fi
}

# Function to build binary
build_binary() {
    local os=$1
    local arch=${2:-"amd64"}
    local output_name="${BINARY_NAME}-${os}-${arch}"

    if [ "$os" = "windows" ]; then
        output_name="${output_name}.exe"
    fi

    echo -e "${YELLOW}[2/3] Building for ${os}/${arch}...${NC}"

    mkdir -p "$BUILD_DIR"

    # Static build with CGO_ENABLED=0
    CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build \
        -ldflags="-s -w -X main.Version=${VERSION}" \
        -o "${BUILD_DIR}/${output_name}" \
        .

    if [ $? -eq 0 ]; then
        local size=$(ls -lh "${BUILD_DIR}/${output_name}" | awk '{print $5}')
        echo -e "${GREEN}✓ Built ${output_name} (${size})${NC}"
    else
        echo -e "${RED}✗ Failed to build ${output_name}${NC}"
        exit 1
    fi
}

# Function to clean build artifacts
clean_build() {
    echo -e "${YELLOW}Cleaning build artifacts...${NC}"
    rm -rf "$BUILD_DIR"
    rm -f "$BINARY_NAME" "$BINARY_NAME.exe"
    echo -e "${GREEN}✓ Cleaned successfully${NC}"
}

# Main build process
main() {
    local target=${1:-"all"}

    echo -e "${BLUE}Target: ${target}${NC}"
    echo ""

    case "$target" in
        linux)
            build_frontend
            build_binary "linux" "amd64"
            ;;
        windows)
            build_frontend
            build_binary "windows" "amd64"
            ;;
        all)
            build_frontend
            build_binary "linux" "amd64"
            build_binary "windows" "amd64"
            ;;
        clean)
            clean_build
            exit 0
            ;;
        *)
            echo -e "${RED}Error: Unknown target '${target}'${NC}"
            echo "Usage: $0 [linux|windows|all|clean]"
            exit 1
            ;;
    esac

    echo ""
    echo -e "${GREEN}[3/3] Build Summary:${NC}"
    echo -e "${BLUE}================================${NC}"
    ls -lh "$BUILD_DIR" 2>/dev/null | grep "$BINARY_NAME" || echo "No binaries found"
    echo -e "${BLUE}================================${NC}"
    echo ""
    echo -e "${GREEN}✓ Build completed successfully!${NC}"
    echo -e "Binaries are in the ${YELLOW}${BUILD_DIR}/${NC} directory"
}

# Run main function
main "$@"
