// @ts-check
import { describe, it, expect, beforeEach } from "vitest";
const { main } = require("./remove_issue_type.cjs");

describe("remove_issue_type", () => {
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
      const handler = await main({ max: 5 });
      expect(typeof handler).toBe("function");
    });

    it("should log configuration on initialization", async () => {
      await main({ max: 3 });
      expect(mockCore.infos.some(msg => msg.includes("max=3"))).toBe(true);
    });
  });

  describe("handleRemoveIssueType", () => {
    it("should remove issue type using explicit item_number", async () => {
      const handler = await main({ max: 10 });
      const updateCalls = [];

      mockGithub.rest.issues.update = async params => {
        updateCalls.push(params);
        return {};
      };

      const result = await handler(
        {
          item_number: 456,
        },
        {}
      );

      expect(result.success).toBe(true);
      expect(result.number).toBe(456);
      expect(updateCalls.length).toBe(1);
      expect(updateCalls[0].issue_number).toBe(456);
      expect(updateCalls[0].type).toBeNull();
    });

    it("should remove issue type from context when item_number not provided", async () => {
      const handler = await main({ max: 10 });
      const updateCalls = [];

      mockGithub.rest.issues.update = async params => {
        updateCalls.push(params);
        return {};
      };

      const result = await handler({}, {});

      expect(result.success).toBe(true);
      expect(result.number).toBe(123);
      expect(updateCalls[0].type).toBeNull();
    });

    it("should handle invalid item_number", async () => {
      const handler = await main({ max: 10 });

      const result = await handler({ item_number: "invalid" }, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain("Invalid item number");
    });

    it("should handle missing item_number and no context", async () => {
      mockContext.payload = {};

      const handler = await main({ max: 10 });

      const result = await handler({}, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain("No issue number available");
    });

    it("should respect max count limit", async () => {
      const handler = await main({ max: 2 });

      const result1 = await handler({ item_number: 1 }, {});
      expect(result1.success).toBe(true);

      const result2 = await handler({ item_number: 2 }, {});
      expect(result2.success).toBe(true);

      const result3 = await handler({ item_number: 3 }, {});
      expect(result3.success).toBe(false);
      expect(result3.error).toContain("Max count");
    });

    it("should handle API errors gracefully", async () => {
      const handler = await main({ max: 10 });

      mockGithub.rest.issues.update = async () => {
        throw new Error("API Error: Not found");
      };

      const result = await handler({ item_number: 100 }, {});

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

      const result = await handler({ item_number: 100 }, {});

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

      await handler({ item_number: 100 }, {});

      expect(updateCalls[0].owner).toBe("test-owner");
      expect(updateCalls[0].repo).toBe("test-repo");
    });
  });
});
