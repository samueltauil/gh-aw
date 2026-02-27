// @ts-check
/// <reference types="@actions/github-script" />

const { spawnSync } = require("child_process");

/**
 * GraphQL mutation to create a verified commit on a branch.
 * Commits created via this mutation are automatically signed and shown as verified
 * in the GitHub UI, unlike commits pushed via `git push` with GITHUB_TOKEN.
 */
const CREATE_COMMIT_ON_BRANCH_MUTATION = `
  mutation CreateVerifiedCommit(
    $repositoryNameWithOwner: String!
    $branchName: String!
    $expectedHeadOid: GitObjectID!
    $headline: String!
    $body: String
    $additions: [FileAddition!]!
    $deletions: [FileDeletion!]!
  ) {
    createCommitOnBranch(input: {
      branch: {
        repositoryNameWithOwner: $repositoryNameWithOwner
        branchName: $branchName
      }
      message: { headline: $headline, body: $body }
      fileChanges: {
        additions: $additions
        deletions: $deletions
      }
      expectedHeadOid: $expectedHeadOid
    }) {
      commit {
        oid
        url
      }
    }
  }
`;

/**
 * Read a file's raw content from the git object store at a specific commit.
 * Returns the content as a base64-encoded string, supporting both text and binary files.
 * Uses spawnSync without UTF-8 encoding to preserve binary content.
 *
 * @param {string} commitHash - The commit hash to read from
 * @param {string} filePath - Path to the file in the git tree
 * @returns {string} Base64-encoded file content
 */
function readFileAtCommit(commitHash, filePath) {
  const result = spawnSync("git", ["show", `${commitHash}:${filePath}`]);
  if (result.error) throw result.error;
  if (result.status !== 0) {
    const stderr = result.stderr ? result.stderr.toString() : "unknown error";
    throw new Error(`Failed to read "${filePath}" at commit ${commitHash}: ${stderr}`);
  }
  // result.stdout is a Buffer when no encoding is specified - safe for binary files
  const buf = Buffer.isBuffer(result.stdout) ? result.stdout : Buffer.from(result.stdout);
  return buf.toString("base64");
}

/**
 * Create a verified commit on a branch using the GitHub GraphQL API.
 * Commits created via this API are automatically signed and shown as verified
 * in the GitHub UI, unlike unverified commits created with `git push` and GITHUB_TOKEN.
 *
 * @param {Function} graphql - GitHub GraphQL client function (e.g. github.graphql or octokit.graphql)
 * @param {string} repositoryNameWithOwner - Repository in "owner/repo" format
 * @param {string} branchName - Target branch name (must already exist on remote)
 * @param {string} expectedHeadOid - Current HEAD OID of the remote branch
 * @param {string} headline - First line of the commit message
 * @param {string|null} body - Rest of the commit message (optional)
 * @param {Array<{path: string, contents: string}>} additions - Files to add/modify (contents base64-encoded)
 * @param {Array<{path: string}>} deletions - Files to delete
 * @returns {Promise<{oid: string, url: string}>} The created commit's OID and URL
 */
async function createVerifiedCommit(graphql, repositoryNameWithOwner, branchName, expectedHeadOid, headline, body, additions, deletions) {
  const result = await graphql(CREATE_COMMIT_ON_BRANCH_MUTATION, {
    repositoryNameWithOwner,
    branchName,
    expectedHeadOid,
    headline,
    body: body || undefined,
    additions: additions || [],
    deletions: deletions || [],
  });
  return result.createCommitOnBranch.commit;
}

/**
 * Push all local commits (since a given remote HEAD) to a remote branch
 * using the GitHub GraphQL API to produce verified/signed commits.
 *
 * The branch must already exist on the remote. Each local commit is translated
 * into a separate GraphQL commit preserving the commit message. File contents
 * are read directly from the git object store, supporting both text and binary files.
 *
 * @param {Function} graphql - GitHub GraphQL client function (github.graphql or octokit.graphql)
 * @param {string} repositoryNameWithOwner - Repository in "owner/repo" format
 * @param {string} branchName - Target branch name (must already exist on remote)
 * @param {string} remoteHead - Remote branch HEAD OID before local commits were applied
 * @param {Function} [_readFile] - Optional file reader override (used for testing)
 * @returns {Promise<{oid: string, url: string}>} The last created commit's OID and URL
 */
async function pushCommitsViaGraphQL(graphql, repositoryNameWithOwner, branchName, remoteHead, _readFile = readFileAtCommit) {
  if (!remoteHead) {
    throw new Error("remoteHead is required to push commits via GraphQL API");
  }

  // Get all local commits since remoteHead, oldest first (so we replay them in order)
  const { stdout: logOutput } = await exec.getExecOutput("git", ["log", "--format=%H", `${remoteHead}..HEAD`, "--reverse"]);
  const commitHashes = logOutput
    .trim()
    .split("\n")
    .filter(h => h.trim());

  if (commitHashes.length === 0) {
    throw new Error("No local commits found to push via GraphQL API");
  }

  core.info(`Pushing ${commitHashes.length} commit(s) via GraphQL API (verified commits)`);

  let expectedHeadOid = remoteHead;
  let lastCommit = null;

  for (const hash of commitHashes) {
    // Get commit subject (headline) and body separately
    const { stdout: subjectOut } = await exec.getExecOutput("git", ["log", "--format=%s", "-1", hash]);
    const { stdout: bodyOut } = await exec.getExecOutput("git", ["log", "--format=%b", "-1", hash]);

    const headline = subjectOut.trim();
    const body = bodyOut.trim() || null;

    // Get files changed in this commit: status (A/M/D/R/C) + paths
    const { stdout: diffOut } = await exec.getExecOutput("git", ["diff-tree", "--no-commit-id", "-r", "--name-status", hash]);

    const additions = [];
    const deletions = [];

    for (const line of diffOut
      .trim()
      .split("\n")
      .filter(l => l.trim())) {
      const parts = line.split("\t");
      const status = parts[0];

      if (status === "D") {
        // Deleted file
        deletions.push({ path: parts[1] });
      } else if (status.startsWith("R") || status.startsWith("C")) {
        // Renamed (R) or Copied (C): delete old path, add new path
        const oldPath = parts[1];
        const newPath = parts[2];
        additions.push({ path: newPath, contents: _readFile(hash, newPath) });
        if (status.startsWith("R")) {
          deletions.push({ path: oldPath });
        }
      } else {
        // Added (A) or Modified (M)
        additions.push({ path: parts[1], contents: _readFile(hash, parts[1]) });
      }
    }

    core.info(`Creating verified commit: "${headline}" (${additions.length} addition(s), ${deletions.length} deletion(s))`);

    const commit = await createVerifiedCommit(graphql, repositoryNameWithOwner, branchName, expectedHeadOid, headline, body, additions, deletions);
    core.info(`Verified commit created: ${commit.url}`);

    expectedHeadOid = commit.oid;
    lastCommit = commit;
  }

  return /** @type {{oid: string, url: string}} */ lastCommit;
}

module.exports = { createVerifiedCommit, pushCommitsViaGraphQL, readFileAtCommit };
