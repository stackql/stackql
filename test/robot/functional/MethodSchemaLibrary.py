"""Method discovery through the real CLI and MCP stdio; no provider requests."""
import json
import subprocess
import threading

from ProviderSessionLibrary import ProviderSessionLibrary


class MethodSchemaLibrary(ProviderSessionLibrary):
    def prepare_method_schemas(self, binary, execution_platform="native", backend="{}"):
        self.prepare_session_provider(binary, execution_platform, backend)
        path = self.root / "registry/src/widgets/v00.00.00000/services/things.yaml"
        doc = json.loads(path.read_text())
        node = {"type": "object", "required": ["mode"], "properties": {
            "mode": {"type": "string", "description": "Machine mode", "enum": ["small", "large"]},
            "children": {"type": "array", "items": {"$ref": "#/components/schemas/Node"}}}}
        lifecycle = {"type": "object", "properties": {"Rules": {"type": "array", "xml": {"name": "RuleList"}, "items": {
            "type": "object", "properties": {"Status": {"type": "string", "description": "Rule status", "enum": ["Enabled", "Disabled"]}}}}}}
        doc["components"]["schemas"] = {"Node": node, "Lifecycle": lifecycle,
            "Display": {"type": "object", "properties": {"line_items": {"type": "array", "items": {"$ref": "#/components/schemas/Lifecycle"}}}}}
        doc["paths"]["/widgets"]["post"] = {"operationId": "create", "requestBody": {"required": True, "content": {"application/json": {"schema": {
            "type": "object", "required": ["flag", "settings"], "properties": {"flag": {"type": "boolean"}, "settings": {"$ref": "#/components/schemas/Node"}}}}}},
            "responses": {"200": {"content": {"application/json": {"schema": {"$ref": "#/components/schemas/Node"}}}}}}
        doc["paths"]["/xml"] = {
            "put": {"operationId": "putXml", "requestBody": {"content": {"application/xml": {"schema": {"type": "object", "properties": {
                "LifecycleConfiguration": {"$ref": "#/components/schemas/Lifecycle"}}}}}}, "responses": {"204": {"description": "accepted"}}},
            "get": {"operationId": "getXml", "responses": {"200": {"content": {"application/xml": {"schema": {"$ref": "#/components/schemas/Lifecycle"}}}}}}}
        for operations in doc["paths"].values():
            for operation in operations.values():
                operation["parameters"] = [{"name": "region", "in": "query", "required": True, "schema": {"type": "string"}},
                    {"name": "alt", "in": "query", "schema": {"type": "string"}}]
        methods = doc["components"]["x-stackQL-resources"]["widgets"]["methods"]
        methods.update({"create": {"operation": {"$ref": "#/paths/~1widgets/post"}, "response": {"mediaType": "application/json"}},
            "put_xml": {"operation": {"$ref": "#/paths/~1xml/put"}, "response": {"openAPIDocKey": "204"}},
            "get_xml": {"operation": {"$ref": "#/paths/~1xml/get"}, "response": {"mediaType": "application/xml", "overrideMediaType": "application/json",
                "schema_override": {"$ref": "#/components/schemas/Display"}, "objectKey": "$.line_items", "transform": {"type": "schema_driven_xml_v0.1.0"}}}})
        path.write_text(json.dumps(doc))

    def schema_sql(self, sql, success=True):
        result = subprocess.run(self.prefix + ["exec", sql] + self.flags, env=self.env, capture_output=True, text=True, timeout=30)
        if success:
            assert result.returncode == 0 and not result.stderr.strip(), result.stdout + result.stderr
        else:
            assert result.stderr.strip(), "invalid discovery controls were accepted"
        assert not self.calls, "metadata discovery called the provider"
        return json.loads(result.stdout) if success else result.stderr

    @staticmethod
    def named(rows, name):
        return next(row for row in rows if row["name"] == name)

    def json_schema_default_is_typed_and_shallow(self):
        rows = self.schema_sql("DESCRIBE METHOD widgets.things.widgets.create;")
        assert self.named(rows, "data__flag")["type"] == "boolean", rows
        settings = self.named(rows, "data__settings")
        shape = json.loads(settings["shape"])
        assert settings["type"] == "object" and shape["child_count"] == 2 and shape["truncated"], shape
        assert "properties" not in shape and not any(row["name"] == "alt" for row in rows), rows
        assert len(json.dumps(rows)) < 4096, "default view unexpectedly expanded"
        assert self.named(rows, "region")["param_type"] == "input_required", rows

    def schema_drill_down_preserves_descriptions_and_enums(self):
        rows = self.schema_sql("SELECT * FROM stackql_preview.metadata.method_fields WHERE provider = 'widgets' AND service = 'things' AND resource = 'widgets' AND method = 'create' AND path = 'data__settings.children.mode' AND descriptions = 'true' AND param_type = 'input';")
        assert len(rows) == 1 and rows[0]["description"] == "Machine mode", rows
        assert json.loads(rows[0]["shape"])["enum"] == ["small", "large"], rows
        rows = self.schema_sql("SELECT * FROM stackql_preview.metadata.method_fields WHERE provider = 'widgets' AND service = 'things' AND resource = 'widgets' AND method = 'create' AND include_global = 'true' AND param_type = 'input_optional';")
        assert [row["name"] for row in rows] == ["alt"], rows

    def xml_schema_reports_nested_request_and_selected_row(self):
        rows = self.schema_sql("SELECT * FROM stackql_preview.metadata.method_fields WHERE provider = 'widgets' AND service = 'things' AND resource = 'widgets' AND method = 'put_xml' AND path = 'data__LifecycleConfiguration.Rules' AND depth = 3 AND descriptions = 'true';")
        shape = json.loads(rows[0]["shape"])
        assert shape["type"] == "array" and shape["wire_name"] == "RuleList", shape
        assert shape["items"]["properties"]["Status"]["enum"] == ["Enabled", "Disabled"], shape
        rows = self.schema_sql("DESCRIBE METHOD widgets.things.widgets.get_xml;")
        assert self.named(rows, "Rules")["param_type"] == "output", rows
        assert not any(row["name"] == "line_items" for row in rows), rows

    def invalid_schema_controls_are_rejected(self):
        base = "SELECT * FROM stackql_preview.metadata.method_fields WHERE provider = 'widgets' AND service = 'things' AND resource = 'widgets' AND method = 'create'"
        for suffix, expected in [(" AND depth = 33", "depth"), (" AND path = 'data__settings.missing'", "does not exist"), (" AND param_type = 'bogus'", "param_type")]:
            assert expected in self.schema_sql(base + suffix + ";", success=False)

    def mcp_method_schema_controls_reach_backend(self):
        flags = list(self.flags)
        flags[flags.index("--preview") + 1] = json.dumps({"unstable": False})
        command = self.prefix + flags + ["--mcp.config", json.dumps({"server": {"transport": "stdio", "audit": {"disabled": True}}, "enabled_tools": ["describe_method"]}), "mcp"]
        process = subprocess.Popen(command, env=self.env, stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        watchdog = threading.Timer(45, process.kill)
        watchdog.start()
        try:
            def request(identifier, method, params):
                process.stdin.write((json.dumps({"jsonrpc": "2.0", "id": identifier, "method": method, "params": params}) + "\n").encode())
                process.stdin.flush()
                while True:
                    line = process.stdout.readline()
                    assert line, "MCP closed before response"
                    message = json.loads(line)
                    if message.get("id") == identifier:
                        return message
            request(1, "initialize", {"protocolVersion": "2025-06-18", "capabilities": {}, "clientInfo": {"name": "schema-test", "version": "1"}})
            process.stdin.write(b'{"jsonrpc":"2.0","method":"notifications/initialized"}\n')
            args = {"provider": "widgets", "service": "things", "resource": "widgets", "method": "create", "format": "json",
                "path": "data__settings.children.mode", "depth": 1, "descriptions": True, "param_type": "input"}
            response = request(2, "tools/call", {"name": "describe_method", "arguments": args})
            assert "error" not in response and not response["result"].get("isError"), response
            content = response["result"].get("structuredContent")
            if content is None:
                content = json.loads(response["result"]["content"][0]["text"])
            rows = content["rows"]
            assert len(rows) == 1 and rows[0]["description"] == "Machine mode", rows
            assert json.loads(rows[0]["shape"])["enum"] == ["small", "large"], rows
            args["depth"] = 33
            failure = request(3, "tools/call", {"name": "describe_method", "arguments": args})
            assert "error" in failure or failure.get("result", {}).get("isError"), failure
            assert not self.calls, "MCP discovery called the provider"
            process.stdin.close()
            process.stdin = None
            process.communicate(timeout=10)
            assert process.returncode == 0
        finally:
            watchdog.cancel()
            if process.poll() is None:
                process.kill()
                process.wait()
