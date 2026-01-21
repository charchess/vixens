#!/usr/bin/env python3
import argparse
import sys
from pathlib import Path

from testing import config
from testing import runners
from testing import reporting
from testing import utils

def main():
    """
    Main entry point for the test runner.
    Parses arguments, loads configuration, and runs the selected tests.
    """
    parser = argparse.ArgumentParser(
        description="Vixens Project Test Runner.",
        formatter_class=argparse.RawTextHelpFormatter
    )
    parser.add_argument(
        'test_type',
        choices=['fonc', 'tech'],
        help="Type of test to run:\n             fonc - Functional tests (high-level checks)\n             tech - Technical tests (in-depth validation)"
    )
    parser.add_argument(
        'environment',
        choices=['dev', 'test', 'staging', 'prod'],
        help="Target environment for the tests."
    )
    parser.add_argument(
        '--tags',
        type=str,
        help="Comma-separated list of tags to run specific tests (e.g., 'terraform,network')."
    )
    parser.add_argument(
        '--output',
        choices=['console', 'json'],
        default='console',
        help="Output format."
    )
    parser.add_argument(
        '--no-color',
        action='store_true',
        help="Disable colored output."
    )

    args = parser.parse_args()

    # --- Pre-flight Checks ---
    if not utils.check_dependencies(['terraform', 'talosctl']):
        sys.exit(1)

    # --- Load Configuration ---
    try:
        reporter = reporting.Reporter(args.output, no_color=args.no_color)
        reporter.print_header(f"Loading configuration for environment: {args.environment}")
        env_config = config.load_config(args.environment)
        reporter.print_message("Configuration loaded successfully.", "green")
    except Exception as e:
        reporter.print_error(f"Failed to load configuration: {e}")
        sys.exit(1)

    # --- Run Tests ---
    reporter.print_header(f"Starting {args.test_type} tests for {args.environment}")

    test_tags = set(args.tags.split(',')) if args.tags else None
    
    test_results = runners.run_tests(
        test_type=args.test_type,
        config=env_config,
        reporter=reporter,
        tags=test_tags
    )

    # --- Print Summary ---
    summary = reporter.print_summary(test_results)
    
    if summary['failed'] > 0 or summary['errors'] > 0:
        sys.exit(1)

if __name__ == "__main__":
    main()
