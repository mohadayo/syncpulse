import pytest
from app import app, metrics_store


@pytest.fixture
def client():
    app.config["TESTING"] = True
    metrics_store.clear()
    with app.test_client() as client:
        yield client


def test_health(client):
    resp = client.get("/health")
    assert resp.status_code == 200
    data = resp.get_json()
    assert data["status"] == "healthy"
    assert data["service"] == "aggregator"
    assert "timestamp" in data


def test_post_metrics(client):
    payload = {"service": "test-svc", "metrics": {"cpu": 45.2, "memory": 1024}}
    resp = client.post("/metrics", json=payload)
    assert resp.status_code == 201
    data = resp.get_json()
    assert data["status"] == "accepted"
    assert data["id"] == 0


def test_post_metrics_empty_body(client):
    resp = client.post("/metrics", content_type="application/json")
    assert resp.status_code == 400


def test_post_metrics_missing_fields(client):
    resp = client.post("/metrics", json={"service": "only-service"})
    assert resp.status_code == 400
    data = resp.get_json()
    assert "Missing required fields" in data["error"]


def test_get_metrics(client):
    client.post("/metrics", json={"service": "svc-a", "metrics": {"cpu": 10}})
    client.post("/metrics", json={"service": "svc-b", "metrics": {"cpu": 20}})
    resp = client.get("/metrics")
    assert resp.status_code == 200
    data = resp.get_json()
    assert len(data) == 2


def test_get_metrics_filtered(client):
    client.post("/metrics", json={"service": "svc-a", "metrics": {"cpu": 10}})
    client.post("/metrics", json={"service": "svc-b", "metrics": {"cpu": 20}})
    resp = client.get("/metrics?service=svc-a")
    assert resp.status_code == 200
    data = resp.get_json()
    assert len(data) == 1
    assert data[0]["service"] == "svc-a"


def test_get_summary(client):
    client.post("/metrics", json={"service": "svc-a", "metrics": {"cpu": 10}})
    client.post("/metrics", json={"service": "svc-a", "metrics": {"cpu": 20}})
    client.post("/metrics", json={"service": "svc-b", "metrics": {"cpu": 30}})
    resp = client.get("/metrics/summary")
    assert resp.status_code == 200
    data = resp.get_json()
    assert data["total_entries"] == 3
    assert data["services"]["svc-a"]["count"] == 2
    assert data["services"]["svc-b"]["count"] == 1
