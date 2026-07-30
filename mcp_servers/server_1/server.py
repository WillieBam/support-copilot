"""MCP Server 1 — System Telemetry & Anomaly Detection.

Provides FastMCP tools to ingest operational telemetry metrics and evaluate system health
using an IsolationForest machine learning model.
"""

import os
import logging
from dotenv import load_dotenv
import joblib
import numpy as np
from fastmcp import FastMCP

load_dotenv()

mcp = FastMCP("support-copilot-mcp")
_logger = logging.getLogger("mcp-tools")

# use relative path
BASE_DIR = os.path.dirname(os.path.abspath(__file__))
MODEL_PATH = os.path.join(BASE_DIR, "isolation_forest_model.pkl")
SCALER_PATH = os.path.join(BASE_DIR, "standard_scaler.pkl")

try:
    model = joblib.load(MODEL_PATH)
    scaler = joblib.load(SCALER_PATH)
    _logger.info("Successfully loaded Isolation Forest model and scaler artifacts.")
except Exception as e:
    _logger.error("Failed to load ML artifacts from %s: %s", BASE_DIR, e)
    raise e

MODEL_FEATURES = [
    "cpu_usage", "memory_usage", "incoming_traffic", "outgoing_traffic",
    "error_rate", "retry_rate", "timeout_count", "network_throughput",
    "request_rate", "response_latency", "availability_percent", "log_repetition_count"
]

@mcp.tool(description="Ingest metrics, apply median imputation for missing operational data, and predict anomaly status.")
def detect_anomalies(
    cpu_usage: float,
    memory_usage: float,
    incoming_traffic: float,
    outgoing_traffic: float,
    error_rate: float,
    network_throughput: float,
    request_rate: float,
    response_latency: float,
    availability_percent: float
) -> dict:
    """
    Evaluates system telemetry vectors via an IsolationForest classifier;
    returns a dict indicating if the system state is Anomaly (0) or Normal (1).
    """
    _logger.info("tool=detect_anomalies request triggered")
    
    imputed_data = {
        "cpu_usage": cpu_usage,
        "memory_usage": memory_usage,
        "incoming_traffic": incoming_traffic,
        "outgoing_traffic": outgoing_traffic,
        "error_rate": error_rate,
        "network_throughput": network_throughput,
        "request_rate": request_rate,
        "response_latency": response_latency,
        "availability_percent": availability_percent,
        # valdate median fallback constraints
        "retry_rate": 0.25,
        "timeout_count": 10,
        "log_repetition_count": 2
    }
    try:
        ordered_vector = [imputed_data[feature] for feature in MODEL_FEATURES]
        input_matrix = np.array([ordered_vector])
        scaled_matrix = scaler.transform(input_matrix)
        # -1 for anomaly, 1 for normal
        raw_prediction = model.predict(scaled_matrix)[0]
        final_status = 0 if raw_prediction == -1 else 1
        status_label = "Anomaly" if final_status == 0 else "Normal"
        _logger.info("tool=detect_anomalies prediction completed. Result:%s (%s)",
                     status_label, final_status)  
        return {
            "status": final_status,
            "label": status_label,
            "engine": "IsolationForest"
        }
    except Exception as e:
        _logger.error("tool=detect_anomalies error encountered: %s",  str(e))
        return {
            "error": f"Inference execution failed: {str(e)}"
        }
