// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Constants
 *
 * This module provides shared constants used across JavaScript actions.
 * These constants should be kept in sync with the constants in pkg/constants/constants.go
 */

/**
 * AgentOutputFilename is the filename of the agent output JSON file
 * @type {string}
 */
const AGENT_OUTPUT_FILENAME = "agent_output.json";

/**
 * Base path for temporary gh-aw files
 * @type {string}
 */
const TMP_GH_AW_PATH = "/tmp/gh-aw";

/**
 * Convert megabytes to bytes
 * @param {number} mb - Size in megabytes
 * @returns {number} Size in bytes
 */
function megabytes(mb) {
  return mb * 1024 * 1024;
}

/**
 * Maximum buffer size for general exec operations (10MB)
 * @type {number}
 */
const MAX_BUFFER_SIZE = megabytes(10);

/**
 * Maximum buffer size for git patch operations (200MB)
 * Handles large commits with binary files such as images.
 * @type {number}
 */
const GIT_PATCH_MAX_BUFFER_SIZE = megabytes(200);

module.exports = {
  AGENT_OUTPUT_FILENAME,
  TMP_GH_AW_PATH,
  megabytes,
  MAX_BUFFER_SIZE,
  GIT_PATCH_MAX_BUFFER_SIZE,
};
