// @ts-check
/// <reference types="@actions/github-script" />

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

/** @type {string} Safe output type handled by this module */
const HANDLER_TYPE = "add_issue_type";

const { getErrorMessage } = require("./error_helpers.cjs");
const { resolveTargetRepoConfig, resolveAndValidateRepo } = require("./repo_helpers.cjs");
const { logStagedPreviewInfo } = require("./staged_preview.cjs");
const { createAuthenticatedGitHubClient } = require("./handler_auth.cjs");

/**
 * Main handler factory for add_issue_type
 * Returns a message handler function that processes individual add_issue_type messages
 * @type {HandlerFactoryFunction}
 */
async function main(config = {}) {
  // Extract configuration
  const allowedTypes = config.allowed || [];
  const maxCount = config.max || 10;
  const { defaultTargetRepo, allowedRepos } = resolveTargetRepoConfig(config);
  const authClient = await createAuthenticatedGitHubClient(config);

  // Check if we're in staged mode
  const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true";

  core.info(`Add issue type configuration: max=${maxCount}`);
  if (allowedTypes.length > 0) {
    core.info(`Allowed issue types: ${allowedTypes.join(", ")}`);
  }
  core.info(`Default target repo: ${defaultTargetRepo}`);
  if (allowedRepos.size > 0) {
    core.info(`Allowed repos: ${[...allowedRepos].join(", ")}`);
  }

  // Track how many items we've processed for max limit
  let processedCount = 0;

  /**
   * Message handler function that processes a single add_issue_type message
   * @param {Object} message - The add_issue_type message to process
   * @param {Object} resolvedTemporaryIds - Map of temporary IDs to {repo, number}
   * @returns {Promise<Object>} Result with success/error status
   */
  return async function handleAddIssueType(message, resolvedTemporaryIds) {
    // Check if we've hit the max limit
    if (processedCount >= maxCount) {
      core.warning(`Skipping add_issue_type: max count of ${maxCount} reached`);
      return {
        success: false,
        error: `Max count of ${maxCount} reached`,
      };
    }

    processedCount++;

    // Resolve and validate target repository
    const repoResult = resolveAndValidateRepo(message, defaultTargetRepo, allowedRepos, "issue");
    if (!repoResult.success) {
      core.warning(`Skipping add_issue_type: ${repoResult.error}`);
      return {
        success: false,
        error: repoResult.error,
      };
    }
    const { repo: itemRepo, repoParts } = repoResult;
    core.info(`Target repository: ${itemRepo}`);

    // Determine target issue number
    const itemNumber = message.item_number !== undefined ? parseInt(String(message.item_number), 10) : context.payload?.issue?.number;

    if (!itemNumber || isNaN(itemNumber)) {
      const error = message.item_number !== undefined ? `Invalid item number: ${message.item_number}` : "No issue number available";
      core.warning(error);
      return { success: false, error };
    }

    const issueType = message.issue_type;

    if (!issueType || typeof issueType !== "string" || issueType.trim() === "") {
      const error = "issue_type is required and must be a non-empty string";
      core.warning(error);
      return { success: false, error };
    }

    const trimmedIssueType = issueType.trim();

    // Validate against allowed types if configured
    if (allowedTypes.length > 0) {
      const normalizedAllowed = allowedTypes.map(t => t.toLowerCase());
      if (!normalizedAllowed.includes(trimmedIssueType.toLowerCase())) {
        const error = `Issue type "${trimmedIssueType}" is not in the allowed list: ${JSON.stringify(allowedTypes)}`;
        core.warning(error);
        return { success: false, error };
      }
    }

    core.info(`Setting issue type "${trimmedIssueType}" on issue #${itemNumber} in ${itemRepo}`);

    // If in staged mode, preview without making API calls
    if (isStaged) {
      logStagedPreviewInfo(`Would set issue type "${trimmedIssueType}" on issue #${itemNumber} in ${itemRepo}`);
      return {
        success: true,
        staged: true,
        previewInfo: {
          number: itemNumber,
          repo: itemRepo,
          issueType: trimmedIssueType,
        },
      };
    }

    try {
      await authClient.rest.issues.update({
        owner: repoParts.owner,
        repo: repoParts.repo,
        issue_number: itemNumber,
        type: trimmedIssueType,
      });

      core.info(`Successfully set issue type "${trimmedIssueType}" on issue #${itemNumber} in ${itemRepo}`);
      return {
        success: true,
        number: itemNumber,
        issueTypeSet: trimmedIssueType,
      };
    } catch (error) {
      const errorMessage = getErrorMessage(error);
      core.error(`Failed to set issue type: ${errorMessage}`);
      return { success: false, error: errorMessage };
    }
  };
}

module.exports = { main };
