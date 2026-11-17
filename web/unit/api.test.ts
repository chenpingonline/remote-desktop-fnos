import { test } from "node:test";
import assert from "node:assert/strict";
import { api, setCSRF } from "../src/api.ts";
const response = (status: number, body: unknown) =>
  ({
    status,
    ok: status >= 200 && status < 300,
    json: async () => body,
  }) as Response;
test("renews an expired CSRF token once while retaining the unsaved body", async () => {
  setCSRF("old", "alice");
  const calls: any[] = [];
  globalThis.fetch = (async (path: any, options: any) => {
    calls.push({ path, options });
    if (calls.length === 1)
      return response(403, { error: { code: "CSRF_EXPIRED" } });
    if (calls.length === 2)
      return response(200, { uid: "alice", csrfToken: "fresh" });
    return response(201, { id: "saved" });
  }) as typeof fetch;
  const body = { name: "draft" };
  assert.deepEqual(await api("/connections", "POST", body), { id: "saved" });
  assert.equal(calls.length, 3);
  assert.equal(calls[0].options.body, calls[2].options.body);
  assert.equal(calls[2].options.headers["X-CSRF-Token"], "fresh");
});
test("does not replay under another NAS identity", async () => {
  setCSRF("old", "alice");
  let calls = 0;
  globalThis.fetch = (async () => {
    calls++;
    return calls === 1
      ? response(403, { error: { code: "CSRF_EXPIRED" } })
      : response(200, { uid: "bob", csrfToken: "other" });
  }) as typeof fetch;
  await assert.rejects(
    api("/connections", "POST", { name: "draft" }),
    /登录用户已改变/,
  );
  assert.equal(calls, 2);
});
test("does not retry an origin rejection or an uncertain network result", async () => {
  setCSRF("old", "alice");
  let calls = 0;
  globalThis.fetch = (async () => {
    calls++;
    return response(403, {
      error: { code: "ORIGIN_REJECTED", message: "来源无效" },
    });
  }) as typeof fetch;
  await assert.rejects(api("/connections", "POST", {}), /来源无效/);
  assert.equal(calls, 1);
  calls = 0;
  globalThis.fetch = (async () => {
    calls++;
    throw new Error("network");
  }) as typeof fetch;
  await assert.rejects(api("/connections", "POST", {}), /network/);
  assert.equal(calls, 1);
});
