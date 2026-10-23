"""Black-box stderr comparisons retain diagnostic counts and bodies."""
import unittest
from types import SimpleNamespace
from stackql_test_tooling.StackQLInterfaces import StackQLInterfaces


class DiagnosticComparisonTest(unittest.TestCase):
    def compare(self, actual, expected, platform="native"):
        def equal(left, right, **_):
            self.assertEqual(left, right)
            return True
        host = SimpleNamespace(_execution_platform=platform, should_be_equal=equal)
        result = SimpleNamespace(stdout="same rows", stderr=actual)
        return StackQLInterfaces._verify_both_streams(
            host, result, "same rows", expected, unordered_http_diagnostics=True)

    def test_only_request_completion_order_is_ignored(self):
        one = 'http response status code: 400, response body: {\n  "id": "one"\n}'
        two = 'http response status code: 404, response body is nil'
        self.assertTrue(self.compare(one + "\n" + two, two + "\n" + one))

    def test_missing_duplicate_diagnostic_is_rejected(self):
        entry = 'http response status code: 404, response body is nil'
        with self.assertRaises(AssertionError):
            self.compare(entry, entry + "\n" + entry)

    def test_changed_response_body_is_rejected(self):
        with self.assertRaises(AssertionError):
            self.compare('http response status code: 400, response body: one',
                         'http response status code: 400, response body: two')

    def test_docker_launcher_noise_is_ignored(self):
        one = 'http response status code: 400, response body: {\n  "id": "one"\n}'
        two = 'http response status code: 404, response body is nil'
        noise = ('time="2026-10-24T05:17:06Z" level=warning msg="The \\"UID\\" variable is not set. Defaulting to a blank string."\n'
                 'time="2026-10-24T05:17:06Z" level=warning msg="/work/docker-compose.yml: the attribute `version` is obsolete, it will be ignored"\n'
                 ' Container execrun-postgres_stackql-1  Running\n'
                 ' Network execrun_default  Created\n')
        self.assertTrue(self.compare(noise + two + "\n" + one, one + "\n" + two, "docker"))
        self.assertTrue(self.compare(one + "\n" + noise + two, two + "\n" + one, "docker"))
        for actual in (noise + one, noise + one + "\n" + two + "\n" + two,
                       noise + one.replace('"one"', '"changed"') + "\n" + two,
                       noise + one + "\n" + two + "\nunexpected application error"):
            with self.assertRaises(AssertionError):
                self.compare(actual, one + "\n" + two, "docker")


if __name__ == "__main__":
    unittest.main()
