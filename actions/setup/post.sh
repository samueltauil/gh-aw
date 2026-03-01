#!/usr/bin/env bash
#
# post.sh - Cleanup script for the gh-aw setup action
#
# This script is run as a post-step by the setup action to remove
# temporary files generated during the execution of the agent.
# It completely erases the /tmp/gh-aw/ directory.
#
# Exit codes:
#   0 - Success (directory removed or did not exist)
#   1 - Error (failed to remove directory)

set -euo pipefail

GH_AW_TMP_DIR="/tmp/gh-aw"

echo "Cleaning up ${GH_AW_TMP_DIR} directory..."

# Validate the target path to prevent accidental removal of unintended directories
if [[ "${GH_AW_TMP_DIR}" != /tmp/* ]]; then
  echo "ERROR: Refusing to remove directory outside /tmp: ${GH_AW_TMP_DIR}"
  exit 1
fi

if [ -d "${GH_AW_TMP_DIR}" ]; then
  rm -rf "${GH_AW_TMP_DIR}"
  echo "✓ Removed ${GH_AW_TMP_DIR}"
else
  echo "${GH_AW_TMP_DIR} does not exist, nothing to clean up"
fi
