#!/bin/bash
set -e

echo "=================================="
echo "Deployment Validation Script"
echo "=================================="
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check functions
check_command() {
    if command -v "$1" &> /dev/null; then
        echo -e "${GREEN}✓${NC} $1 is installed"
        return 0
    else
        echo -e "${RED}✗${NC} $1 is NOT installed"
        return 1
    fi
}

# Docker Compose Validation
echo "Docker Compose Validation"
echo "-------------------------"

if check_command "docker"; then
    docker_version=$(docker --version)
    echo "  Version: $docker_version"

    # Check if Docker daemon is running
    if docker info &> /dev/null; then
        echo -e "  ${GREEN}✓${NC} Docker daemon is running"
    else
        echo -e "  ${RED}✗${NC} Docker daemon is NOT running"
    fi
else
    echo -e "  ${YELLOW}!${NC} Docker Compose validation skipped"
fi

if [ -f "docker-compose.yml" ]; then
    echo -e "${GREEN}✓${NC} docker-compose.yml exists"

    # Validate syntax
    if docker compose config > /dev/null 2>&1; then
        echo -e "${GREEN}✓${NC} docker-compose.yml syntax is valid"
    else
        echo -e "${RED}✗${NC} docker-compose.yml has syntax errors"
    fi
else
    echo -e "${RED}✗${NC} docker-compose.yml NOT found"
fi

if [ -f ".env.example" ]; then
    echo -e "${GREEN}✓${NC} .env.example exists"
else
    echo -e "${RED}✗${NC} .env.example NOT found"
fi

if [ -f "deploy/configs/generator.yaml" ]; then
    echo -e "${GREEN}✓${NC} deploy/configs/generator.yaml exists"
else
    echo -e "${RED}✗${NC} deploy/configs/generator.yaml NOT found"
fi

if [ -f "deploy/configs/sender.yaml" ]; then
    echo -e "${GREEN}✓${NC} deploy/configs/sender.yaml exists"
else
    echo -e "${RED}✗${NC} deploy/configs/sender.yaml NOT found"
fi

echo ""

# Helm Chart Validation
echo "Helm Chart Validation"
echo "---------------------"

if check_command "helm"; then
    helm_version=$(helm version --short)
    echo "  Version: $helm_version"
else
    echo -e "  ${YELLOW}!${NC} Helm chart validation skipped"
fi

if [ -d "helm/telemetry-gen-and-send" ]; then
    echo -e "${GREEN}✓${NC} Helm chart directory exists"

    # Validate Chart.yaml
    if [ -f "helm/telemetry-gen-and-send/Chart.yaml" ]; then
        echo -e "${GREEN}✓${NC} Chart.yaml exists"
    else
        echo -e "${RED}✗${NC} Chart.yaml NOT found"
    fi

    # Validate values.yaml
    if [ -f "helm/telemetry-gen-and-send/values.yaml" ]; then
        echo -e "${GREEN}✓${NC} values.yaml exists"
    else
        echo -e "${RED}✗${NC} values.yaml NOT found"
    fi

    # Validate templates directory
    if [ -d "helm/telemetry-gen-and-send/templates" ]; then
        echo -e "${GREEN}✓${NC} templates/ directory exists"

        required_templates=(
            "_helpers.tpl"
            "deployment.yaml"
            "configmap-generator.yaml"
            "configmap-sender.yaml"
            "secret.yaml"
            "serviceaccount.yaml"
            "pvc.yaml"
            "NOTES.txt"
        )

        for template in "${required_templates[@]}"; do
            if [ -f "helm/telemetry-gen-and-send/templates/$template" ]; then
                echo -e "  ${GREEN}✓${NC} templates/$template exists"
            else
                echo -e "  ${RED}✗${NC} templates/$template NOT found"
            fi
        done
    else
        echo -e "${RED}✗${NC} templates/ directory NOT found"
    fi

    # Run helm lint if helm is available
    if command -v helm &> /dev/null; then
        echo ""
        echo "Running helm lint..."
        lint_output=$(helm lint helm/telemetry-gen-and-send 2>&1)
        if echo "$lint_output" | grep -q "0 chart(s) failed"; then
            echo -e "${GREEN}✓${NC} Helm lint passed"
        else
            echo -e "${RED}✗${NC} Helm lint found errors"
        fi
    fi
else
    echo -e "${RED}✗${NC} Helm chart directory NOT found"
fi

echo ""

# Kubernetes validation
echo "Kubernetes Validation"
echo "--------------------"

if check_command "kubectl"; then
    kubectl_version=$(kubectl version --client --short 2>/dev/null || kubectl version --client)
    echo "  Version: $kubectl_version"

    # Check cluster connectivity
    if kubectl cluster-info &> /dev/null; then
        echo -e "  ${GREEN}✓${NC} Connected to Kubernetes cluster"
        cluster_info=$(kubectl config current-context)
        echo "  Context: $cluster_info"
    else
        echo -e "  ${YELLOW}!${NC} Not connected to a Kubernetes cluster"
    fi
else
    echo -e "  ${YELLOW}!${NC} Kubernetes validation skipped"
fi

echo ""

# Documentation validation
echo "Documentation Validation"
echo "-----------------------"

if [ -f "README.md" ]; then
    echo -e "${GREEN}✓${NC} README.md exists"

    # Check for Docker Compose section
    if grep -q "Docker Compose" README.md; then
        echo -e "  ${GREEN}✓${NC} Docker Compose section found"
    else
        echo -e "  ${RED}✗${NC} Docker Compose section NOT found"
    fi

    # Check for Helm section
    if grep -q "Helm" README.md; then
        echo -e "  ${GREEN}✓${NC} Helm section found"
    else
        echo -e "  ${RED}✗${NC} Helm section NOT found"
    fi
else
    echo -e "${RED}✗${NC} README.md NOT found"
fi

if [ -f "DEPLOYMENT.md" ]; then
    echo -e "${GREEN}✓${NC} DEPLOYMENT.md exists"
else
    echo -e "${RED}✗${NC} DEPLOYMENT.md NOT found"
fi

if [ -f "helm/telemetry-gen-and-send/README.md" ]; then
    echo -e "${GREEN}✓${NC} Helm chart README.md exists"
else
    echo -e "${RED}✗${NC} Helm chart README.md NOT found"
fi

echo ""
echo "=================================="
echo "Validation Complete!"
echo "=================================="
echo ""
echo "Quick Start Commands:"
echo ""
echo "Docker Compose:"
echo "  cp .env.example .env"
echo "  # Edit .env and set HONEYCOMB_API_KEY"
echo "  docker compose up --scale sender=5"
echo ""
echo "Helm Chart:"
echo "  helm install test ./helm/telemetry-gen-and-send \\"
echo "    --set honeycomb.apiKey=\"your-api-key\""
echo ""
