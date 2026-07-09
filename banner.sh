#!/bin/bash
# banner.sh — trade-kit ASCII art banner
# Usage: bash banner.sh
#   or:  source banner.sh  (to use in other scripts)

BOLD='\033[1m'
CYAN='\033[1;36m'
DIM='\033[2m'
RESET='\033[0m'

echo -e "${CYAN}"
cat << 'BANNER'
  ┌─────────────────────────────────────────────────────────────┐
  │                                                             │
  │   ████████╗██████╗  █████╗ ██████╗ ███████╗                 │
  │   ╚══██╔══╝██╔══██╗██╔══██╗██╔══██╗██╔════╝                │
  │      ██║   ██████╔╝███████║██║  ██║█████╗                   │
  │      ██║   ██╔══██╗██╔══██║██║  ██║██╔══╝                   │
  │      ██║   ██║  ██║██║  ██║██████╔╝███████╗                 │
  │      ╚═╝   ╚═╝  ╚═╝╚═╝  ╚═╝╚═════╝ ╚══════╝                │
  │                                                             │
  │   ██╗  ██╗██╗████████╗                                      │
  │   ██║ ██╔╝██║╚══██╔══╝                                     │
  │   █████╔╝ ██║   ██║                                        │
  │   ██╔═██╗ ██║   ██║                                        │
  │   ██║  ██╗██║   ██║                                        │
  │   ╚═╝  ╚═╝╚═╝   ╚═╝                                       │
  │                                                             │
  │   Multi-broker CLI toolkit for retail traders               │
  │   Tiger · Moomoo · eToro · 15 tools · Paper mode default   │
  │                                                             │
BANNER
echo -e "  │   v$(cat "$(dirname "$0")/VERSION" 2>/dev/null || echo '0.7.0')              github.com/Orbital-Trade/trade-kit    │"
cat << 'BANNER'
  │                                                             │
  └─────────────────────────────────────────────────────────────┘
BANNER
echo -e "${RESET}"
