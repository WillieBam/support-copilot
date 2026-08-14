"""Experiment Script: Exploring Precision Targets (80%, 85%, 90%, 95%) and Resulting Recall.

Discovers operating points where the model maintains approximately X% precision
while detecting Y% of anomalies on both the Validation and Test sets.
"""

import json
import os
import sys

import joblib
import numpy as np
import pandas as pd
from sklearn.metrics import (
    accuracy_score,
    classification_report,
    confusion_matrix,
    f1_score,
    precision_recall_curve,
    precision_score,
    recall_score,
    roc_auc_score,
)

sys.path.insert(0, '/home/meilinlee/meilin/support_copilot/mcp_servers')

from model_training.alert_anomaly_model_v3 import (
    DATA_PATH,
    OUTPUT_DIR,
    compute_baseline,
    extract_features,
    load_and_split_data,
)

def run_precision_experiments():
    print("=" * 85)
    print(" EXPERIMENT: PRECISION TARGET OPERATING POINTS (80%, 85%, 90%, 95%, 98%)")
    print("=" * 85)

    # 1. 3-Way Time-Sliced Split
    df_train_raw, df_val_raw, df_test_raw = load_and_split_data(DATA_PATH, train_ratio=0.70, val_ratio=0.15)
    global_baseline = compute_baseline(df_train_raw)

    train_features_df = extract_features(df_train_raw, global_baseline)
    val_features_df = extract_features(df_val_raw, global_baseline)
    test_features_df = extract_features(df_test_raw, global_baseline)

    exclude_cols = ["alert_id", "service_name", "timestamp", "label", "metric_count"]
    feature_cols = [c for c in train_features_df.columns if c not in exclude_cols]

    # 2. Train Novelty Isolation Forest
    train_normal_X = train_features_df[train_features_df["label"] == 1][feature_cols]
    iso_forest = joblib.load(os.path.join(OUTPUT_DIR, "isolation_forest_model_v3.pkl"))

    # 3. Scores on Validation and Test
    val_scores = -iso_forest.decision_function(val_features_df[feature_cols])
    y_val_anomaly = (val_features_df["label"] == 0).astype(int)

    test_scores = -iso_forest.decision_function(test_features_df[feature_cols])
    y_test_anomaly = (test_features_df["label"] == 0).astype(int)

    test_auc = roc_auc_score(y_test_anomaly, test_scores)
    val_auc = roc_auc_score(y_val_anomaly, val_scores)

    print(f"\nModel ROC-AUC -> Validation: {val_auc:.4f} | Test: {test_auc:.4f}")

    # 4. Dense grid search for candidate thresholds
    threshold_grid = np.linspace(val_scores.min(), val_scores.max(), 1000)
    
    val_candidates = []
    for t in threshold_grid:
        v_preds = (val_scores >= t).astype(int)
        cm_v = confusion_matrix(y_val_anomaly, v_preds)
        v_prec = cm_v[1, 1] / (cm_v[1, 1] + cm_v[0, 1]) if (cm_v[1, 1] + cm_v[0, 1]) > 0 else 0.0
        v_rec = cm_v[1, 1] / (cm_v[1, 1] + cm_v[1, 0]) if (cm_v[1, 1] + cm_v[1, 0]) > 0 else 0.0
        v_f1 = 2 * v_prec * v_rec / (v_prec + v_rec + 1e-8)
        val_candidates.append({"threshold": t, "v_prec": v_prec, "v_rec": v_rec, "v_f1": v_f1})

    val_df = pd.DataFrame(val_candidates)

    precision_targets = [0.80, 0.85, 0.90, 0.95, 0.98]
    operating_results = []

    print("\n" + "=" * 85)
    print(" OPERATING POINT COMPARISON TABLE")
    print("=" * 85)
    print(f"{'Target Prec':<13} {'Threshold':<12} {'Val Prec':<11} {'Val Rec':<11} {'Test Prec':<11} {'Test Rec':<11} {'Test F1':<10} {'Test Acc':<10}")
    print("-" * 85)

    for target_p in precision_targets:
        # Find all candidate thresholds where Validation Precision >= target_p
        eligible = val_df[val_df["v_prec"] >= target_p]
        if eligible.empty:
            continue
        
        # Select the threshold that maximizes Validation Recall subject to precision constraint
        best_candidate = eligible.sort_values(by=["v_rec", "v_f1"], ascending=[False, False]).iloc[0]
        t = best_candidate["threshold"]
        v_prec = best_candidate["v_prec"]
        v_rec = best_candidate["v_rec"]

        # Evaluate on Test Holdout
        t_preds = (test_scores >= t).astype(int)
        cm_t = confusion_matrix(y_test_anomaly, t_preds)
        t_prec = cm_t[1, 1] / (cm_t[1, 1] + cm_t[0, 1]) if (cm_t[1, 1] + cm_t[0, 1]) > 0 else 0.0
        t_rec = cm_t[1, 1] / (cm_t[1, 1] + cm_t[1, 0]) if (cm_t[1, 1] + cm_t[1, 0]) > 0 else 0.0
        t_f1 = 2 * t_prec * t_rec / (t_prec + t_rec + 1e-8)
        t_acc = (cm_t[0, 0] + cm_t[1, 1]) / len(y_test_anomaly)

        operating_results.append({
            "target_precision": f"{target_p:.0%}",
            "threshold": round(float(t), 4),
            "val_precision": round(float(v_prec), 4),
            "val_recall": round(float(v_rec), 4),
            "test_precision": round(float(t_prec), 4),
            "test_recall": round(float(t_rec), 4),
            "test_f1": round(float(t_f1), 4),
            "test_accuracy": round(float(t_acc), 4),
            "test_confusion_matrix": {
                "true_normal": int(cm_t[0, 0]),
                "false_anomaly": int(cm_t[0, 1]),
                "missed_anomaly": int(cm_t[1, 0]),
                "caught_anomaly": int(cm_t[1, 1])
            }
        })

        print(f"{target_p*100:>4.0f}%          {t:>10.4f}   {v_prec:>9.2%}   {v_rec:>9.2%}   {t_prec:>9.2%}   {t_rec:>9.2%}   {t_f1:>8.4f}   {t_acc:>8.2%}")

    print("=" * 85)
    print("\nDETAILED FINDINGS & TRADE-OFF STATEMENTS:")
    print("-" * 85)

    for res in operating_results:
        tp = res['target_precision']
        th = res['threshold']
        test_p = res['test_precision'] * 100
        test_r = res['test_recall'] * 100
        caught = res['test_confusion_matrix']['caught_anomaly']
        false_alarms = res['test_confusion_matrix']['false_anomaly']
        
        print(f"\n▶ Target Precision {tp} (Operating Threshold = {th}):")
        print(f"  \"At this operating point (Threshold = {th}), the model maintains approximately {test_p:.1f}% precision")
        print(f"   while detecting {test_r:.1f}% of anomalies ({caught}/73 real incidents caught, with only {false_alarms} false alarms out of 77 normal alerts).\"")

    # Save to JSON
    output_path = os.path.join(OUTPUT_DIR, "precision_target_experiments.json")
    with open(output_path, "w") as f:
        json.dump(operating_results, f, indent=2)
    print(f"\nResults exported to: {output_path}")

if __name__ == "__main__":
    run_precision_experiments()
