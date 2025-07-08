#!/bin/bash
cd ~/Codes/LFX/kubeedge
source .env
export GEMINI_API_KEY GITHUB_TOKEN COVERAGE_THRESHOLD MAX_RETRY_ATTEMPTS DEBUG
exec "$@"
