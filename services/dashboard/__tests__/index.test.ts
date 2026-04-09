import request from "supertest";
import { app } from "../src/index";

describe("Dashboard BFF", () => {
  describe("GET /health", () => {
    it("should return healthy status", async () => {
      const res = await request(app).get("/health");
      expect(res.status).toBe(200);
      expect(res.body.status).toBe("healthy");
      expect(res.body.service).toBe("dashboard");
      expect(res.body.timestamp).toBeDefined();
    });
  });

  describe("GET /api/status", () => {
    it("should return service statuses", async () => {
      const res = await request(app).get("/api/status");
      expect(res.status).toBe(200);
      expect(res.body.services).toBeDefined();
      expect(Array.isArray(res.body.services)).toBe(true);
      expect(res.body.services.length).toBe(2);
      expect(res.body.checked_at).toBeDefined();
      // Services are likely unreachable in test env
      for (const svc of res.body.services) {
        expect(svc.name).toBeDefined();
        expect(svc.status).toBeDefined();
        expect(svc.latency_ms).toBeDefined();
      }
    });
  });

  describe("GET /api/metrics", () => {
    it("should return 502 when aggregator is unavailable", async () => {
      const res = await request(app).get("/api/metrics");
      expect(res.status).toBe(502);
      expect(res.body.error).toBeDefined();
    });
  });

  describe("GET /api/summary", () => {
    it("should return 502 when aggregator is unavailable", async () => {
      const res = await request(app).get("/api/summary");
      expect(res.status).toBe(502);
      expect(res.body.error).toBeDefined();
    });
  });
});
