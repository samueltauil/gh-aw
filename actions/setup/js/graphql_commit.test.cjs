import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

describe("graphql_commit.cjs", () => {
  let mockCore;
  let mockExec;
  let mockGraphql;
  let createVerifiedCommit;
  let pushCommitsViaGraphQL;

  beforeEach(() => {
    mockCore = {
      info: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),
    };

    mockExec = {
      exec: vi.fn().mockResolvedValue(0),
      getExecOutput: vi.fn(),
    };

    mockGraphql = vi.fn().mockResolvedValue({
      createCommitOnBranch: {
        commit: {
          oid: "abc123def456",
          url: "https://github.com/owner/repo/commit/abc123def456",
        },
      },
    });

    global.core = mockCore;
    global.exec = mockExec;

    delete require.cache[require.resolve("./graphql_commit.cjs")];
    ({ createVerifiedCommit, pushCommitsViaGraphQL } = require("./graphql_commit.cjs"));
  });

  afterEach(() => {
    delete global.core;
    delete global.exec;
    vi.clearAllMocks();
  });

  // ──────────────────────────────────────────────────────
  // createVerifiedCommit
  // ──────────────────────────────────────────────────────

  describe("createVerifiedCommit", () => {
    it("should call graphql with the correct mutation and variables", async () => {
      const result = await createVerifiedCommit(mockGraphql, "owner/repo", "feature-branch", "abc000", "feat: add feature", "Detailed description", [{ path: "src/file.js", contents: "Y29udGVudA==" }], [{ path: "old/file.js" }]);

      expect(mockGraphql).toHaveBeenCalledOnce();
      const [query, variables] = mockGraphql.mock.calls[0];
      expect(query).toContain("createCommitOnBranch");
      expect(variables).toMatchObject({
        repositoryNameWithOwner: "owner/repo",
        branchName: "feature-branch",
        expectedHeadOid: "abc000",
        headline: "feat: add feature",
        body: "Detailed description",
        additions: [{ path: "src/file.js", contents: "Y29udGVudA==" }],
        deletions: [{ path: "old/file.js" }],
      });
      expect(result).toEqual({ oid: "abc123def456", url: "https://github.com/owner/repo/commit/abc123def456" });
    });

    it("should omit body when null", async () => {
      await createVerifiedCommit(mockGraphql, "owner/repo", "main", "abc000", "fix: something", null, [], []);

      const [, variables] = mockGraphql.mock.calls[0];
      expect(variables.body).toBeUndefined();
    });

    it("should default additions and deletions to empty arrays when not provided", async () => {
      await createVerifiedCommit(mockGraphql, "owner/repo", "main", "abc000", "chore: empty commit", null, undefined, undefined);

      const [, variables] = mockGraphql.mock.calls[0];
      expect(variables.additions).toEqual([]);
      expect(variables.deletions).toEqual([]);
    });

    it("should return the commit oid and url from the graphql response", async () => {
      mockGraphql.mockResolvedValue({
        createCommitOnBranch: {
          commit: { oid: "deadbeef1234", url: "https://github.com/org/project/commit/deadbeef1234" },
        },
      });

      const commit = await createVerifiedCommit(mockGraphql, "org/project", "main", "head0", "msg", null, [], []);

      expect(commit.oid).toBe("deadbeef1234");
      expect(commit.url).toBe("https://github.com/org/project/commit/deadbeef1234");
    });

    it("should propagate graphql errors", async () => {
      mockGraphql.mockRejectedValue(new Error("GraphQL request failed"));

      await expect(createVerifiedCommit(mockGraphql, "owner/repo", "main", "abc000", "msg", null, [], [])).rejects.toThrow("GraphQL request failed");
    });
  });

  // ──────────────────────────────────────────────────────
  // pushCommitsViaGraphQL
  // ──────────────────────────────────────────────────────

  describe("pushCommitsViaGraphQL", () => {
    /** Mock file reader for testing (avoids actual git object store calls) */
    const mockReadFile = vi.fn().mockReturnValue("Y29udGVudA=="); // base64 "content"

    beforeEach(() => {
      mockReadFile.mockClear();
    });

    /**
     * Helper to set up exec.getExecOutput mock for a single commit.
     */
    function setupSingleCommit({ commitHash, subject, body = "", diffOutput }) {
      mockExec.getExecOutput.mockImplementation(async (cmd, args) => {
        if (cmd === "git" && args[0] === "log" && args[1] === "--format=%H") {
          return { stdout: commitHash, exitCode: 0 };
        }
        if (cmd === "git" && args[0] === "log" && args[1] === "--format=%s") {
          return { stdout: subject, exitCode: 0 };
        }
        if (cmd === "git" && args[0] === "log" && args[1] === "--format=%b") {
          return { stdout: body, exitCode: 0 };
        }
        if (cmd === "git" && args[0] === "diff-tree") {
          return { stdout: diffOutput, exitCode: 0 };
        }
        return { stdout: "", exitCode: 0 };
      });
    }

    it("should throw when remoteHead is empty", async () => {
      await expect(pushCommitsViaGraphQL(mockGraphql, "owner/repo", "main", "", mockReadFile)).rejects.toThrow("remoteHead is required");
    });

    it("should throw when no local commits are found", async () => {
      mockExec.getExecOutput.mockResolvedValue({ stdout: "", exitCode: 0 });

      await expect(pushCommitsViaGraphQL(mockGraphql, "owner/repo", "main", "base-sha", mockReadFile)).rejects.toThrow("No local commits found");
    });

    it("should push a single added file as a verified commit", async () => {
      setupSingleCommit({
        commitHash: "commitabc",
        subject: "feat: add new file",
        diffOutput: "A\tsrc/new-file.js",
      });

      const result = await pushCommitsViaGraphQL(mockGraphql, "owner/repo", "feature", "remoteHead0", mockReadFile);

      expect(mockGraphql).toHaveBeenCalledOnce();
      const [, variables] = mockGraphql.mock.calls[0];
      expect(variables.repositoryNameWithOwner).toBe("owner/repo");
      expect(variables.branchName).toBe("feature");
      expect(variables.expectedHeadOid).toBe("remoteHead0");
      expect(variables.headline).toBe("feat: add new file");
      expect(variables.additions).toHaveLength(1);
      expect(variables.additions[0].path).toBe("src/new-file.js");
      expect(variables.deletions).toHaveLength(0);
      expect(result.oid).toBe("abc123def456");
      expect(mockReadFile).toHaveBeenCalledWith("commitabc", "src/new-file.js");
    });

    it("should handle file deletion without reading file content", async () => {
      setupSingleCommit({
        commitHash: "commitdel",
        subject: "chore: remove old file",
        diffOutput: "D\tsrc/old-file.js",
      });

      await pushCommitsViaGraphQL(mockGraphql, "owner/repo", "main", "remoteHead0", mockReadFile);

      const [, variables] = mockGraphql.mock.calls[0];
      expect(variables.additions).toHaveLength(0);
      expect(variables.deletions).toHaveLength(1);
      expect(variables.deletions[0].path).toBe("src/old-file.js");
      // readFile should not be called for deletions
      expect(mockReadFile).not.toHaveBeenCalled();
    });

    it("should handle renamed files (delete old path, add new path)", async () => {
      setupSingleCommit({
        commitHash: "commitren",
        subject: "refactor: rename file",
        diffOutput: "R100\tsrc/old.js\tsrc/new.js",
      });

      await pushCommitsViaGraphQL(mockGraphql, "owner/repo", "main", "remoteHead0", mockReadFile);

      const [, variables] = mockGraphql.mock.calls[0];
      expect(variables.additions).toHaveLength(1);
      expect(variables.additions[0].path).toBe("src/new.js");
      expect(variables.deletions).toHaveLength(1);
      expect(variables.deletions[0].path).toBe("src/old.js");
      expect(mockReadFile).toHaveBeenCalledWith("commitren", "src/new.js");
    });

    it("should push multiple commits in order (oldest first), chaining expectedHeadOid", async () => {
      const commits = [
        { hash: "commit001", subject: "first commit", diff: "A\tfile1.js" },
        { hash: "commit002", subject: "second commit", diff: "M\tfile1.js" },
      ];

      mockExec.getExecOutput.mockImplementation(async (cmd, args) => {
        if (cmd === "git" && args[0] === "log" && args[1] === "--format=%H") {
          return { stdout: commits.map(c => c.hash).join("\n"), exitCode: 0 };
        }
        if (cmd === "git" && args[0] === "log" && args[1] === "--format=%s") {
          const hash = args[args.length - 1];
          const commit = commits.find(c => c.hash === hash);
          return { stdout: commit ? commit.subject : "", exitCode: 0 };
        }
        if (cmd === "git" && args[0] === "log" && args[1] === "--format=%b") {
          return { stdout: "", exitCode: 0 };
        }
        if (cmd === "git" && args[0] === "diff-tree") {
          const hash = args[args.length - 1];
          const commit = commits.find(c => c.hash === hash);
          return { stdout: commit ? commit.diff : "", exitCode: 0 };
        }
        return { stdout: "", exitCode: 0 };
      });

      mockGraphql
        .mockResolvedValueOnce({ createCommitOnBranch: { commit: { oid: "oid001", url: "https://github.com/c/oid001" } } })
        .mockResolvedValueOnce({ createCommitOnBranch: { commit: { oid: "oid002", url: "https://github.com/c/oid002" } } });

      const result = await pushCommitsViaGraphQL(mockGraphql, "owner/repo", "main", "base-oid", mockReadFile);

      expect(mockGraphql).toHaveBeenCalledTimes(2);

      // First commit uses base-oid as expectedHeadOid
      const [, firstVars] = mockGraphql.mock.calls[0];
      expect(firstVars.expectedHeadOid).toBe("base-oid");
      expect(firstVars.headline).toBe("first commit");

      // Second commit chains from the first commit's OID
      const [, secondVars] = mockGraphql.mock.calls[1];
      expect(secondVars.expectedHeadOid).toBe("oid001");
      expect(secondVars.headline).toBe("second commit");

      // Returns the last commit
      expect(result.oid).toBe("oid002");
    });

    it("should include commit body when present", async () => {
      setupSingleCommit({
        commitHash: "commitbody",
        subject: "feat: feature with body",
        body: "This is the commit body\n\nWith multiple paragraphs.",
        diffOutput: "A\tfile.txt",
      });

      await pushCommitsViaGraphQL(mockGraphql, "owner/repo", "main", "remoteHead0", mockReadFile);

      const [, variables] = mockGraphql.mock.calls[0];
      expect(variables.body).toBe("This is the commit body\n\nWith multiple paragraphs.");
    });

    it("should omit body when commit has no body", async () => {
      setupSingleCommit({
        commitHash: "commitnobody",
        subject: "fix: quick fix",
        body: "",
        diffOutput: "M\tfile.txt",
      });

      await pushCommitsViaGraphQL(mockGraphql, "owner/repo", "main", "remoteHead0", mockReadFile);

      const [, variables] = mockGraphql.mock.calls[0];
      expect(variables.body).toBeUndefined();
    });

    it("should propagate graphql errors", async () => {
      setupSingleCommit({
        commitHash: "commit000",
        subject: "some commit",
        diffOutput: "A\tfile.txt",
      });

      mockGraphql.mockRejectedValue(new Error("Branch not found"));

      await expect(pushCommitsViaGraphQL(mockGraphql, "owner/repo", "main", "remoteHead0", mockReadFile)).rejects.toThrow("Branch not found");
    });

    it("should propagate readFile errors", async () => {
      setupSingleCommit({
        commitHash: "commitfail",
        subject: "some commit",
        diffOutput: "A\tfile.txt",
      });

      const failingReadFile = vi.fn().mockImplementation(() => {
        throw new Error("git object not found");
      });

      await expect(pushCommitsViaGraphQL(mockGraphql, "owner/repo", "main", "remoteHead0", failingReadFile)).rejects.toThrow("git object not found");
    });
  });
});
