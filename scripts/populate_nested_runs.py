# /// script
# requires-python = ">=3.13"
# dependencies = []
# ///
"""
Create example nested run trees to demonstrate the nested runs feature.

Creates a hyperparameter sweep structure:
- Level 0: Experiment configurations (e.g., different model architectures)
- Level 1: Hyperparameter sets within each configuration
- Level 2: Individual trial runs with different random seeds
"""

import sys
import os
import random
from pathlib import Path

# Add parent directory to path to import apparatus
sys.path.insert(0, str(Path(__file__).parent.parent / "logging"))

import apparatus


def main():
    print("Creating nested run examples...")

    # Get tracking URI from environment or use default
    tracking_uri = os.environ.get("APPARATUS_TRACKING_URI", "http://localhost:8080")

    # Create two top-level "experiment" runs representing different model architectures
    architectures = [
        {"name": "ResNet-50", "layers": 50, "base_lr": 0.1},
        {"name": "VGG-16", "layers": 16, "base_lr": 0.01},
    ]

    for arch in architectures:
        print(f"\n=== Creating run tree for {arch['name']} ===")

        # Level 0: Architecture run
        arch_run_uuid = apparatus.create_run(
            f"arch/{arch['name']}",
            tracking_uri=tracking_uri
        )
        print(f"  Created architecture run: {arch['name']}")

        apparatus.log_param(arch_run_uuid, "architecture", arch["name"], tracking_uri=tracking_uri)
        apparatus.log_param(arch_run_uuid, "num_layers", arch["layers"], tracking_uri=tracking_uri)

        # Level 1: Different hyperparameter configurations
        hp_configs = [
            {"lr_mult": 1.0, "batch_size": 32, "optimizer": "SGD"},
            {"lr_mult": 0.5, "batch_size": 64, "optimizer": "SGD"},
            {"lr_mult": 1.0, "batch_size": 32, "optimizer": "Adam"},
        ]

        for hp_idx, hp in enumerate(hp_configs):
            lr = arch["base_lr"] * hp["lr_mult"]
            hp_name = f"hp/{hp['optimizer']}_lr{lr}_bs{hp['batch_size']}"

            # Level 1: Hyperparameter run
            hp_run_uuid = apparatus.create_run(
                hp_name,
                parent_run_uuid=arch_run_uuid,
                tracking_uri=tracking_uri
            )
            print(f"    Created HP run: {hp_name}")

            apparatus.log_param(hp_run_uuid, "learning_rate", lr, tracking_uri=tracking_uri)
            apparatus.log_param(hp_run_uuid, "batch_size", hp["batch_size"], tracking_uri=tracking_uri)
            apparatus.log_param(hp_run_uuid, "optimizer", hp["optimizer"], tracking_uri=tracking_uri)

            # Level 2: Multiple trial runs with different seeds
            num_trials = 2
            for seed in range(num_trials):
                trial_name = f"trial/seed_{seed}"

                # Level 2: Trial run
                trial_run_uuid = apparatus.create_run(
                    trial_name,
                    parent_run_uuid=hp_run_uuid,
                    tracking_uri=tracking_uri
                )
                print(f"      Created trial run: {trial_name}")

                apparatus.log_param(trial_run_uuid, "seed", seed, tracking_uri=tracking_uri)

                # Simulate training metrics
                random.seed(seed + hp_idx * 100 + architectures.index(arch) * 1000)
                base_loss = 2.0 - (hp["lr_mult"] * 0.3) - (0.1 if hp["optimizer"] == "Adam" else 0)

                epochs = list(range(10))
                losses = [base_loss * (0.9 ** e) + random.uniform(-0.05, 0.05) for e in epochs]
                accuracies = [min(0.95, 0.5 + 0.05 * e + random.uniform(-0.02, 0.02)) for e in epochs]

                apparatus.log_metrics(
                    trial_run_uuid, "loss",
                    x_values=[float(e) for e in epochs],
                    y_values=losses,
                    tracking_uri=tracking_uri
                )
                apparatus.log_metrics(
                    trial_run_uuid, "accuracy",
                    x_values=[float(e) for e in epochs],
                    y_values=accuracies,
                    tracking_uri=tracking_uri
                )

    print("\n" + "=" * 50)
    print("Created nested run structure:")
    print("  - 2 architecture runs (level 0)")
    print("  - 3 hyperparameter runs each (level 1)")
    print("  - 2 trial runs each (level 2)")
    print("  - Total: 2 + 6 + 12 = 20 runs")
    print("\nVisit http://localhost:8080 to view the nested runs!")


if __name__ == "__main__":
    main()
