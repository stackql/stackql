"""Render a robot output.xml as a GitHub step summary traffic light.

Usage: python robot_summary.py --output test/robot/reports/output.xml --title "Omni readiness"

Prints GitHub-flavoured markdown to stdout; append it to $GITHUB_STEP_SUMMARY.
Exits 0 regardless of the pass rate, so it never decides a job's outcome.
"""
import argparse
import collections
import sys

from robot.api import ExecutionResult, ResultVisitor


_MAX_LISTED_FAILURES = 50


class _Collector(ResultVisitor):
    def __init__(self):
        self.passed = 0
        self.failed = 0
        self.skipped = 0
        self.failures = []
        self.failed_by_suite = collections.Counter()

    def visit_test(self, test):
        if test.passed:
            self.passed += 1
        elif test.skipped:
            self.skipped += 1
        else:
            self.failed += 1
            self.failures.append((test.parent.name, test.name, (test.message or '').splitlines()[0] if test.message else ''))
            self.failed_by_suite[test.parent.name] += 1


def _light(rate):
    if rate >= 100.0:
        return '🟢'
    if rate >= 90.0:
        return '🟡'
    return '🔴'


def _escape(text):
    return text.replace('|', '\\|').replace('\n', ' ')


def render(result_path, title, context):
    result = ExecutionResult(result_path)
    collector = _Collector()
    result.visit(collector)
    total = collector.passed + collector.failed
    rate = (100.0 * collector.passed / total) if total else 0.0
    light = _light(rate)
    lines = [
        f'# {light} {title}',
        '',
    ]
    if context:
        lines += [f'_{context}_', '']
    lines += [
        f'**Pass rate:** {rate:.1f}% · **Passed:** {collector.passed} · **Failed:** {collector.failed} · **Total:** {total}',
        '',
        '## Summary',
        '',
        '| Outcome | Count |',
        '|---|---:|',
        f'| 🟢 Passed | {collector.passed} |',
        f'| 🔴 Failed | {collector.failed} |',
        f'| ⚪ Skipped | {collector.skipped} |',
        f'| **Total** | **{total}** |',
        '',
    ]
    if collector.failed_by_suite:
        lines += ['## Failures by suite', '', '| Suite | Failed |', '|---|---:|']
        for suite, count in collector.failed_by_suite.most_common():
            lines.append(f'| {_escape(suite)} | {count} |')
        lines.append('')
    if collector.failures:
        shown = collector.failures[:_MAX_LISTED_FAILURES]
        lines += [
            '<details>',
            f'<summary>Failed tests ({len(collector.failures)}{", first " + str(_MAX_LISTED_FAILURES) + " shown" if len(collector.failures) > _MAX_LISTED_FAILURES else ""})</summary>',
            '',
            '| Suite | Test | Message |',
            '|---|---|---|',
        ]
        for suite, test, message in shown:
            lines.append(f'| {_escape(suite)} | {_escape(test)} | {_escape(message)[:200]} |')
        lines += ['', '</details>', '']
    return '\n'.join(lines)


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output', required=True, help='robot output.xml path')
    parser.add_argument('--title', default='Robot results')
    parser.add_argument('--context', default='', help='one line shown under the title')
    args = parser.parse_args(argv)
    sys.stdout.write(render(args.output, args.title, args.context))
    return 0


if __name__ == '__main__':
    sys.exit(main())
