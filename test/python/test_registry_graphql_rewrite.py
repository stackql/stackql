import os
from pathlib import Path
import tempfile
import unittest

import yaml
from stackql_test_tooling.registry_rewrite import ProviderArgs, rewrite_provider


class GraphQLEndpointRewriteTests(unittest.TestCase):
    def test_rewrite_preserves_paths_and_covers_every_operation(self):
        previous = os.getcwd()
        with tempfile.TemporaryDirectory() as root:
            source = Path(root) / "source"
            target = Path(root) / "target"
            source.mkdir()
            target.mkdir()
            paths = {}
            for path in ("/graphql", "/graphql/keyset", "/v2/query"):
                paths[path] = {"post": {"x-stackQL-graphQL": {
                    "url": "https://provider.example" + path + "?tenant=x"
                }}}
            (source / "service.yaml").write_text(yaml.safe_dump({"paths": paths}))
            try:
                rewrite_provider(ProviderArgs("p", str(source), str(target), 1234,
                                              replacement_host="mock.example"))
            finally:
                os.chdir(previous)
            result = yaml.safe_load((target / "service.yaml").read_text())
            for path in paths:
                self.assertEqual(result["paths"][path]["post"]["x-stackQL-graphQL"]["url"],
                                 "https://mock.example:1234" + path + "?tenant=x")


if __name__ == "__main__":
    unittest.main()
