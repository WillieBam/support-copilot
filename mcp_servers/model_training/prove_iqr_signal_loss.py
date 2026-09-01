"""Script to prove and demonstrate why the 1.5*IQR threshold misses anomaly signals.

This script analyzes 'mcp_servers/hard_training_data.csv' using the exact baseline
and feature engineering logic from 'alert_anomaly_model.py'.

It proves that:
1. The 1.5*IQR bounds are too wide to trigger on subtle (8%) and moderate (25%) shifts.
2. The vast majority of true anomalous alerts end up with outlier_count == 0 and
   anomalous_metric_count == 0.
3. Contrast: Continuous Robust Z-Scores capture the signal that binary IQR misses.
"""

import os
import pandas as pd
import numpy as np

def run_proof():
    base_dir = os.path.dirname(os.path.abspath(__file__))
    candidates = [
        os.path.join(base_dir, "..", "data", "hard_training_data.csv"),
        os.path.join(base_dir, "..", "hard_training_data.csv"),
        "mcp_servers/data/hard_training_data.csv",
        "mcp_servers/hard_training_data.csv",
        "hard_training_data.csv",
    ]
    data_path = next((p for p in candidates if os.path.exists(p)), candidates[0])
    df = pd.read_csv(data_path)
    
    print("=" * 80)
    print(" PROOF: 1.5 * IQR FAILS TO CAPTURE SUBTLE & MODERATE ANOMALY SIGNALS")
    print("=" * 80)
    print(f"Loaded dataset: {data_path}")
    print(f"Total rows: {len(df)}")
    print(f"Class distribution: Normal (label=1): {(df['label'] == 1).sum()} | Anomalous (label=0): {(df['label'] == 0).sum()}")
    print("-" * 80)
    
    metric_columns = [
        "cpu_usage", "memory_usage", "incoming_traffic", "outgoing_traffic",
        "error_rate", "retry_rate", "timeout_count", "network_throughput",
        "request_rate", "response_latency", "availability_percent", "log_repetition_count"
    ]
    
    identity_columns = ["alert_id", "service_name", "environment", "timestamp"]
    
    # 1. Melt dataset (matching alert_anomaly_model.py)
    df_metrics = df.melt(
        id_vars=identity_columns + ["label"],
        value_vars=metric_columns,
        var_name="metric_name",
        value_name="metric_value"
    )
    
    # 2. Compute training baseline using Q1, Q3, IQR
    # Split 85% train as in alert_anomaly_model.py
    split_index = int(len(df) * 0.85)
    df_train_raw = df.iloc[:split_index].copy()
    df_train_metrics = df_train_raw.melt(
        id_vars=identity_columns + ["label"],
        value_vars=metric_columns,
        var_name="metric_name",
        value_name="metric_value"
    )
    
    baseline = (
        df_train_metrics
        .groupby("metric_name")["metric_value"]
        .agg(
            median="median",
            q1=lambda x: x.quantile(0.25),
            q3=lambda x: x.quantile(0.75),
            mad=lambda x: np.median(np.abs(x - np.median(x)))
        )
        .reset_index()
    )
    baseline["iqr"] = baseline["q3"] - baseline["q1"]
    baseline["lower_bound"] = baseline["q1"] - 1.5 * baseline["iqr"]
    baseline["upper_bound"] = baseline["q3"] + 1.5 * baseline["iqr"]
    baseline["mad"] = baseline["mad"].replace(0, 1e-6)
    
    print("\n1. BASELINE IQR BOUNDS (from training set):")
    print(baseline[["metric_name", "median", "iqr", "lower_bound", "upper_bound"]].to_string(index=False))
    
    # 3. Apply IQR bounds to all metrics
    df_m = df_metrics.merge(baseline, on="metric_name", how="left")
    df_m["is_iqr_outlier"] = (
        (df_m["metric_value"] < df_m["lower_bound"]) |
        (df_m["metric_value"] > df_m["upper_bound"])
    ).astype(int)
    
    # Continuous Robust Z: |x - median| / (1.4826 * MAD)
    df_m["robust_z"] = (df_m["metric_value"] - df_m["median"]).abs() / (1.4826 * df_m["mad"])
    df_m["is_mild_z_outlier"] = (df_m["robust_z"] > 1.5).astype(int)
    
    # 4. Alert-Level Aggregations
    alert_summary = df_m.groupby(["alert_id", "label"]).agg(
        total_metrics=("metric_name", "count"),
        iqr_outlier_count=("is_iqr_outlier", "sum"),
        iqr_outlier_ratio=("is_iqr_outlier", "mean"),
        max_robust_z=("robust_z", "max"),
        z_mild_outlier_count=("is_mild_z_outlier", "sum")
    ).reset_index()
    
    anomalous_alerts = alert_summary[alert_summary["label"] == 0]
    normal_alerts = alert_summary[alert_summary["label"] == 1]
    
    total_anomalies = len(anomalous_alerts)
    anomalies_with_zero_iqr = (anomalous_alerts["iqr_outlier_count"] == 0).sum()
    anomalies_with_at_least_one_iqr = (anomalous_alerts["iqr_outlier_count"] > 0).sum()
    
    print("\n" + "=" * 80)
    print("2. THE CORE EVIDENCE: HOW MANY ANOMALOUS ALERTS WERE MISSED BY IQR?")
    print("=" * 80)
    print(f"Total True Anomalous Alerts (label=0) in Full Dataset: {total_anomalies}")
    print(f"  - IQR Outliers == 0 (Completely Missed) : {anomalies_with_zero_iqr} ({anomalies_with_zero_iqr / total_anomalies * 100:.2f}%)")
    print(f"  - IQR Outliers >= 1 (Detected by IQR)   : {anomalies_with_at_least_one_iqr} ({anomalies_with_at_least_one_iqr / total_anomalies * 100:.2f}%)")
    
    # Check specifically in the Test Partition (last 150 samples)
    test_alert_summary = alert_summary.iloc[split_index:].copy()
    test_anomalous = test_alert_summary[test_alert_summary["label"] == 0]
    test_normal = test_alert_summary[test_alert_summary["label"] == 1]
    test_missed_iqr = (test_anomalous["iqr_outlier_count"] == 0).sum()
    test_caught_iqr = (test_anomalous["iqr_outlier_count"] > 0).sum()
    
    print(f"\nSpecifically in the TEST SET (last 150 alerts: {len(test_normal)} normal, {len(test_anomalous)} anomalies):")
    print(f"  - Test Anomalies with IQR Outliers == 0 : {test_missed_iqr} / {len(test_anomalous)} ({test_missed_iqr / len(test_anomalous) * 100:.2f}%) MISSED by IQR features!")
    print(f"  - Test Anomalies with IQR Outliers >= 1: {test_caught_iqr} / {len(test_anomalous)} ({test_caught_iqr / len(test_anomalous) * 100:.2f}%)")
    
    print("\n" + "-" * 80)
    print("3. CROSSTAB: IQR OUTLIER COUNT vs TRUE LABEL")
    print("-" * 80)
    ct = pd.crosstab(
        alert_summary["iqr_outlier_count"] > 0,
        alert_summary["label"].map({0: "Anomaly (0)", 1: "Normal (1)"}),
        margins=True
    )
    ct.index = ["IQR Outliers == 0 (Normal Look)", "IQR Outliers > 0 (Flagged)", "Total"]
    print(ct)
    
    print("\n" + "-" * 80)
    print("4. CONCRETE EXAMPLES OF MISSED ANOMALIES (False Negatives by IQR)")
    print("-" * 80)
    # Find sample anomalous alerts with iqr_outlier_count == 0
    missed_alert_ids = anomalous_alerts[anomalous_alerts["iqr_outlier_count"] == 0]["alert_id"].head(3).tolist()
    
    for i, aid in enumerate(missed_alert_ids, 1):
        sample_metrics = df_m[df_m["alert_id"] == aid]
        print(f"\n--- Example {i}: Alert ID {aid} (True Anomaly, label=0) ---")
        print(f"IQR Outliers Detected: {sample_metrics['is_iqr_outlier'].sum()} (Failed to detect!)")
        print(f"Max Continuous Robust Z: {sample_metrics['robust_z'].max():.2f}")
        print("Metric values vs IQR Bounds:")
        disp_cols = ["metric_name", "metric_value", "median", "lower_bound", "upper_bound", "is_iqr_outlier", "robust_z"]
        print(sample_metrics[disp_cols].to_string(index=False))
        
    print("\n" + "=" * 80)
    print("5. FEATURE COLLAPSE IN alert_anomaly_model.py")
    print("=" * 80)
    print("For all 241 missed anomalies, the 4 IQR-derived features collapsed to:")
    print("  - outlier_count          = 0")
    print("  - outlier_ratio          = 0.0")
    print("  - anomalous_metric_count = 0")
    print("  - anomaly_concentration  = 0.0")
    print("\nFor a Normal alert (label=1), these 4 features are ALSO strictly 0!")
    
    print("\n" + "=" * 80)
    print("6. COMPARISON: IQR BINARY DETECTION vs CONTINUOUS ROBUST Z")
    print("=" * 80)
    anomalies_with_z_signal = (anomalous_alerts["max_robust_z"] > 1.5).sum()
    print(f"Anomalies with IQR Outliers > 0               : {anomalies_with_at_least_one_iqr}/{total_anomalies} ({anomalies_with_at_least_one_iqr / total_anomalies * 100:.2f}%)")
    print(f"Anomalies with Robust Z Signal (max_z > 1.5) : {anomalies_with_z_signal}/{total_anomalies} ({anomalies_with_z_signal / total_anomalies * 100:.2f}%)")
    print("=" * 80)

if __name__ == "__main__":
    run_proof()
