// @ts-check
import { describe, it, expect } from "vitest";
const { AGENT_OUTPUT_FILENAME, TMP_GH_AW_PATH, megabytes, MAX_BUFFER_SIZE, GIT_PATCH_MAX_BUFFER_SIZE } = require("./constants.cjs");

describe("constants", () => {
  it("should export AGENT_OUTPUT_FILENAME", () => {
    expect(AGENT_OUTPUT_FILENAME).toBe("agent_output.json");
  });

  it("should export TMP_GH_AW_PATH", () => {
    expect(TMP_GH_AW_PATH).toBe("/tmp/gh-aw");
  });

  it("megabytes should convert MB to bytes", () => {
    expect(megabytes(1)).toBe(1024 * 1024);
    expect(megabytes(10)).toBe(10 * 1024 * 1024);
    expect(megabytes(200)).toBe(200 * 1024 * 1024);
  });

  it("MAX_BUFFER_SIZE should be 10MB", () => {
    expect(MAX_BUFFER_SIZE).toBe(megabytes(10));
  });

  it("GIT_PATCH_MAX_BUFFER_SIZE should be 200MB", () => {
    expect(GIT_PATCH_MAX_BUFFER_SIZE).toBe(megabytes(200));
  });

  it("GIT_PATCH_MAX_BUFFER_SIZE should be larger than MAX_BUFFER_SIZE", () => {
    expect(GIT_PATCH_MAX_BUFFER_SIZE).toBeGreaterThan(MAX_BUFFER_SIZE);
  });
});
