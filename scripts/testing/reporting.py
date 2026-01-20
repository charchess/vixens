import json
from termcolor import colored

class Reporter:
    """Handles all output for the test runner."""

    def __init__(self, output_format='console', no_color=False):
        self.output_format = output_format
        self.no_color = no_color or output_format == 'json'

    def _color(self, message, color=None, on_color=None, attrs=None):
        """A wrapper for termcolor.colored that respects the no_color flag."""
        if self.no_color:
            return message
        return colored(message, color, on_color, attrs)

    def print_header(self, message):
        if self.output_format == 'console':
            print(self._color(f"\n{'='*10} {message} {'='*10}", 'cyan', attrs=['bold']))

    def print_message(self, message, color='white'):
        if self.output_format == 'console':
            print(self._color(message, color))

    def print_error(self, message):
         if self.output_format == 'console':
            print(self._color(f"ERROR: {message}", 'red', attrs=['bold']))

    def record_test_result(self, test_name, success, message="", details=None):
        """
        Prints and returns a structured result for a single test.
        """
        if self.output_format == 'console':
            status = self._color('[PASS]', 'green') if success else self._color('[FAIL]', 'red')
            print(f"{status} {test_name}: {message}")
            if details and not success:
                print(self._color(f"      Details: {details}", "yellow"))
        
        return {
            "test_name": test_name,
            "success": success,
            "message": message,
            "details": details
        }

    def print_summary(self, results):
        """
        Prints a final summary of all test results.
        """
        passed = sum(1 for r in results if r['success'])
        failed = len(results) - passed
        
        summary_data = {
            "total": len(results),
            "passed": passed,
            "failed": failed,
            "errors": 0 # Placeholder for now
        }

        if self.output_format == 'json':
            # In JSON mode, print the full results and summary
            full_report = {
                "summary": summary_data,
                "results": results
            }
            print(json.dumps(full_report, indent=2))
        else:
            self.print_header("Test Summary")
            print(f"Total tests: {summary_data['total']}")
            print(self._color(f"Passed: {summary_data['passed']}", 'green'))
            print(self._color(f"Failed: {summary_data['failed']}", 'red'))
        
        return summary_data

