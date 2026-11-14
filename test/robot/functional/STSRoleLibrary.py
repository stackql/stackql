"""SQL authentication regression checks using the isolated Flask STS fixture."""
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import threading

from werkzeug.serving import make_server

sys.path.insert(0, str(Path(__file__).resolve().parents[2] / 'python/stackql_test_tooling/flask'))
from sts import STSFixture, create_app


class STSRoleLibrary:
    def prepare_sts(self, binary, execution_platform='native'):
        self.root = Path(tempfile.mkdtemp(prefix='stackql-sts-'))
        self.state = STSFixture()
        self.server = make_server('0.0.0.0' if execution_platform == 'docker' else '127.0.0.1', 0, create_app(self.state), threaded=True)
        self.worker = threading.Thread(target=self.server.serve_forever, daemon=True)
        self.worker.start()
        host = 'host.docker.internal' if execution_platform == 'docker' else '127.0.0.1'
        self.endpoint = f'http://{host}:{self.server.server_port}'
        self.prefix = [shutil.which(binary) or str(Path(binary).resolve())]
        if execution_platform == 'docker':
            self.prefix = ['docker', '--log-level', 'error', 'compose', '--progress', 'quiet', '-p', 'execrun', 'run', '--rm', '--no-deps', '-T',
                           '--user', f'{os.getuid()}:{os.getgid()}', '--workdir', str(self.root), '-v', f'{self.root}:{self.root}',
                           '--entrypoint', 'stackql', 'stackqlsrv']
        registry = self.root / 'registry'
        version = registry / 'src/awsfixture/v00.00.00000'
        (version / 'services').mkdir(parents=True)
        provider = {'name': 'awsfixture', 'id': 'awsfixture', 'version': 'v00.00.00000', 'config': {'auth': {'type': 'aws_signing_v4'}},
                    'providerServices': {'iam': {'id': 'iam:v00.00.00000', 'name': 'iam', 'preferred': True, 'service': {'$ref': 'awsfixture/v00.00.00000/services/iam.yaml'}}}}
        doc = {'openapi': '3.0.0', 'info': {'title': 'IAM fixture', 'version': '1', 'x-serviceName': 'iam'},
               'servers': [{'url': 'https://iam.{region}.example', 'variables': {'region': {'default': 'ap-southeast-2'}}}],
               'paths': {'/users': {'get': {'operationId': 'ListUsers', 'parameters': [{'name': 'previous', 'in': 'query', 'schema': {'type': 'string'}}],
                   'responses': {'200': {'content': {'application/json': {'schema': {'type': 'array', 'items': {'type': 'object', 'properties': {'id': {'type': 'string'}}}}}}}}}}},
               'components': {'x-stackQL-resources': {'users': {'id': 'awsfixture.iam.users', 'name': 'users', 'methods': {'list': {'operation': {'$ref': '#/paths/~1users/get'}, 'response': {'mediaType': 'application/json', 'openAPIDocKey': '200'}}},
                   'sqlVerbs': {'select': [{'$ref': '#/components/x-stackQL-resources/users/methods/list'}]}}}}}
        (version / 'provider.yaml').write_text(json.dumps(provider))
        (version / 'services/iam.yaml').write_text(json.dumps(doc))
        self.flags = ['--approot', str(self.root / 'app'), '--registry', json.dumps({'localDocRoot': str(registry), 'verifyConfig': {'nopVerify': True}}),
                      '--preview', json.dumps({'unstable': True, 'endpoint': self.endpoint}), '--output', 'json', '--sqlBackend', json.dumps({'dsn': str(self.root / 'session.db')})]
        self.env = os.environ.copy()
        self.env.update({'AWS_ACCESS_KEY_ID': self.state.base[0], 'AWS_SECRET_ACCESS_KEY': self.state.base[1], 'AWS_SESSION_TOKEN': self.state.base[2],
                         'STS_TEST_ROLE': self.state.role, 'STS_TEST_EXTERNAL': self.state.external})
        if execution_platform == 'docker':
            position = self.prefix.index('--entrypoint')
            names = ['AWS_ACCESS_KEY_ID', 'AWS_SECRET_ACCESS_KEY', 'AWS_SESSION_TOKEN', 'STS_TEST_ROLE', 'STS_TEST_EXTERNAL']
            self.prefix[position:position] = [part for name in names for part in ('-e', name + '=' + self.env[name])]
        self.auth = {'type': 'aws_assume_role', 'keyID': self.state.base[0], 'credentialsenvvar': 'AWS_SECRET_ACCESS_KEY',
                     'aws_role_arn': self.state.role, 'aws_role_external_id': self.state.external, 'aws_role_session_name': self.state.session,
                     'aws_role_duration_seconds': 900, 'aws_sts_region': self.state.region, 'aws_sts_endpoint': self.endpoint + '/sts'}

    def run_sql(self, sql, auth=None, success=True):
        result = subprocess.run(self.prefix + ['exec', sql] + self.flags + ['--auth', json.dumps({'awsfixture': self.auth if auth is None else auth})],
                                env=self.env, capture_output=True, text=True, timeout=40)
        assert (result.returncode == 0 and not result.stderr.strip()) == success, result.stdout + result.stderr
        for secret in [self.state.base[1], self.state.base[2], 'ROLESECRET', 'ROLETOKEN', self.state.external]:
            assert secret not in result.stdout + result.stderr, 'credentials leaked in SQL output'
        if success:
            assert 'role-user' in result.stdout, result.stdout
        return result

    def query(self):
        return "SELECT id FROM awsfixture.iam.users WHERE region = 'ap-southeast-2';"

    def role_auth_fields_reach_sts_and_sign_downstream(self):
        self.run_sql(self.query())
        assert [c['action'] for c in self.state.requests] == ['AssumeRole', 'ListUsers'], self.state.requests
        assert self.state.requests[-1]['key'] == 'ROLEKEY1'

    def repeated_sql_reuses_unexpired_role_credentials(self):
        self.run_sql(self.query() + self.query())
        assert [c['action'] for c in self.state.requests] == ['AssumeRole', 'ListUsers', 'ListUsers'], self.state.requests

    def union_branches_share_role_acquisition(self):
        branch = self.query().rstrip(';')
        result = self.run_sql(' UNION ALL '.join([branch] * 8) + ';')
        assert len(json.loads(result.stdout)) == 8, result.stdout
        assert sum(c['action'] == 'AssumeRole' for c in self.state.requests) == 1, self.state.requests
        assert len([c for c in self.state.requests if c['action'] == 'ListUsers']) >= 1

    def canceled_sql_stops_before_downstream_dispatch(self):
        self.state.block.clear()
        process = subprocess.Popen(self.prefix + ['exec', self.query()] + self.flags + ['--auth', json.dumps({'awsfixture': self.auth}), '--apirequesttimeout', '1'],
                                   env=self.env, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
        try:
            assert self.state.started.wait(10), 'SQL did not reach STS'
            stdout, stderr = process.communicate(timeout=25)
            assert 'deadline exceeded' in stderr or 'canceled' in stderr, stdout + stderr
            assert not any(c['action'] == 'ListUsers' for c in self.state.requests), self.state.requests
        finally:
            if process.poll() is None:
                process.kill()
                process.communicate(timeout=5)
            self.state.block.set()

    def role_auth_environment_indirection(self):
        auth = dict(self.auth)
        del auth['aws_role_arn'], auth['aws_role_external_id']
        auth.update(aws_role_arn_env_var='STS_TEST_ROLE', aws_role_external_id_env_var='STS_TEST_EXTERNAL')
        self.run_sql(self.query(), auth)
        assert self.state.requests[-1]['key'] == 'ROLEKEY1'

    def role_expiry_refreshes_between_sql_statements(self):
        self.state.lifetime = 2
        self.state.downstream_delay = 2.2
        self.run_sql(self.query() + self.query())
        assert [c['key'] for c in self.state.requests if c['action'] == 'ListUsers'] == ['ROLEKEY1', 'ROLEKEY2'], self.state.requests

    def rejected_sts_never_dispatches_provider_query(self):
        self.state.failure = True
        result = self.run_sql(self.query(), success=False)
        assert 'STS AssumeRole rejected: status 403, code AccessDenied' in result.stderr, result.stderr
        assert self.state.requests and all(c['action'] == 'AssumeRole' for c in self.state.requests), self.state.requests

    def expired_sts_never_dispatches_provider_query(self):
        self.state.expired = True
        result = self.run_sql(self.query(), success=False)
        assert 'expired credentials' in result.stderr, result.stderr
        assert self.state.requests and all(c['action'] == 'AssumeRole' for c in self.state.requests), self.state.requests

    def static_session_credentials_do_not_call_sts(self):
        self.state.issued[self.state.base[0]] = self.state.base
        self.run_sql(self.query(), {'type': 'aws_signing_v4', 'keyID': self.state.base[0], 'credentialsenvvar': 'AWS_SECRET_ACCESS_KEY'})
        assert [c['action'] for c in self.state.requests] == ['ListUsers'], self.state.requests
        assert self.state.requests[0]['key'] == self.state.base[0]

    def sql_auth_rotation_replaces_role_source(self):
        self.state.issued[self.state.base[0]] = self.state.base
        static = {'type': 'aws_signing_v4', 'keyID': self.state.base[0], 'credentialsenvvar': 'AWS_SECRET_ACCESS_KEY'}
        rotated = dict(self.auth, aws_role_session_name='fixture-rotated')
        self.state.sessions.add('fixture-rotated')
        setting = lambda auth: 'SET "$.auth.awsfixture" = \' ' + json.dumps(auth) + "';"
        self.run_sql(self.query() + setting(static) + self.query() + setting(rotated) + self.query())
        keys = [c['key'] for c in self.state.requests if c['action'] == 'ListUsers']
        assert keys[0].startswith('ROLEKEY') and keys[1] == self.state.base[0] and keys[2].startswith('ROLEKEY') and keys[0] != keys[2], self.state.requests

    def unset_role_configuration_fails_before_sts(self):
        sql = 'SET "$.auth.awsfixture" = \'' + json.dumps(self.auth) + "';"
        sql += 'UNSET "$.auth.awsfixture.aws_role_arn";' + self.query()
        result = self.run_sql(sql, success=False)
        assert 'AWS role ARN not provided' in result.stderr, result.stderr
        assert not self.state.requests, self.state.requests

    def stop_sts(self):
        self.state.block.set()
        self.server.shutdown()
        self.worker.join(5)
        shutil.rmtree(self.root)
