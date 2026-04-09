import express, { Request, Response } from "express";

const app = express();
app.use(express.json());

const PORT = parseInt(process.env.DASHBOARD_PORT || "8003", 10);
const AGGREGATOR_URL = process.env.AGGREGATOR_URL || "http://localhost:8001";
const COLLECTOR_URL = process.env.COLLECTOR_URL || "http://localhost:8002";

const logger = {
  info: (msg: string, ...args: unknown[]) =>
    console.log(`[dashboard] [INFO] ${msg}`, ...args),
  warn: (msg: string, ...args: unknown[]) =>
    console.warn(`[dashboard] [WARN] ${msg}`, ...args),
  error: (msg: string, ...args: unknown[]) =>
    console.error(`[dashboard] [ERROR] ${msg}`, ...args),
};

interface ServiceStatus {
  name: string;
  status: string;
  latency_ms: number;
}

async function checkService(name: string, url: string): Promise<ServiceStatus> {
  const start = Date.now();
  try {
    const resp = await fetch(`${url}/health`);
    const latency = Date.now() - start;
    if (resp.ok) {
      return { name, status: "healthy", latency_ms: latency };
    }
    return { name, status: `unhealthy (HTTP ${resp.status})`, latency_ms: latency };
  } catch {
    const latency = Date.now() - start;
    return { name, status: "unreachable", latency_ms: latency };
  }
}

app.get("/health", (_req: Request, res: Response) => {
  res.json({ status: "healthy", service: "dashboard", timestamp: Date.now() / 1000 });
});

app.get("/api/status", async (_req: Request, res: Response) => {
  logger.info("Checking service statuses");
  const statuses = await Promise.all([
    checkService("aggregator", AGGREGATOR_URL),
    checkService("collector", COLLECTOR_URL),
  ]);
  res.json({ services: statuses, checked_at: Date.now() / 1000 });
});

app.get("/api/metrics", async (_req: Request, res: Response) => {
  try {
    const resp = await fetch(`${AGGREGATOR_URL}/metrics`);
    if (!resp.ok) {
      logger.error("Aggregator returned %d", resp.status);
      res.status(502).json({ error: "Failed to fetch metrics from aggregator" });
      return;
    }
    const data = await resp.json();
    logger.info("Fetched %d metrics from aggregator", Array.isArray(data) ? data.length : 0);
    res.json(data);
  } catch (err) {
    logger.error("Error fetching metrics: %s", err);
    res.status(502).json({ error: "Aggregator service unavailable" });
  }
});

app.get("/api/summary", async (_req: Request, res: Response) => {
  try {
    const resp = await fetch(`${AGGREGATOR_URL}/metrics/summary`);
    if (!resp.ok) {
      res.status(502).json({ error: "Failed to fetch summary from aggregator" });
      return;
    }
    const data = await resp.json();
    res.json(data);
  } catch (err) {
    logger.error("Error fetching summary: %s", err);
    res.status(502).json({ error: "Aggregator service unavailable" });
  }
});

export { app };

if (require.main === module) {
  app.listen(PORT, () => {
    logger.info(`Dashboard BFF started on port ${PORT}`);
  });
}
