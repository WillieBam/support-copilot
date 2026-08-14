
"""Alert Anomaly Detection Model v2 — Continuous Deviation & Novelty Isolation Forest.

Key improvements over v1:
1. Replaces binary 1.5*IQR clipping with continuous Robust Z-Scores (Median + MAD)
   to detect subtle (8%) and moderate (25%) metric perturbations.
2. Uses Top-K order statistics and Anomaly Energy to prevent anomaly signal dilution
   when alerts contain dynamic subsets of metrics.
3. Groups metrics into domain categories (cpu, memory, traffic, errors, latency, availability, logs)
   with 0.0 neutral padding for absent metric groups.
4. Trains Isolation Forest in Novelty Detection mode (on clean normal telemetry, label=1).
5. Calibrates decision threshold on continuous anomaly score [-decision_function(X)]
   via Precision-Recall / F1 optimization.
6. Exports serialized model artifacts, baseline reference, and metadata.
"""

import os
import json
import joblib
import numpy as np
import pandas as pd
from sklearn.ensemble import IsolationForest
from sklearn.metrics import (
    classification_report, confusion_matrix, roc_auc_score, precision_recall_curve,
    f1_score, precision_score, recall_score, accuracy_score
)

# Paths
BASE_DIR = os.path.dirname(os.path.abspath(__file__))
DATA_PATH = os.path.join(BASE_DIR, "..", "data", "hard_training_data.csv")
if not os.path.exists(DATA_PATH):
    DATA_PATH = os.path.join(BASE_DIR, "..", "hard_training_data.csv")
OUTPUT_DIR = os.path.join(BASE_DIR, "..", "server_1")
os.makedirs(OUTPUT_DIR, exist_ok=True)

# Metric definitions & groups
IDENTITY_COLUMNS = ["alert_id", "service_name", "environment", "timestamp"]
LABEL_COLUMN = "label"

METRIC_COLUMNS = [
    "cpu_usage", "memory_usage", "incoming_traffic", "outgoing_traffic",
    "error_rate", "retry_rate", "timeout_count", "network_throughput",
    "request_rate", "response_latency", "availability_percent", "log_repetition_count"
]

METRIC_GROUPS = {
    "cpu": ["cpu_usage"],
    "memory": ["memory_usage"],
    "traffic": ["incoming_traffic", "outgoing_traffic", "network_throughput", "request_rate"],
    "errors": ["error_rate", "retry_rate", "timeout_count"],
    "latency": ["response_latency"],
    "availability": ["availability_percent"],
    "logs": ["log_repetition_count"]
}

METRIC_TO_GROUP = {m: g for g, metrics in METRIC_GROUPS.items() for m in metrics}


def load_and_split_data(csv_path: str, train_ratio: float = 0.70, val_ratio: float = 0.15):
    """Loads dataset and performs time-based 3-way split: Train (70%), Val (15%), Test (15%)."""
    df_raw = pd.read_csv(csv_path)
    df_raw["timestamp"] = pd.to_datetime(df_raw["timestamp"])
    df_raw = df_raw.sort_values("timestamp").reset_index(drop=True)

    n_total = len(df_raw)
    train_end = int(n_total * train_ratio)
    val_end = int(n_total * (train_ratio + val_ratio))

    df_train = df_raw.iloc[:train_end].copy()
    df_val = df_raw.iloc[train_end:val_end].copy()
    df_test = df_raw.iloc[val_end:].copy()

    print(f"Dataset loaded: {n_total} total rows (Time-sliced chronologically)")
    print(f"  - Train Split (70%): {len(df_train)} rows | Normal(1)={(df_train['label'] == 1).sum()}, Anomaly(0)={(df_train['label'] == 0).sum()} | Time: {df_train['timestamp'].min().strftime('%Y-%m-%d %H:%M')} -> {df_train['timestamp'].max().strftime('%Y-%m-%d %H:%M')}")
    print(f"  - Val Split   (15%): {len(df_val)} rows | Normal(1)={(df_val['label'] == 1).sum()}, Anomaly(0)={(df_val['label'] == 0).sum()} | Time: {df_val['timestamp'].min().strftime('%Y-%m-%d %H:%M')} -> {df_val['timestamp'].max().strftime('%Y-%m-%d %H:%M')}")
    print(f"  - Test Split  (15%): {len(df_test)} rows | Normal(1)={(df_test['label'] == 1).sum()}, Anomaly(0)={(df_test['label'] == 0).sum()} | Time: {df_test['timestamp'].min().strftime('%Y-%m-%d %H:%M')} -> {df_test['timestamp'].max().strftime('%Y-%m-%d %H:%M')}")

    return df_train, df_val, df_test


def compute_baseline(df_train_raw: pd.DataFrame):
    """Computes robust statistical baselines (Median, MAD, Mean, Std) exclusively from NORMAL train samples."""
    # Filter normal training samples only
    df_normal = df_train_raw[df_train_raw["label"] == 1].copy()

    df_normal_melt = df_normal.melt(
        id_vars=IDENTITY_COLUMNS + [LABEL_COLUMN],
        value_vars=METRIC_COLUMNS,
        var_name="metric_name",
        value_name="metric_value"
    )

    # 1. Global metric baseline (fallback)
    global_baseline = (
        df_normal_melt
        .groupby("metric_name")["metric_value"]
        .agg(
            median="median",
            mean="mean",
            std="std",
            mad=lambda x: np.median(np.abs(x - np.median(x))),
            q25=lambda x: x.quantile(0.25),
            q75=lambda x: x.quantile(0.75)
        )
        .reset_index()
    )
    global_baseline["mad"] = global_baseline["mad"].replace(0, 1e-6)
    global_baseline["std"] = global_baseline["std"].replace(0, 1e-6)
    global_baseline["iqr"] = (global_baseline["q75"] - global_baseline["q25"]).replace(0, 1e-6)

    # 2. Per-service metric baseline
    service_baseline = (
        df_normal_melt
        .groupby(["service_name", "metric_name"])["metric_value"]
        .agg(
            median="median",
            mean="mean",
            std="std",
            mad=lambda x: np.median(np.abs(x - np.median(x))),
            q25=lambda x: x.quantile(0.25),
            q75=lambda x: x.quantile(0.75)
        )
        .reset_index()
    )
    service_baseline["mad"] = service_baseline["mad"].replace(0, 1e-6)
    service_baseline["std"] = service_baseline["std"].replace(0, 1e-6)

    return global_baseline, service_baseline


def extract_features(df_raw: pd.DataFrame, global_baseline: pd.DataFrame):
    """Extracts invariant continuous deviation features and domain aggregations per alert."""
    df_melt = df_raw.melt(
        id_vars=IDENTITY_COLUMNS + [LABEL_COLUMN],
        value_vars=METRIC_COLUMNS,
        var_name="metric_name",
        value_name="metric_value"
    )

    # Attach group name
    df_melt["group_name"] = df_melt["metric_name"].map(METRIC_TO_GROUP)

    # Merge with robust baseline
    m = df_melt.merge(global_baseline, on="metric_name", how="left")

    # 1. Continuous Robust Z-score: |x - median| / (1.4826 * MAD)
    m["robust_z"] = (m["metric_value"] - m["median"]).abs() / (1.4826 * m["mad"])
    m["rel_change"] = (m["metric_value"] - m["median"]) / m["median"].replace(0, 1e-6)

    # 2. Soft Outlier Thresholds
    m["is_mild_outlier"] = (m["robust_z"] > 1.5).astype(float)
    m["is_strong_outlier"] = (m["robust_z"] > 2.5).astype(float)

    def top_k_mean(s, k=3):
        return s.nlargest(min(k, len(s))).mean() if len(s) > 0 else 0.0

    def top_k_energy(s, k=3):
        return (s.nlargest(min(k, len(s))) ** 2).sum() if len(s) > 0 else 0.0

    # 3. Order Statistics & Alert Summary Aggregations
    alert_summary = (
        m.groupby("alert_id")
        .agg(
            metric_count=("metric_name", "count"),
            robust_z_max=("robust_z", "max"),
            robust_z_top2_mean=("robust_z", lambda x: top_k_mean(x, 2)),
            robust_z_top3_mean=("robust_z", lambda x: top_k_mean(x, 3)),
            robust_z_mean=("robust_z", "mean"),
            robust_z_p90=("robust_z", lambda x: x.quantile(0.90)),
            max_abs_rel_change=("rel_change", lambda x: x.abs().max()),
            top2_mean_rel_change=("rel_change", lambda x: top_k_mean(x.abs(), 2)),
            mild_outlier_count=("is_mild_outlier", "sum"),
            mild_outlier_ratio=("is_mild_outlier", "mean"),
            strong_outlier_count=("is_strong_outlier", "sum"),
            strong_outlier_ratio=("is_strong_outlier", "mean"),
            top3_anomaly_energy=("robust_z", lambda x: top_k_energy(x, 3))
        )
        .reset_index()
    )

    # 4. Group-level max deviations (pivoted with 0.0 neutral padding for missing groups)
    group_max = (
        m.groupby(["alert_id", "group_name"])["robust_z"]
        .max()
        .unstack(fill_value=0.0)
    )
    group_max.columns = [f"group_{c}_max_z" for c in group_max.columns]
    group_max = group_max.reset_index()

    # Ensure all expected groups exist in column list
    expected_group_cols = [f"group_{g}_max_z" for g in METRIC_GROUPS.keys()]
    for col in expected_group_cols:
        if col not in group_max.columns:
            group_max[col] = 0.0

    features = alert_summary.merge(group_max, on="alert_id", how="left").fillna(0.0)

    # Attach labels & metadata
    labels = df_raw[["alert_id", "service_name", "timestamp", "label"]].drop_duplicates("alert_id")
    features = features.merge(labels, on="alert_id", how="left")

    return features


def train_and_calibrate():
    print("=" * 80)
    print(" TRAINING ALERT ANOMALY DETECTION MODEL v2 (TIME-SLICED NOVELTY ISOLATION FOREST)")
    print("=" * 80)

    # 1. Load data & 3-way Time Slicing Split (70% Train, 15% Val, 15% Test)
    df_train_raw, df_val_raw, df_test_raw = load_and_split_data(DATA_PATH, train_ratio=0.70, val_ratio=0.15)

    # 2. Compute Baseline from Train Normal data ONLY
    global_baseline, service_baseline = compute_baseline(df_train_raw)
    print("\n--- Baseline Reference (from Train Normal samples) ---")
    print(global_baseline[["metric_name", "median", "mad", "std", "iqr"]].head(6).to_string(index=False))

    # 3. Extract Features for Train, Val, and Test using Train Baseline
    train_features_df = extract_features(df_train_raw, global_baseline)
    val_features_df = extract_features(df_val_raw, global_baseline)
    test_features_df = extract_features(df_test_raw, global_baseline)

    # Feature column set
    exclude_cols = ["alert_id", "service_name", "timestamp", "label", "metric_count"]
    feature_cols = [c for c in train_features_df.columns if c not in exclude_cols]
    print(f"\nEngineered Feature Set ({len(feature_cols)} features):")
    for i, f in enumerate(feature_cols, 1):
        print(f"  {i}. {f}")

    # Feature comparison
    print("\n--- Feature Mean Comparison (Normal vs Anomaly on Train) ---")
    comp = pd.DataFrame({
        "Normal (label=1)": train_features_df[train_features_df["label"] == 1][feature_cols].mean(),
        "Anomaly (label=0)": train_features_df[train_features_df["label"] == 0][feature_cols].mean()
    })
    comp["Ratio (Anomaly/Normal)"] = comp["Anomaly (label=0)"] / (comp["Normal (label=1)"] + 1e-6)
    print(comp.round(4).to_string())

    # 4. Novelty Detection Training
    # Train Isolation Forest ONLY on normal training samples (Train Split)
    train_normal_X = train_features_df[train_features_df["label"] == 1][feature_cols]

    iso_forest = IsolationForest(
        n_estimators=300,
        max_samples=min(256, len(train_normal_X)),
        contamination=0.08,  # expected background noise in normal telemetry
        random_state=42,
        n_jobs=-1
    )
    iso_forest.fit(train_normal_X)
    print("\nIsolation Forest successfully fitted on Normal baseline alerts (Train Set).")

    # 5. Threshold Calibration on VALIDATION SET (15% Middle Time Slice)
    # Raw score: -decision_function(X) -> Higher score = more anomalous
    val_scores = -iso_forest.decision_function(val_features_df[feature_cols])
    y_val_anomaly = (val_features_df["label"] == 0).astype(int)  # 1 = Anomaly, 0 = Normal

    val_precisions, val_recalls, val_thresholds = precision_recall_curve(y_val_anomaly, val_scores)
    val_f1_scores = 2 * (val_precisions * val_recalls) / (val_precisions + val_recalls + 1e-8)
    best_idx = np.argmax(val_f1_scores)
    optimal_threshold = float(val_thresholds[min(best_idx, len(val_thresholds) - 1)])

    # Operating point presets from Validation Set:
    # 1. Balanced F1 (default optimal)
    # 2. High Recall (aiming >= 85% recall)
    high_recall_indices = np.where(val_recalls >= 0.85)[0]
    high_recall_idx = high_recall_indices[-1] if len(high_recall_indices) > 0 else 0
    high_recall_threshold = float(val_thresholds[min(high_recall_idx, len(val_thresholds) - 1)])

    # 3. High Precision (aiming >= 95% precision)
    high_prec_indices = np.where(val_precisions >= 0.95)[0]
    high_prec_idx = high_prec_indices[0] if len(high_prec_indices) > 0 else best_idx
    high_prec_threshold = float(val_thresholds[min(high_prec_idx, len(val_thresholds) - 1)])

    val_auc = roc_auc_score(y_val_anomaly, val_scores)
    print(f"\n" + "=" * 80)
    print(" VALIDATION SET CALIBRATION RESULTS (150 alerts)")
    print("=" * 80)
    print(f"Validation ROC-AUC: {val_auc:.4f}")
    print(f"1. Balanced (Max F1) : Threshold={optimal_threshold:.4f} | Val F1={val_f1_scores[best_idx]:.4f} | Rec={val_recalls[best_idx]:.4f} | Prec={val_precisions[best_idx]:.4f}")
    print(f"2. High Recall (>=85%): Threshold={high_recall_threshold:.4f} | Val F1={val_f1_scores[high_recall_idx]:.4f} | Rec={val_recalls[high_recall_idx]:.4f} | Prec={val_precisions[high_recall_idx]:.4f}")
    print(f"3. High Precision    : Threshold={high_prec_threshold:.4f} | Val F1={val_f1_scores[high_prec_idx]:.4f} | Rec={val_recalls[high_prec_idx]:.4f} | Prec={val_precisions[high_prec_idx]:.4f}")

    # 6. Unbiased Final Evaluation on TEST SET (15% Future Time Slice)
    test_scores = -iso_forest.decision_function(test_features_df[feature_cols])
    y_test_anomaly = (test_features_df["label"] == 0).astype(int)  # 1 = Anomaly, 0 = Normal

    test_preds = (test_scores >= optimal_threshold).astype(int)
    test_auc = roc_auc_score(y_test_anomaly, test_scores)
    test_precision = precision_score(y_test_anomaly, test_preds)
    test_recall = recall_score(y_test_anomaly, test_preds)
    test_f1 = f1_score(y_test_anomaly, test_preds)
    test_acc = accuracy_score(y_test_anomaly, test_preds)

    print("\n" + "=" * 80)
    print(" UNBIASED TEST SET EVALUATION RESULTS (150 alerts - Balanced Threshold)")
    print("=" * 80)
    print(f"Test Accuracy           : {test_acc:.4f} ({test_acc:.2%})")
    print(f"Test Precision (Anomaly): {test_precision:.4f} ({test_precision:.2%})")
    print(f"Test Recall    (Anomaly): {test_recall:.4f} ({test_recall:.2%})")
    print(f"Test F1-Score  (Anomaly): {test_f1:.4f}")
    print(f"Test ROC-AUC Score      : {test_auc:.4f}")

    print("\nConfusion Matrix (Test Set):")
    cm = confusion_matrix(y_test_anomaly, test_preds)
    cm_df = pd.DataFrame(
        cm,
        index=["Actual Normal (0)", "Actual Anomaly (1)"],
        columns=["Pred Normal (0)", "Pred Anomaly (1)"]
    )
    print(cm_df.to_string())

    print("\nClassification Report (Test Set):")
    print(classification_report(
        y_test_anomaly,
        test_preds,
        target_names=["Normal Alert", "Anomalous Alert"],
        digits=4
    ))

    # Evaluate High Recall Threshold on Test Set
    test_preds_high_rec = (test_scores >= high_recall_threshold).astype(int)
    high_rec_prec = precision_score(y_test_anomaly, test_preds_high_rec)
    high_rec_rec = recall_score(y_test_anomaly, test_preds_high_rec)
    high_rec_f1 = f1_score(y_test_anomaly, test_preds_high_rec)
    high_rec_acc = accuracy_score(y_test_anomaly, test_preds_high_rec)

    print("\n" + "-" * 80)
    print(f" TEST SET EVALUATION RESULTS (High Recall Threshold: {high_recall_threshold:.4f})")
    print("-" * 80)
    print(f"Test Accuracy           : {high_rec_acc:.4f} ({high_rec_acc:.2%})")
    print(f"Test Precision (Anomaly): {high_rec_prec:.4f} ({high_rec_prec:.2%})")
    print(f"Test Recall    (Anomaly): {high_rec_rec:.4f} ({high_rec_rec:.2%})")
    print(f"Test F1-Score  (Anomaly): {high_rec_f1:.4f}")
    print(classification_report(
        y_test_anomaly,
        test_preds_high_rec,
        target_names=["Normal Alert", "Anomalous Alert"],
        digits=4
    ))

    # 7. Save Artifacts for MCP Server 1
    model_artifact_path = os.path.join(OUTPUT_DIR, "isolation_forest_model_v2.pkl")
    baseline_artifact_path = os.path.join(OUTPUT_DIR, "baseline_reference.json")
    metadata_artifact_path = os.path.join(OUTPUT_DIR, "model_metadata.json")

    joblib.dump(iso_forest, model_artifact_path)

    # Save baseline as JSON lookup
    baseline_dict = global_baseline.set_index("metric_name").to_dict(orient="index")
    with open(baseline_artifact_path, "w") as f:
        json.dump(baseline_dict, f, indent=2)

    # Save metadata
    metadata = {
        "model_version": "v2.0",
        "model_type": "IsolationForest_Novelty",
        "split_strategy": "time_sliced_70_15_15",
        "split_sizes": {
            "train": len(df_train_raw),
            "val": len(df_val_raw),
            "test": len(df_test_raw)
        },
        "feature_columns": feature_cols,
        "optimal_threshold": optimal_threshold,
        "operating_thresholds": {
            "balanced": optimal_threshold,
            "high_recall": high_recall_threshold,
            "high_precision": high_prec_threshold
        },
        "val_roc_auc": round(float(val_auc), 4),
        "test_roc_auc": round(float(test_auc), 4),
        "test_f1_score": round(float(test_f1), 4),
        "metric_groups": METRIC_GROUPS
    }
    with open(metadata_artifact_path, "w") as f:
        json.dump(metadata, f, indent=2)

    print("\nArtifacts successfully exported:")
    print(f"  - Model   : {model_artifact_path}")
    print(f"  - Baseline: {baseline_artifact_path}")
    print(f"  - Metadata: {metadata_artifact_path}")
    print("=" * 80)

    return metadata


if __name__ == "__main__":
    train_and_calibrate()
