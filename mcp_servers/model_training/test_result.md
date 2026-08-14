# 1. Experiment A — Original Synthetic Dataset

## Dataset

You originally generated:

Normal values + injected anomalous values

You had:

1000 alerts
955 normal
45 anomalous

Your alert-level features initially included:

- mean_deviation
- max_deviation
- outlier_count
- outlier_ratio
- mean_relative_change
- max_relative_change
- above_ratio
- below_ratio

You eventually used a subset for Isolation Forest:

```text
[
    "mean_deviation",
    "max_deviation",
    "outlier_ratio",
    "mean_relative_change",
    "max_relative_change",
    "above_ratio"
]
```

Train/test split

Train: 850
Test: 150

Class distribution:

Train:
0 = 75
1 = 775

Test:
0 = 20
1 = 130

Train
[[?]]

You didn't provide the original training confusion matrix for this experiment, but you provided the test result.

Test
[130  12]
[  0   8]

Classification report:

```text
                         precision    recall    f1-score   support

Normal / True Alert       1.00      0.92      0.96       142
Anomalous / False Alert   0.40      1.00      0.57         8

accuracy                           0.92       150
macro avg                  0.70      0.96      0.76       150
weighted avg               0.97      0.92      0.94       150
```

## Important observation

The model detected all anomalous test samples, but produced quite a few false anomaly predictions.

You then inspected:

False anomalies: 12
True anomalies: 8

The false anomalies generally had relatively normal-looking statistical features.

# 2. Experiment B — Added anomalous_metric_count and anomaly_concentration

You then added:

- anomalous_metric_count
- anomaly_concentration

Your full feature set became:

```text
[
    "mean_deviation",
    "max_deviation",
    "outlier_ratio",
    "mean_relative_change",
    "max_relative_change",
    "above_ratio",
    "below_ratio",
    "anomalous_metric_count",
    "anomaly_concentration"
]
```

The additional features were very strongly separated:

anomaly_concentration

label 0:
mean = 151.95

label 1:
mean = 0

and:

anomalous_metric_count

label 0:
mean = 3.54

label 1:
mean = 0

This made the anomaly detection problem substantially easier.

Experiment B — Train
[782  31]
[  0  37]

Classification report:

```text
              precision    recall    f1-score    support

0             1.00        0.96      0.98        813
1             0.54        1.00      0.70         37

accuracy                            0.96        850
macro avg     0.77        0.98      0.84        850
weighted avg  0.98        0.96      0.97        850
```

Experiment B — Test
[134   8]
[  0   8]

Classification report:

```text
                         precision    recall    f1-score    support

Normal / True Alert       1.00        0.94      0.97        142
Anomalous / False Alert   0.50        1.00      0.67          8

accuracy                             0.95        150
macro avg                 0.75        0.97      0.82        150
weighted avg              0.97        0.95      0.95        150
```

You then compared the misclassified samples.

False positives

8

Average:

- mean_deviation              0.4307
- max_deviation               0.8506
- outlier_ratio               0
- mean_relative_change       -0.1057
- max_relative_change         0.7017
- above_ratio                 0.3542
- below_ratio                 0.6458
- anomalous_metric_count      0
- anomaly_concentration       0

True positives

8

Average:

- mean_deviation              51.8171
- max_deviation               580.125
- outlier_ratio                0.1667
- mean_relative_change          9.5783
- max_relative_change          105.125
- above_ratio                   0.5521
- below_ratio                   0.4479
- anomalous_metric_count        2
- anomaly_concentration       187.6347

This showed that the new features were extremely informative.

# 3. Experiment C — Removed above_ratio and below_ratio

You then tested whether these two features were actually useful.

You removed:

- above_ratio
- below_ratio

So Set C became:

```text
[
    "mean_deviation",
    "max_deviation",
    "outlier_ratio",
    "mean_relative_change",
    "max_relative_change",
    "anomalous_metric_count",
    "anomaly_concentration"
]
```

## First Set C result

You initially got exactly the same result as Set B:

Train
[790  23]
[  0  37]

```text
              precision    recall    f1-score    support

0             1.00        0.97      0.99        813
1             0.62        1.00      0.76         37

accuracy                            0.97        850
macro avg     0.81        0.99      0.87        850
weighted avg  0.98        0.97      0.98        850
```

Test
[134   8]
[  0   8]

```text
                         precision    recall    f1-score    support

Normal / True Alert       1.00        0.94      0.97        142
Anomalous / False Alert   0.50        1.00      0.67          8

accuracy                             0.95        150
macro avg                 0.75        0.97      0.82        150
weighted avg              0.97        0.95      0.95        150
```

At this point you correctly noticed that the result was suspiciously good.

# 4. Experiment D — Regenerated Dataset, Set C

You then regenerated the synthetic dataset and reran Set C.

This changed the class distribution significantly.

Train
0 = 730
1 = 120
Test
0 = 125
1 = 25

This time:

Train
[728   2]
[  0 120]

```text
              precision    recall    f1-score    support

0             1.00        1.00      1.00        730
1             0.98        1.00      0.99        120

accuracy                            1.00        850
macro avg     0.99        1.00      1.00        850
weighted avg  1.00        1.00      1.00        850
```

Test
[124   1]
[  0  25]

```text
                         precision    recall    f1-score    support

Normal / True Alert       1.00        0.99      1.00        125
Anomalous / False Alert   0.96        1.00      0.98         25

accuracy                             0.99        150
macro avg                 0.98        1.00      0.99        150
weighted avg              0.99        0.99      0.99        150
```

This was your 99% experiment.

We then identified the reason: your synthetic anomalies were still very easily distinguishable from normal data.

# 5. Experiment E — Tougher Synthetic Dataset

You then changed the generator substantially.

The important changes were:

## More metric groups

You added:

```text
"logs": ["log_repetition_count"]
```

So you now had:

cpu
memory
traffic
errors
latency
availability
logs
Only some metrics in a group are perturbed

Instead of perturbing every metric in a group, you changed it to:

1–2 metrics per selected group

This made anomalies less obvious.

1–4 groups affected
num_affected_groups = random.choice([1, 2, 3, 4])
Severity distribution
subtle     50%
moderate   35%
extreme    15%

This was much more challenging.

# 6. Experiment E — Current result

Your latest dataset contains approximately:

1000 total alerts

with roughly balanced:

normal ≈ 50%
anomaly ≈ 50%

You ran the Set C feature configuration.

Train
[407   0]
[289 154]

Classification report:

```text
              precision    recall    f1-score    support

0             0.58        1.00      0.74        407
1             1.00        0.35      0.52        443

accuracy                            0.66        850
macro avg     0.79        0.67      0.63        850
weighted avg  0.80        0.66      0.62        850
```

Test
[77   0]
[46  27]

Classification report:

```text
                         precision    recall    f1-score    support

Normal / True Alert       0.63        1.00      0.77         77
Anomalous / False Alert   1.00        0.37      0.54         73

accuracy                             0.69        150
macro avg                 0.81        0.68      0.66        150
weighted avg              0.81        0.69      0.66        150
```

Important: based on your generator, 0 = anomalous and 1 = normal, so the labels "Normal / True Alert" and "Anomalous / False Alert" in this printed report appear to be reversed. The raw confusion matrix itself is unambiguous.

# 7. Overall comparison

| Experiment | Dataset | Feature set | Train Acc. | Test Acc. | Test Precision (Anomaly) | Test Recall (Anomaly) | Test F1 (Anomaly) |
| --- | --- | --- | --- | --- | --- | --- | --- |
| **A** | Original synthetic | 6 features | — | 92% | 40% | 100% | 0.57 |
| **B** | Original regenerated | 9 features | 96% | 95% | 50% | 100% | 0.67 |
| **C-1** | Original regenerated | 7 features, removed ratios | 97% | 95% | 50% | 100% | 0.67 |
| **E** | Tough/hard synthetic | 7 features (IQR-based) | 66% | 69% | 100% | 37%* | 0.54* |
| **F (v2 - Balanced)** | Tough/hard synthetic | 19 continuous features | 90% | 83% | **98%** | **67%** | **0.80** |
| **G-1 (v3 - Target 80% Prec)** | Tough/hard synthetic | 19 continuous features (Val Prec $\ge 80\%$ constraint) | 80% | 75% | **73%** *(Val: 81%)* | **77%** *(Val: 79%)* | **0.75** *(Val: 0.80)* |
| **G-2 (v3 - Target 85% Prec)** | Tough/hard synthetic | 19 continuous features (Val Prec $\ge 85\%$ constraint) | 81% | 80% | **82%** *(Val: 86%)* | **75%** *(Val: 77%)* | **0.79** *(Val: 0.81)* |
| **G-3 (v3 - Target 90% Prec)** | Tough/hard synthetic | 19 continuous features (Val Prec $\ge 90\%$ constraint) | 82% | 79% | **82%** *(Val: 90%)* | **73%** *(Val: 76%)* | **0.77** *(Val: 0.82)* |

# 8. Experiment F — Continuous Deviation & Novelty Isolation Forest (Model v2)

Script: `alert_anomaly_model_v2.py`

### 3-Way Time-Sliced Data Splitting:
- **Train Split (70%)**: 700 rows (`2026-01-01 00:15` to `2026-01-22 11:03`) — Used to compute robust baselines and fit Isolation Forest trees on clean normal samples (`label == 1`).
- **Validation Split (15%)**: 150 rows (`2026-01-22 15:01` to `2026-01-27 12:09`) — Used to calibrate decision thresholds (Balanced: `0.0602`, High Recall: `-0.0720`) on the Precision-Recall curve without test leakage.
- **Test Split (15%)**: 150 rows (`2026-01-27 13:06` to `2026-01-31 23:49`) — Unbiased future holdout evaluation simulating production.

### Key Improvements:
1. **Continuous Robust Z-Scores**: Replaced binary 1.5*IQR clipping with continuous $\text{Robust Z} = |x - \text{median}| / (1.4826 \times \text{MAD})$.
2. **Top-K Order Statistics & Anomaly Energy**: Added `robust_z_max`, `robust_z_top2_mean`, `robust_z_top3_mean`, `top3_anomaly_energy` to prevent metric dilution in dynamic alert payloads.
3. **Domain Group Invariants**: Added 7 domain group max deviations (`group_cpu_max_z`, `group_traffic_max_z`, etc.) with `0.0` neutral padding for absent metric groups.
4. **Novelty Detection Mode**: Isolation Forest fitted exclusively on clean normal training telemetry (`label == 1`).
5. **Threshold Calibration on Validation Set**: Tuned classification threshold on `-decision_function(X)` via PR curve without leaking Test Set.

### Validation Set Results (150 alerts: 79 Normal, 71 Anomalous):
- **Validation ROC-AUC**: `0.8627`
- **Balanced Threshold (`0.0602`)**: Val F1 = `0.8320`, Recall = `73.24%`, Precision = `96.30%`
- **High Recall Threshold (`-0.0720`)**: Val F1 = `0.7093`, Recall = `85.92%`, Precision = `60.40%`

### Unbiased Test Set Results (150 alerts: 77 Normal, 73 Anomalous):

**Balanced Operating Threshold (0.0602)**:
- Test ROC-AUC: **0.8404**
- Accuracy: **83.33%**

Confusion Matrix:
```text
                    Pred Normal (0)  Pred Anomaly (1)
Actual Normal (0)                76                 1
Actual Anomaly (1)               24                49
```

Classification Report:
```text
                 precision    recall  f1-score   support

   Normal Alert     0.7600    0.9870    0.8588        77
Anomalous Alert     0.9800    0.6712    0.7967        73

       accuracy                         0.8333       150
      macro avg     0.8700    0.8291    0.8278       150
   weighted avg     0.8671    0.8333    0.8286       150
```

**High Recall Operating Threshold (-0.0720)**:
- Anomaly Recall: **83.56%** (61 of 73 anomalies detected)
- Anomaly Precision: **57.01%**
- F1-Score: **0.6778**

# 9. Experiment G — Precision-Constrained Balanced Model (Model v3)

Script: `alert_anomaly_model_v3.py`

### Goal:
Calibrate balanced operating thresholds specifically constrained to Target Precision levels for interactive human-in-the-loop support investigation.

---

### Option 1: Target Precision $\ge 80\%$ (High Recall Priority)

* **Threshold**: `-0.0363`
* **Validation Performance**:
  * Validation Precision: **81.16%**
  * Validation Recall: **78.87%**
  * Validation F1-Score: **0.8000**
  * Validation ROC-AUC: **0.8627**
* **Unbiased Test Set Evaluation (150 alerts: 77 Normal, 73 Anomalous)**:
  * **Test Accuracy**: **74.67%**
  * **Test Precision (Anomaly)**: **72.73%**
  * **Test Recall (Anomaly)**: **76.71%** (56 of 73 anomalies caught)
  * **Test F1-Score (Anomaly)**: **0.7467**
  * **Test ROC-AUC**: **0.8404**

Confusion Matrix:
```text
                    Pred Normal (0)  Pred Anomaly (1)
Actual Normal (0)                56                21
Actual Anomaly (1)               17                56
```

Classification Report:
```text
                 precision    recall  f1-score   support

   Normal Alert     0.7671    0.7273    0.7467        77
Anomalous Alert     0.7273    0.7671    0.7467        73

       accuracy                         0.7467       150
      macro avg     0.7472    0.7472    0.7467       150
      weighted avg     0.7477    0.7467    0.7467       150
```

---

### Option 2: Target Precision $\ge 85\%$ (Balanced Accuracy & F1 Sweet Spot — Top Performer)

* **Threshold**: `-0.0188`
* **Validation Performance**:
  * Validation Precision: **85.94%**
  * Validation Recall: **77.46%**
  * Validation F1-Score: **0.8148**
  * Validation ROC-AUC: **0.8627**
* **Unbiased Test Set Evaluation (150 alerts: 77 Normal, 73 Anomalous)**:
  * **Test Accuracy**: **80.00%**
  * **Test Precision (Anomaly)**: **82.09%**
  * **Test Recall (Anomaly)**: **75.34%** (55 of 73 anomalies caught)
  * **Test F1-Score (Anomaly)**: **0.7857**
  * **Test ROC-AUC**: **0.8404**

Confusion Matrix:
```text
                    Pred Normal (0)  Pred Anomaly (1)
Actual Normal (0)                65                12
Actual Anomaly (1)               18                55
```

Classification Report:
```text
                 precision    recall  f1-score   support

   Normal Alert     0.7831    0.8442    0.8125        77
Anomalous Alert     0.8209    0.7534    0.7857        73

       accuracy                         0.8000       150
      macro avg     0.8020    0.7988    0.7991       150
      weighted avg     0.8015    0.8000    0.7995       150
```

---

### Option 3: Target Precision $\ge 90\%$ (High Precision Priority — Recommended)

* **Threshold**: `-0.0109`
* **Validation Performance**:
  * Validation Precision: **90.00%**
  * Validation Recall: **76.06%**
  * Validation F1-Score: **0.8244**
  * Validation ROC-AUC: **0.8627**
* **Unbiased Test Set Evaluation (150 alerts: 77 Normal, 73 Anomalous)**:
  * **Test Accuracy**: **78.67%**
  * **Test Precision (Anomaly)**: **81.54%**
  * **Test Recall (Anomaly)**: **72.60%** (53 of 73 anomalies caught)
  * **Test F1-Score (Anomaly)**: **0.7681**
  * **Test ROC-AUC**: **0.8404**

Confusion Matrix:
```text
                    Pred Normal (0)  Pred Anomaly (1)
Actual Normal (0)                65                12
Actual Anomaly (1)               20                53
```

Classification Report:
```text
                 precision    recall  f1-score   support

   Normal Alert     0.7647    0.8442    0.8025        77
Anomalous Alert     0.8154    0.7260    0.7681        73

       accuracy                         0.7867       150
      macro avg     0.7900    0.7851    0.7853       150
      weighted avg     0.7894    0.7867    0.7858       150
```

### Exported Artifacts in `mcp_servers/server_1/`:
- `isolation_forest_model_v3.pkl`
- `baseline_reference_v3.json`
- `model_metadata_v3.json`

# 10. Experiment H — Precision Target Exploration (80%, 85%, 90%, 95%, 98%)

Script: `experiment_precision_targets.py`

### Operating Point Comparison Table:

| Target Precision | Threshold | Validation Precision | Validation Recall | Test Precision (Anomaly) | Test Recall (Anomaly) | Test F1-Score | Test Accuracy |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **80%** | `-0.0370` | 81.16% | 78.87% | **72.73%** | **76.71%** (56/73) | 0.7467 | 74.67% |
| **85%** | `-0.0295` | 85.94% | 77.46% | **78.57%** | **75.34%** (55/73) | 0.7692 | 78.00% |
| **90%** | `-0.0117` | 90.00% | 76.06% | **81.54%** | **72.60%** (53/73) | 0.7681 | 78.67% |
| **95%** | `+0.0531` | 96.30% | 73.24% | **98.00%** | **67.12%** (49/73) | 0.7967 | 83.33% |
| **98%** | `+0.0626` | 98.08% | 71.83% | **98.00%** | **67.12%** (49/73) | 0.7967 | 83.33% |

### Key Findings & Trade-off Statements:

1. **Target 80% (Threshold = `-0.0370`)**:
   > *"At this operating point, the model maintains approximately **72.7%** test precision while detecting **76.7%** of anomalies (56/73 real incidents caught, with 21 false alarms out of 77 normal alerts)."*

2. **Target 85% (Threshold = `-0.0295`)**:
   > *"At this operating point, the model maintains approximately **78.6%** test precision while detecting **75.3%** of anomalies (55/73 real incidents caught, with only 15 false alarms out of 77 normal alerts)."*

3. **Target 90% (Threshold = `-0.0117`)**:
   > *"At this operating point, the model maintains approximately **81.5%** test precision while detecting **72.6%** of anomalies (53/73 real incidents caught, with only 12 false alarms out of 77 normal alerts)."*

4. **Target 95% (Threshold = `+0.0531`)**:
   > *"At this operating point, the model maintains approximately **98.0%** test precision while detecting **67.1%** of anomalies (49/73 real incidents caught, with only 1 false alarm out of 77 normal alerts)."*