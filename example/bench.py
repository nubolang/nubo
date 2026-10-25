#!/usr/bin/env python3

import argparse
import re
import statistics
import subprocess


TIMING_PATTERN = re.compile(
    r"\(debug\) (Load/parse|Execution|Total) time:\s+([\d.]+)(ns|µs|us|ms|s)"
)


def to_ms(value: float, unit: str) -> float:
    match unit:
        case "ns":
            return value / 1_000_000
        case "µs" | "us":
            return value / 1_000
        case "ms":
            return value
        case "s":
            return value * 1_000
        case _:
            raise ValueError(f"Unknown time unit: {unit}")


def run_once(file: str) -> dict[str, float]:
    result = subprocess.run(
        ["./bin/nubo", file, "--stopwatch"],
        capture_output=True,
        text=True,
    )

    output = result.stdout + result.stderr

    if result.returncode != 0:
        print(output)
        raise RuntimeError(f"nubo exited with code {result.returncode}")

    timings = {}

    for name, value, unit in TIMING_PATTERN.findall(output):
        timings[name] = to_ms(float(value), unit)

    required = {"Load/parse", "Execution", "Total"}

    if timings.keys() != required:
        raise RuntimeError(
            f"Could not parse timings.\n\nOutput:\n{output}"
        )

    return timings


def print_stats(name: str, values: list[float]) -> None:
    print(
        f"{name:<12} "
        f"mean={statistics.mean(values):7.3f} ms  "
        f"median={statistics.median(values):7.3f} ms  "
        f"min={min(values):7.3f} ms  "
        f"max={max(values):7.3f} ms"
    )


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("file")
    parser.add_argument("-n", "--runs", type=int, default=100)
    parser.add_argument(
        "-w",
        "--warmup",
        type=int,
        default=5,
        help="Number of unmeasured warmup runs",
    )

    args = parser.parse_args()

    print(f"Warmup: {args.warmup} runs")

    for _ in range(args.warmup):
        run_once(args.file)

    print(f"Benchmark: {args.runs} runs\n")

    results = {
        "Load/parse": [],
        "Execution": [],
        "Total": [],
    }

    for i in range(args.runs):
        timings = run_once(args.file)

        for name, value in timings.items():
            results[name].append(value)

        print(
            f"\r{i + 1}/{args.runs} "
            f"load={timings['Load/parse']:.3f} ms "
            f"exec={timings['Execution']:.3f} ms "
            f"total={timings['Total']:.3f} ms",
            end="",
            flush=True,
        )

    print("\n")

    print_stats("Load/parse", results["Load/parse"])
    print_stats("Execution", results["Execution"])
    print_stats("Total", results["Total"])


if __name__ == "__main__":
    main()
