import logging
import os
import time
from flask import Flask, jsonify, request

app = Flask(__name__)

LOG_LEVEL = os.environ.get("LOG_LEVEL", "INFO").upper()
logging.basicConfig(
    level=getattr(logging, LOG_LEVEL, logging.INFO),
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
)
logger = logging.getLogger("aggregator")

PORT = int(os.environ.get("AGGREGATOR_PORT", 8001))

# In-memory metrics store
metrics_store: list[dict] = []


@app.route("/health")
def health():
    return jsonify({"status": "healthy", "service": "aggregator", "timestamp": time.time()})


@app.route("/metrics", methods=["POST"])
def receive_metrics():
    data = request.get_json()
    if not data:
        logger.warning("Received empty metrics payload")
        return jsonify({"error": "Request body must be JSON"}), 400

    if "service" not in data or "metrics" not in data:
        logger.warning("Missing required fields: service, metrics")
        return jsonify({"error": "Missing required fields: service, metrics"}), 400

    entry = {
        "service": data["service"],
        "metrics": data["metrics"],
        "received_at": time.time(),
    }
    metrics_store.append(entry)
    logger.info("Stored metrics from service=%s", data["service"])
    return jsonify({"status": "accepted", "id": len(metrics_store) - 1}), 201


@app.route("/metrics", methods=["GET"])
def get_metrics():
    service_filter = request.args.get("service")
    if service_filter:
        filtered = [m for m in metrics_store if m["service"] == service_filter]
        logger.info("Returning %d metrics for service=%s", len(filtered), service_filter)
        return jsonify(filtered)
    logger.info("Returning all %d metrics", len(metrics_store))
    return jsonify(metrics_store)


@app.route("/metrics/summary", methods=["GET"])
def get_summary():
    services = {}
    for entry in metrics_store:
        svc = entry["service"]
        if svc not in services:
            services[svc] = {"count": 0, "last_received": 0}
        services[svc]["count"] += 1
        services[svc]["last_received"] = max(services[svc]["last_received"], entry["received_at"])
    logger.info("Returning summary for %d services", len(services))
    return jsonify({"services": services, "total_entries": len(metrics_store)})


def create_app():
    return app


if __name__ == "__main__":
    logger.info("Starting aggregator on port %d", PORT)
    app.run(host="0.0.0.0", port=PORT)
