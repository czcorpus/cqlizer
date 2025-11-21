#!/usr/bin/env python3
"""Train LightGBM model for CQL query performance classification."""

import argparse
import json
import os

import lightgbm as lgb
import msgpack
import numpy as np
from sklearn.metrics import auc, classification_report, precision_recall_curve
from sklearn.model_selection import train_test_split


def load_msgpack_features(path: str) -> tuple[np.ndarray, np.ndarray]:
    """Load features from msgpack file.

    Adjust unpacking based on your actual msgpack structure.
    """
    with open(path, "rb") as f:
        data = msgpack.unpack(f)
    X = np.array([item for item in data["features"]])
    y = np.array([item for item in data["label"]])
    return X, y


def train_model(X: np.ndarray, y: np.ndarray, output_path: str):
    """Train LightGBM and save model."""

    X_train, X_test, y_train, y_test = train_test_split(
        X, y, test_size=0.2, random_state=42, stratify=y
    )

    # Calculate scale_pos_weight for class imbalance (your 1-5% slow queries)
    neg_count = np.sum(y_train == 0)
    pos_count = np.sum(y_train == 1)
    scale_pos_weight = neg_count / pos_count

    print(f"Scale pos weight: {scale_pos_weight:.2f}")

    params = {
        "objective": "binary",
        "metric": ["auc", "binary_logloss"],
        "scale_pos_weight": scale_pos_weight,
        "max_depth": 6,
        "learning_rate": 0.05,
        "num_leaves": 81,
        "min_child_samples": 20,
        "subsample": 0.8,
        "colsample_bytree": 0.8,
        "random_state": 42,
        "verbose": -1,
    }

    train_data = lgb.Dataset(X_train, label=y_train)
    valid_data = lgb.Dataset(X_test, label=y_test, reference=train_data)

    model = lgb.train(
        params,
        train_data,
        num_boost_round=200,
        valid_sets=[train_data, valid_data],
        valid_names=["train", "valid"],
        callbacks=[
            lgb.early_stopping(stopping_rounds=20),
            lgb.log_evaluation(period=10),
        ],
    )

    # Evaluate
    y_prob = model.predict(X_test, num_iteration=model.best_iteration)
    y_pred = (y_prob > 0.5).astype(int)

    print("\nClassification Report:")
    print(classification_report(y_test, y_pred, target_names=["normal", "slow"]))

    # PR-AUC (more meaningful than ROC-AUC for imbalanced data)
    precision, recall, _ = precision_recall_curve(y_test, y_prob)
    pr_auc = auc(recall, precision)
    print(f"PR-AUC: {pr_auc:.4f}")

    # Feature importance
    print("\nTop 10 Feature Importances (gain):")
    importance = model.feature_importance(importance_type="gain")
    indices = np.argsort(importance)[::-1][:10]
    for i, idx in enumerate(indices):
        print(f"  {i + 1}. Feature {idx}: {importance[idx]:.4f}")

    # Save model in text format (compatible with leaves)
    model.save_model(output_path)
    print(f"\nModel saved to: {output_path}")
    print(f"Best iteration: {model.best_iteration}")

    with open(os.path.splitext(output_path)[0] + ".metadata.json", "w") as fw:
        json.dump(params, fw)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        description="Train LightGBM for CQL classification"
    )
    parser.add_argument("--input", "-i", required=True, help="Path to msgpack features")
    parser.add_argument(
        "--output",
        "-o",
        default="model.txt",
        help="Output model path (.txt for leaves compatibility)",
    )
    args = parser.parse_args()

    X, y = load_msgpack_features(args.input)
    print(f"Loaded {len(X)} samples, {X.shape[1]} features")
    print(
        f"Class distribution: {np.sum(y == 0)} normal, {np.sum(y == 1)} slow ({100 * np.mean(y):.2f}% positive)"
    )
    train_model(X, y, args.output)
