# /// script
# requires-python = ">=3.13"
# dependencies = ["numpy", "matplotlib"]
# ///
"""
Generate Lorenz attractor data and log to Apparatus.

Integrates the Lorenz system using RK4 time integrator and logs multiple runs
with different parameters, including trajectory plots.
"""

import sys
from pathlib import Path
import tempfile

# Add parent directory to path to import apparatus
sys.path.insert(0, str(Path(__file__).parent.parent / "logging"))

import apparatus
import numpy as np
import matplotlib
matplotlib.use('Agg')  # Non-interactive backend
import matplotlib.pyplot as plt
from mpl_toolkits.mplot3d import Axes3D



def lorenz_derivatives(state, sigma, rho, beta):
    """Compute derivatives of the Lorenz system."""
    x, y, z = state
    dx = sigma * (y - x)
    dy = x * (rho - z) - y
    dz = x * y - beta * z
    return np.array([dx, dy, dz])


def rk4_step(state, dt, sigma, rho, beta):
    """Single RK4 integration step."""
    k1 = lorenz_derivatives(state, sigma, rho, beta)
    k2 = lorenz_derivatives(state + 0.5 * dt * k1, sigma, rho, beta)
    k3 = lorenz_derivatives(state + 0.5 * dt * k2, sigma, rho, beta)
    k4 = lorenz_derivatives(state + dt * k3, sigma, rho, beta)
    return state + (dt / 6.0) * (k1 + 2*k2 + 2*k3 + k4)


def integrate_lorenz(initial_state, sigma, rho, beta, t_max=30.0, dt=0.01):
    """Integrate Lorenz system using RK4."""
    num_steps = int(t_max / dt)
    trajectory = np.zeros((num_steps + 1, 3))
    times = np.zeros(num_steps + 1)

    trajectory[0] = initial_state
    times[0] = 0.0

    state = initial_state.copy()
    for i in range(num_steps):
        state = rk4_step(state, dt, sigma, rho, beta)
        trajectory[i + 1] = state
        times[i + 1] = (i + 1) * dt

    return times, trajectory


def create_3d_plot(times, trajectory, title, filepath):
    """Create a 3D trajectory plot using matplotlib."""
    fig = plt.figure(figsize=(10, 8))
    ax = fig.add_subplot(111, projection='3d')

    ax.plot(trajectory[:, 0], trajectory[:, 1], trajectory[:, 2],
            linewidth=0.5, alpha=0.8)
    ax.set_xlabel('X')
    ax.set_ylabel('Y')
    ax.set_zlabel('Z')
    ax.set_title(title)

    plt.savefig(filepath, dpi=150, bbox_inches='tight')
    plt.close()
    return True


def create_trace_plot(times, trajectory, i, filepath):
    dim = ['X', 'Y', 'Z'][i]
    fig = plt.figure(figsize=(10, 8))
    ax = fig.add_subplot(111)

    ax.plot(times, trajectory[:, i])
    ax.set_xlabel('t')
    ax.set_ylabel(dim)
    ax.set_title(dim)

    plt.savefig(filepath, dpi=150, bbox_inches='tight')
    plt.close()
    return True

def main():
    print("Generating Lorenz attractor runs...")

    # Get tracking URI from environment or use default
    import os
    tracking_uri = os.environ.get("APPARATUS_TRACKING_URI", "http://localhost:8080")

    # Define 10 different configurations
    configs = [
        # Classic Lorenz attractor
        {"sigma": 10.0, "rho": 28.0, "beta": 8/3, "x0": 1.0, "y0": 1.0, "z0": 1.0},
        {"sigma": 10.0, "rho": 28.0, "beta": 8/3, "x0": 0.0, "y0": 1.0, "z0": 1.05},
        {"sigma": 10.0, "rho": 28.0, "beta": 8/3, "x0": -1.0, "y0": -1.0, "z0": 1.0},

        # Vary rho
        {"sigma": 10.0, "rho": 20.0, "beta": 8/3, "x0": 1.0, "y0": 1.0, "z0": 1.0},
        {"sigma": 10.0, "rho": 24.0, "beta": 8/3, "x0": 1.0, "y0": 1.0, "z0": 1.0},
        {"sigma": 10.0, "rho": 35.0, "beta": 8/3, "x0": 1.0, "y0": 1.0, "z0": 1.0},

        # Vary sigma
        {"sigma": 8.0, "rho": 28.0, "beta": 8/3, "x0": 1.0, "y0": 1.0, "z0": 1.0},
        {"sigma": 12.0, "rho": 28.0, "beta": 8/3, "x0": 1.0, "y0": 1.0, "z0": 1.0},

        # Vary beta
        {"sigma": 10.0, "rho": 28.0, "beta": 2.0, "x0": 1.0, "y0": 1.0, "z0": 1.0},
        {"sigma": 10.0, "rho": 28.0, "beta": 3.0, "x0": 1.0, "y0": 1.0, "z0": 1.0},
    ]

    for i, config in enumerate(configs):
        run_name = f"lorenz_run_{i+1:02d}"
        print(f"\nRun {i+1}/10: {run_name}")

        # Create run
        run_uuid = apparatus.create_run(run_name, tracking_uri=tracking_uri)

        # Log parameters
        apparatus.log_param(run_uuid, "sigma", config["sigma"], tracking_uri=tracking_uri)
        apparatus.log_param(run_uuid, "rho", config["rho"], tracking_uri=tracking_uri)
        apparatus.log_param(run_uuid, "beta", config["beta"], tracking_uri=tracking_uri)
        apparatus.log_param(run_uuid, "x0", config["x0"], tracking_uri=tracking_uri)
        apparatus.log_param(run_uuid, "y0", config["y0"], tracking_uri=tracking_uri)
        apparatus.log_param(run_uuid, "z0", config["z0"], tracking_uri=tracking_uri)

        # Integrate
        initial_state = np.array([config["x0"], config["y0"], config["z0"]])
        times, trajectory = integrate_lorenz(
            initial_state,
            config["sigma"],
            config["rho"],
            config["beta"]
        )

        # Log metrics - sample every 100 steps to avoid too many data points
        sample_interval = 10
        apparatus.log_metrics(run_uuid, "x", y_values=trajectory[:, 0], x_values=times, tracking_uri=tracking_uri)
        apparatus.log_metrics(run_uuid, "y", y_values=trajectory[:, 1], x_values=times, tracking_uri=tracking_uri)
        apparatus.log_metrics(run_uuid, "z", y_values=trajectory[:, 2], x_values=times, tracking_uri=tracking_uri)

        # Create and upload plot
        with tempfile.NamedTemporaryFile(suffix='.png') as f:
            plot_path = f.name

            plot_title = f"σ={config['sigma']}, ρ={config['rho']}, β={config['beta']:.2f}"
            if create_3d_plot(times, trajectory, plot_title, plot_path):
                apparatus.log_artifact(run_uuid, "plots/trajectory.png", plot_path, tracking_uri=tracking_uri)
                print(f"  ✓ Logged trajectory plot")

            for i in range(3):
                create_trace_plot(times, trajectory, i, plot_path)
                dim = ['X', 'Y', 'Z'][i]
                apparatus.log_artifact(run_uuid, f"plots/traces/{dim}.png", plot_path, tracking_uri=tracking_uri)

        print(f"  ✓ Logged {len(times)//sample_interval} metric points")

    print("\n✓ All runs completed successfully!")


if __name__ == "__main__":
    main()

