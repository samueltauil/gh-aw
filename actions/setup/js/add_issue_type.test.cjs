// @ts-check
import { describe, it, expect, beforeEach } from "vitest";
const { main } = require("./add_issue_type.cjs");

describe("add_issue_type", () => {
  let mockCore;
  let mockGithub;
  let mockContext;

  beforeEach(() => {
    // Reset mocks before each test
    mockCore = {
      info: () => {},
      warning: () => {},
      error: () => {},
      messages: [],
      infos: [],
      warnings: [],
      errors: [],
    };

    // Capture all logged messages
    mockCore.info = msg => {
      mockCore.infos.push(msg);
      mockCore.messages.push({ level: "info", message: msg });
    };
    mockCore.warning = msg => {
      mockCore.warnings.push(msg);
      mockCore.messages.push({ level: "warning", message: msg });
    };
    mockCore.error = msg => {
      mockCore.errors.push(msg);
      mockCore.messages.push({ level: "error", message: msg });
    };

    mockGithub = {
      rest: {
        issues: {
          update: async () => ({}),
        },
      },
    };

    mockContext = {
      repo: {
        owner: "test-owner",
        repo: "test-repo",
      },
      payload: {
        issue: {
          number: 123,
        },
      },
    };

    // Set globals
    global.core = mockCore;
    global.github = mockGithub;
    global.context = mockContext;
  });

  describe("main factory", () => {
    it("should create a handler function with default configuration", async () => {
      const handler = await main();
      expect(typeof handler).toBe("function");
    });

    it("should create a handler function with custom configuration", async () => {
      const handler = await main({
        allowed: ["Bug", "Feature"],
        max: 5,
      });
      expect(typeof handler).toBe("function");
    });

    it("should log configuration on initialization", async () => {
      await main({ allowed: ["Bug", "Feature"], max: 3 });
      expect(mockCore.infos.some(msg => msg.includes("max=3"))).toBe(true);
      expect(mockCore.infos.some(msg => msg.includes("Bug, Feature"))).toBe(true);
    });
  });

  describe("handleAddIssueType", () => {
    it("should set issue type using explicit item_number", async () => {
      const handler = await main({ max: 10 });
      const updateCalls = [];

      mockGithub.rest.issues.update = async params => {
        updateCalls.push(params);
        return {};
      };

      const result = await handler(
        {
          item_number: 456,
          issue_type: "Bug",
        },
        {}
      );

      expect(result.success).toBe(true);
      expect(result.number).toBe(456);
      expect(result.issueTypeSet).toBe("Bug");
      expect(updateCalls.length).toBe(1);
      expect(updateCalls[0].issue_number).toBe(456);
      expect(updateCalls[0].type).toBe("Bug");
    });

    it("should set issue type from context when item_number not provided", async () => {
      const handler = await main({ max: 10 });
      const updateCalls = [];

      mockGithub.rest.issues.update = async params => {
        updateCalls.push(params);
        return {};
      };

      const result = await handler(
        {
          issue_type: "Feature",
        },
        {}
      );

      expect(result.success).toBe(true);
      expect(result.number).toBe(123);
      expect(result.issueTypeSet).toBe("Feature");
    });

    it("should trim whitespace from issue_type", async () => {
      const handler = await main({ max: 10 });
      const updateCalls = [];

      mockGithub.rest.issues.update = async params => {
        updateCalls.push(params);
        return {};
      };

      const result = await handler(
        {
          item_number: 100,
          issue_type: "  Bug  ",
        },
        {}
      );

      expect(result.success).toBe(true);
      expect(result.issueTypeSet).toBe("Bug");
      expect(updateCalls[0].type).toBe("Bug");
    });

    it("should handle invalid item_number", async () => {
      const handler = await main({ max: 10 });

      const result = await handler(
        {
          item_number: "invalid",
          issue_type: "Bug",
        },
        {}
      );

      expect(result.success).toBe(false);
      expect(result.error).toContain("Invalid item number");
    });

    it("should handle missing item_number and no context", async () => {
      mockContext.payload = {};

      const handler = await main({ max: 10 });

      const result = await handler(
        {
          issue_type: "Bug",
        },
        {}
      );

      expect(result.success).toBe(false);
      expect(result.error).toContain("No issue number available");
    });

    it("should handle missing issue_type", async () => {
      const handler = await main({ max: 10 });

      const result = await handler(
        {
          item_number: 100,
        },
        {}
      );

      expect(result.success).toBe(false);
      expect(result.error).toContain("issue_type is required");
    });

    it("should handle empty issue_type", async () => {
      const handler = await main({ max: 10 });

      const result = await handler(
        {
          item_number: 100,
          issue_type: "   ",
        },
        {}
      );

      expect(result.success).toBe(false);
      expect(result.error).toContain("issue_type is required");
    });

    it("should respect max count limit", async () => {
      const handler = await main({ max: 2 });

      const result1 = await handler({ item_number: 1, issue_type: "Bug" }, {});
      expect(result1.success).toBe(true);

      const result2 = await handler({ item_number: 2, issue_type: "Feature" }, {});
      expect(result2.success).toBe(true);

      const result3 = await handler({ item_number: 3, issue_type: "Task" }, {});
      expect(result3.success).toBe(false);
      expect(result3.error).toContain("Max count");
    });

    it("should filter issue types based on allowed list", async () => {
      const handler = await main({
        allowed: ["Bug", "Feature"],
        max: 10,
      });

      const result = await handler(
        {
          item_number: 100,
          issue_type: "UnknownType",
        },
        {}
      );

      expect(result.success).toBe(false);
      expect(result.error).toContain("not in the allowed list");
    });

    it("should allow issue types matching allowed list (case-insensitive)", async () => {
      const handler = await main({
        allowed: ["Bug", "Feature"],
        max: 10,
      });

      const updateCalls = [];
      mockGithub.rest.issues.update = async params => {
        updateCalls.push(params);
        return {};
      };

      const result = await handler(
        {
          item_number: 100,
          issue_type: "bug",
        },
        {}
      );

      expect(result.success).toBe(true);
    });

    it("should handle API errors gracefully", async () => {
      const handler = await main({ max: 10 });

      mockGithub.rest.issues.update = async () => {
        throw new Error("API Error: Not found");
      };

      const result = await handler(
        {
          item_number: 100,
          issue_type: "Bug",
        },
        {}
      );

      expect(result.success).toBe(false);
      expect(result.error).toContain("API Error");
    });

    it("should support target-repo from config", async () => {
      const handler = await main({
        max: 10,
        "target-repo": "external-org/external-repo",
      });
      const updateCalls = [];

      mockGithub.rest.issues.update = async params => {
        updateCalls.push(params);
        return {};
      };

      const result = await handler(
        {
          item_number: 100,
          issue_type: "Bug",
        },
        {}
      );

      expect(result.success).toBe(true);
      expect(updateCalls[0].owner).toBe("external-org");
      expect(updateCalls[0].repo).toBe("external-repo");
    });

    it("should use context.repo owner/repo for API call", async () => {
      const handler = await main({ max: 10 });
      const updateCalls = [];

      mockGithub.rest.issues.update = async params => {
        updateCalls.push(params);
        return {};
      };

      await handler({ item_number: 100, issue_type: "Bug" }, {});

      expect(updateCalls[0].owner).toBe("test-owner");
      expect(updateCalls[0].repo).toBe("test-repo");
    });
  });
});
