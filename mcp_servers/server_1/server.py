"""MCP Server 1 — System Telemetry & Anomaly Detection.
Provides FastMCP tools to ingest operational telemetry metrics and evaluate system alert
using the calibrated Novelty Isolation Forest v3 model with continuous 
Robust Z-score feature extraction.
"""

import os
import json
import logging
from typing import Optional
from dotenv import load_dotenv
import joblib
import numpy as np
import pandas as pd
from fastmcp import FastMCP

load_dotenv()

mcp = FastMCP("support-copilot-mcp")
_logger = logging.getLogger("mcp-tools")

# Relative paths to model artifacts
BASE_DIR = os.path.dirname(os.path.abspath(__file__))
MODEL_PATH = os.path.join(BASE_DIR, "isolation_forest_model_v3.pkl")
BASELINE_PATH = os.path.join(BASE_DIR, "baseline_reference_v3.json")
METADATA_PATH = os.path.join(BASE_DIR, "model_metadata_v3.json")

# Fallback to v2 if v3 not present
if not os.path.exists(MODEL_PATH):
    MODEL_PATH = os.path.join(BASE_DIR, "isolation_forest_model_v2.pkl")
    BASELINE_PATH = os.path.join(BASE_DIR, "baseline_reference.json")
    METADATA_PATH = os.path.join(BASE_DIR, "model_metadata.json")

try:
    model = joblib.load(MODEL_PATH)
    with open(BASELINE_PATH, "r") as f:
        baseline = json.load(f)
    with open(METADATA_PATH, "r") as f:
        metadata = json.load(f)
        
    FEATURE_COLUMNS = metadata.get("feature_columns", [])
    BALANCED_THRESHOLD = metadata.get("balanced_threshold", metadata.get("optimal_threshold", -0.0117))
    METRIC_GROUPS = metadata.get("metric_groups", {
        "cpu": ["cpu_usage"],
        "memory": ["memory_usage"],
        "traffic": ["incoming_traffic", "outgoing_traffic", "network_throughput", "request_rate"],
        "errors": ["error_rate", "retry_rate", "timeout_count"],
        "latency": ["response_latency"],
        "availability": ["availability_percent"],
        "logs": ["log_repetition_count"]
    })
    METRIC_TO_GROUP = {m_name: g for g, metrics in METRIC_GROUPS.items() for m_name in metrics}
    
    _logger.info("Successfully loaded Isolation Forest v3 model and baseline artifacts. Threshold: %s", BALANCED_THRESHOLD)
except Exception as e:
    _logger.error("Failed to load ML artifacts from %s: %s", BASE_DIR, e)
    raise e


def compute_alert_feature_dataframe(metric_data: dict) -> tuple[pd.DataFrame, list[dict]]:
    """Transforms raw metric dictionary into a pandas DataFrame matching exact feature column names
    expected by the Novelty Isolation Forest model.
    """
    robust_z_dict = {}
    rel_change_dict = {}
    offending_metrics = []
    
    # Compute metric-level continuous deviations against baseline
    for m_name, val in metric_data.items():
        if val is None:
            continue
            
        if m_name in baseline:
            median = baseline[m_name]["median"]
            mad = baseline[m_name].get("mad", 1e-6)
            if mad == 0:
                mad = 1e-6
                
            rz = abs(val - median) / (1.4826 * mad)
            rc = (val - median) / (median if median != 0 else 1e-6)
            
            robust_z_dict[m_name] = rz
            rel_change_dict[m_name] = rc
            
            if rz >= 1.5:
                direction = "elevated above" if val > median else "dropped below"
                pct = abs(rc) * 100
                offending_metrics.append({
                    "metric": m_name,
                    "value": round(float(val), 4),
                    "median": round(float(median), 4),
                    "robust_z": round(float(rz), 2),
                    "relative_change_pct": f"{pct:.1f}%",
                    "description": f"{m_name}={val} ({pct:.1f}% {direction} median, Robust Z={rz:.2f})"
                })
        else:
            robust_z_dict[m_name] = 0.0
            rel_change_dict[m_name] = 0.0

    z_vals = np.array(list(robust_z_dict.values())) if robust_z_dict else np.array([0.0])
    rc_vals = np.array([abs(v) for v in rel_change_dict.values()]) if rel_change_dict else np.array([0.0])
    
    sorted_z = np.sort(z_vals)[::-1]
    sorted_rc = np.sort(rc_vals)[::-1]
    
    # Extract alert-level invariant features
    rz_max = float(sorted_z[0]) if len(sorted_z) > 0 else 0.0
    rz_top2_mean = float(np.mean(sorted_z[:min(2, len(sorted_z))])) if len(sorted_z) > 0 else 0.0
    rz_top3_mean = float(np.mean(sorted_z[:min(3, len(sorted_z))])) if len(sorted_z) > 0 else 0.0
    rz_mean = float(np.mean(z_vals)) if len(z_vals) > 0 else 0.0
    rz_p90 = float(np.percentile(z_vals, 90)) if len(z_vals) > 0 else 0.0
    
    max_abs_rc = float(sorted_rc[0]) if len(sorted_rc) > 0 else 0.0
    top2_mean_rc = float(np.mean(sorted_rc[:min(2, len(sorted_rc))])) if len(sorted_rc) > 0 else 0.0
    
    mild_outliers = (z_vals > 1.5).astype(float)
    strong_outliers = (z_vals > 2.5).astype(float)
    mild_count = float(np.sum(mild_outliers))
    mild_ratio = float(np.mean(mild_outliers)) if len(mild_outliers) > 0 else 0.0
    strong_count = float(np.sum(strong_outliers))
    strong_ratio = float(np.mean(strong_outliers)) if len(strong_outliers) > 0 else 0.0
    
    top3_energy = float(np.sum(sorted_z[:min(3, len(sorted_z))] ** 2)) if len(sorted_z) > 0 else 0.0
    
    # Domain group maximum deviations
    group_max_z = {f"group_{g}_max_z": 0.0 for g in METRIC_GROUPS.keys()}
    for m_name, rz in robust_z_dict.items():
        g = METRIC_TO_GROUP.get(m_name)
        if g:
            col_name = f"group_{g}_max_z"
            if rz > group_max_z[col_name]:
                group_max_z[col_name] = float(rz)

    # Map into DataFrame with exact feature columns expected by model
    feature_map = {
        "robust_z_max": rz_max,
        "robust_z_top2_mean": rz_top2_mean,
        "robust_z_top3_mean": rz_top3_mean,
        "robust_z_mean": rz_mean,
        "robust_z_p90": rz_p90,
        "max_abs_rel_change": max_abs_rc,
        "top2_mean_rel_change": top2_mean_rc,
        "mild_outlier_count": mild_count,
        "mild_outlier_ratio": mild_ratio,
        "strong_outlier_count": strong_count,
        "strong_outlier_ratio": strong_ratio,
        "top3_anomaly_energy": top3_energy,
        **group_max_z
    }
    
    df_features = pd.DataFrame([feature_map])
    for col in FEATURE_COLUMNS:
        if col not in df_features.columns:
            df_features[col] = 0.0
            
    return df_features[FEATURE_COLUMNS], offending_metrics


@mcp.tool(description="Ingest dynamic operational telemetry metrics, evaluate continuous robust deviation, and predict anomaly status.")
def detect_anomalies(
    cpu_usage: Optional[float] = None,
    memory_usage: Optional[float] = None,
    incoming_traffic: Optional[float] = None,
    outgoing_traffic: Optional[float] = None,
    error_rate: Optional[float] = None,
    network_throughput: Optional[float] = None,
    request_rate: Optional[float] = None,
    response_latency: Optional[float] = None,
    availability_percent: Optional[float] = None,
    retry_rate: Optional[float] = None,
    timeout_count: Optional[float] = None,
    log_repetition_count: Optional[float] = None,
    service_name: str = "default"
) -> dict:
    """Evaluates telemetry metrics via the calibrated Novelty Isolation Forest v3 classifier.

    Returns dict containing status (0=Anomaly, 1=Normal), anomaly score, risk level, confidence, and explanation.
    """
    _logger.info("tool=detect_anomalies request triggered for service=%s", service_name)
    
    # Collect non-None metrics
    raw_inputs = {
        "cpu_usage": cpu_usage,
        "memory_usage": memory_usage,
        "incoming_traffic": incoming_traffic,
        "outgoing_traffic": outgoing_traffic,
        "error_rate": error_rate,
        "retry_rate": retry_rate,
        "timeout_count": timeout_count,
        "network_throughput": network_throughput,
        "request_rate": request_rate,
        "response_latency": response_latency,
        "availability_percent": availability_percent,
        "log_repetition_count": log_repetition_count
    }
    
    # Filter out None values to support dynamic subsets
    metric_data = {k: v for k, v in raw_inputs.items() if v is not None}
    
    # If all inputs are omitted, populate with baseline medians
    if not metric_data:
        metric_data = {k: baseline[k]["median"] for k in baseline}
        
    try:
        features_df, offending_metrics = compute_alert_feature_dataframe(metric_data)
        
        # Raw score: -decision_function(X) -> Higher score = more anomalous
        raw_score = -float(model.decision_function(features_df)[0])
        anomaly_score = round(raw_score, 4)
        
        # Classification against calibrated balanced threshold with directional outlier verification
        has_elevation = (
            (features_df["robust_z_max"].values[0] >= 1.5) or
            (features_df["mild_outlier_count"].values[0] > 0) or
            (features_df["max_abs_rel_change"].values[0] >= 0.5)
        )
        is_anomaly = bool(anomaly_score >= BALANCED_THRESHOLD and has_elevation)
        final_status = 0 if is_anomaly else 1  # 0 for Anomaly (Real Alert), 1 for Normal (False Alarm)
        status_label = "Real Alert (Anomaly)" if is_anomaly else "False Alarm (Normal)"
        prediction_str = "REAL_ALERT" if is_anomaly else "FALSE_ALARM"
        
        # Confidence score based on distance from decision threshold
        distance = abs(anomaly_score - BALANCED_THRESHOLD)
        confidence = round(min(0.99, max(0.50, 0.50 + distance * 3.0)), 2)
        
        # Risk level determination
        if not is_anomaly:
            risk_level = "LOW"
        elif anomaly_score >= 0.0531:
            risk_level = "CRITICAL"
        elif anomaly_score >= 0.0:
            risk_level = "HIGH"
        else:
            risk_level = "MEDIUM"
            
        # Summary explanation
        if offending_metrics:
            spike_descriptions = [o['description'] for o in offending_metrics]
            summary = f"Real alert confirmed ({risk_level} risk, score={anomaly_score}): {'; '.join(spike_descriptions)}."
        else:
            if is_anomaly:
                summary = f"Real alert: Multi-metric anomaly pattern detected across telemetry groups (score={anomaly_score})."
            else:
                summary = "False alarm: System telemetry is operating within normal healthy bounds."

        response = {
            "status": final_status,
            "label": status_label,
            "engine": "IsolationForest_v3",
            "prediction": prediction_str,
            "confidence": confidence,
            "risk_level": risk_level,
            "anomaly_score": anomaly_score,
            "threshold": round(float(BALANCED_THRESHOLD), 4),
            "offending_metrics": offending_metrics,
            "summary": summary
        }
        
        _logger.info("tool=detect_anomalies completed. Result: %s (score=%s, risk=%s)",
                     status_label, anomaly_score, risk_level)
        return response

    except Exception as e:
        _logger.error("tool=detect_anomalies error encountered: %s", str(e))
        return {
            "status": 1,
            "label": "Normal",
            "engine": "IsolationForest_v3",
            "error": f"Inference execution failed: {str(e)}"
        }
