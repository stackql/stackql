"""Provider transaction scenarios sharing the authenticated access fixture."""
import json
import re
import time
import os
import subprocess
import threading
from pathlib import Path


def verify_provider_transaction(scenario, run, token, managed, counts, root, base, config_file):
    config = {'collection': 'sql-test', 'keys': ['item'], 'targets': {
        'stackql_unstable_accessdemo.service.items': {'key': 'item', 'identity': 'id', 'addressed_by': 'id'}}}
    if scenario == 'provider_transaction_eager_rollback':
        config['mode'] = 'eager'
    configure = 'SET "$.transaction" = \'' + json.dumps(config) + "';"
    insert = "INSERT INTO stackql_unstable_accessdemo.service.items(name) SELECT 'transaction-item';"
    read = 'SELECT id FROM stackql_unstable_accessdemo.service.items;'
    operator = token(subject='operator')
    if scenario == 'provider_transaction_default':
        # No $.transaction SET and no caller-auth opt-in.
        code, output = run('BEGIN;' + insert + insert + read + 'ROLLBACK;', protected=False)
        assert code == 0 and 'Rollback OK' in output, output
        assert not managed and counts['mutation'] == 4, (managed, counts)
        code, output = run('BEGIN;' + insert + 'COMMIT;', protected=False)
        assert code == 0 and len(managed) == 1 and counts['mutation'] == 5, output
        records = list((Path(root) / 'iac').rglob('ledger/sql/.transactions/*.json'))
        assert records, 'default SQL transaction did not persist Omni checkpoints'
        code, output = run('ROLLBACK;', protected=False)
        assert code != 0 and counts['mutation'] == 5, f'standalone rollback code={code}: {output}'
        return
    if scenario == 'provider_transaction_auth_disabled':
        for mode in ('eager', 'lazy'):
            config['mode'] = mode
            configure = 'SET "$.transaction" = \' ' + json.dumps(config) + "';"
            code, output = run(configure + 'BEGIN;' + insert + read + 'ROLLBACK;', protected=False)
            assert code == 0 and 'Rollback OK' in output and not managed, output
        assert counts['mutation'] == 4, counts
        return
    if scenario in ('provider_transaction_resume', 'provider_transaction_resume_queue', 'provider_transaction_recovery'):
        config['lease_seconds'] = 3
        config['mode'] = 'lazy' if scenario == 'provider_transaction_resume_queue' else 'eager'
        configure = 'SET "$.transaction" = \' ' + json.dumps(config) + "';"
        code, output = run(configure + 'BEGIN;' + insert, operator)
        if scenario == 'provider_transaction_recovery':
            assert code != 0 and 'inverse token' in output, output
        else:
            assert code == 0, output
        match = re.search(r'transaction begun: (\S+)', output)
        assert match, output
        transaction_id = match.group(1)
        config['resume_id'] = transaction_id
        configure = 'SET "$.transaction" = \' ' + json.dumps(config) + "';"
        # The process exited with an open transaction. Reclaim its actual durable
        # lease by expiry, without rewriting state files or force-unlock tricks.
        time.sleep(4)
        if scenario == 'provider_transaction_resume_queue':
            assert not managed and counts['mutation'] == 0, (managed, counts)
            code, output = run(configure + 'BEGIN;COMMIT;', operator)
            assert code == 0 and len(managed) == 1 and counts['mutation'] == 1, output
        elif scenario == 'provider_transaction_resume':
            assert len(managed) == 1 and counts['mutation'] == 1, (managed, counts)
            code, output = run(configure + 'BEGIN;ROLLBACK;', operator)
            assert code == 0 and not managed and counts['mutation'] == 2, output
        else:
            assert len(managed) == 1, managed
            output = recover_in_shell(base, config_file, operator, configure, next(iter(managed)))
            assert 'Rollback OK' in output and not managed and counts['mutation'] == 2, output

        # A closed durable transaction ID cannot replay its work.
        before = counts['mutation']
        code, output = run(configure + 'BEGIN;', operator)
        assert code != 0 and counts['mutation'] == before, output
        return
    if scenario == 'provider_transaction_lazy_rollback':
        code, output = run(configure + 'BEGIN;' + insert + 'ROLLBACK;', operator)
        assert code == 0 and 'Rollback OK' in output, output
        assert not managed and counts['mutation'] == 0, (managed, counts)
    elif scenario == 'provider_transaction_eager_rollback':
        code, output = run(configure + 'BEGIN;' + insert + 'ROLLBACK;', operator)
        assert code == 0 and 'Rollback OK' in output, output
        assert not managed and counts['mutation'] == 2, (managed, counts)
    elif scenario == 'provider_transaction_returning':
        returning = insert.rstrip(';') + ' RETURNING name;'
        code, output = run(configure + 'BEGIN;' + returning + 'ROLLBACK;', operator)
        assert code == 0 and 'transaction-item' in output and 'Rollback OK' in output, output
        assert not managed and counts['mutation'] == 2, (managed, counts)
    elif scenario == 'provider_transaction_missing_inverse':
        code, output = run(configure + 'BEGIN;' + insert + 'COMMIT;', operator)
        assert code != 0 and 'declared inverse' in output, output
        assert not managed and counts['mutation'] == 0, (managed, counts)
    elif scenario == 'provider_transaction_status':
        code, output = run(configure + 'BEGIN;SELECT * FROM stackql_preview.txn.status;ROLLBACK;', operator)
        assert code == 0 and 'version' in output and 'active' in output, output
        assert not managed and counts['mutation'] == 0, (managed, counts)
    elif scenario == 'provider_transaction_read_barrier':
        code, output = run(configure + 'BEGIN;' + insert + read + 'ROLLBACK;', operator)
        assert code == 0 and 'Rollback OK' in output, output
        assert counts['read_after_mutations'] and counts['read_after_mutations'][0] == 1, counts
        assert not managed and counts['mutation'] == 2, (managed, counts)
    elif scenario == 'provider_transaction_commit':
        code, output = run(configure + 'BEGIN;' + insert + 'COMMIT;', operator)
        assert code == 0 and 'transaction committed' in output, output
        assert len(managed) == 1 and counts['mutation'] == 1, (managed, counts)
        records = list((Path(root) / 'iac').rglob('ledger/sql-test/.transactions/*.json'))
        assert records, 'transaction status was not persisted in the IaC ledger'
        assert list((Path(root) / 'iac').rglob('journal/txn-*.jsonl')), 'mutation was not journalled'
    elif scenario == 'provider_transaction_irreversible':
        code, output = run(configure + "BEGIN; DELETE FROM stackql_unstable_accessdemo.service.items WHERE id = 'existing';", operator)
        assert code != 0 and 'inverse' in output.lower(), output
        assert not managed and counts['mutation'] == 0, (managed, counts)
    elif scenario == 'provider_transaction_denied':
        code, output = run(configure + 'BEGIN;' + insert + 'COMMIT;', token(subject='reader'))
        assert code != 0 and 'denied' in output.lower(), output
        assert counts['mutation'] == 0 and not managed, (managed, counts)
    else:
        raise AssertionError('unknown transaction scenario: ' + scenario)


def status_row(output):
    for match in re.finditer(r'\[\s*\{', output):
        try:
            rows, _ = json.JSONDecoder().raw_decode(output[match.start():])
        except ValueError:
            continue
        if rows and 'version' in rows[0] and 'pending' in rows[0]:
            return rows[0]
    raise AssertionError('missing transaction status: ' + output)


def recover_in_shell(base, config_file, credential, configure, identity):
    env = os.environ.copy()
    env['STACKQL_CALLER_TOKEN'] = credential
    process = subprocess.Popen(base + ['--access.config', json.dumps({'path': str(config_file)}), 'shell'],
                               env=env, stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True, bufsize=1)
    lines = []
    def read_output():
        # JSON output need not end in a newline; inspect it without waiting for EOF.
        while True:
            char = process.stdout.read(1)
            if not char:
                return
            lines.append(char)
    reader = threading.Thread(target=read_output, daemon=True)
    reader.start()
    try:
        process.stdin.write(configure + '\nBEGIN;\nSELECT * FROM stackql_preview.txn.status;\n')
        process.stdin.flush()
        deadline = time.monotonic() + 25
        while True:
            output = ''.join(lines)
            try:
                snapshot = status_row(output)
                break
            except AssertionError:
                assert time.monotonic() < deadline and process.poll() is None, output
                time.sleep(0.02)
        pending = json.loads(snapshot['pending'])
        assert len(pending) == 1, snapshot
        resources = json.dumps([{'key': 'item', 'outcome': 'applied', 'version': pending[0]['version'], 'identity': identity}])
        command = "SELECT * FROM stackql_preview.txn.recover WHERE version = '" + snapshot['version'] + "' AND resources = '" + resources + "' AND continue = 'false';"
        process.stdin.write(command + '\nROLLBACK;\nexit\n')
        process.stdin.flush()
        process.wait(timeout=25)
        reader.join(3)
        output = ''.join(lines)
        assert credential not in output, 'caller credential leaked'
        return output
    finally:
        if process.poll() is None:
            process.kill()
            process.wait(timeout=5)
        process.stdin.close()
        process.stdout.close()
