import contextlib
import importlib.util
import io
import json
import os
import shutil
from pathlib import Path
import stat
import subprocess
import tempfile
import unittest
from unittest import mock

HERE = Path(__file__).resolve().parent
ROOT = HERE.parent.parent

def load(name):
    spec = importlib.util.spec_from_file_location(name, HERE / (name + ".py"))
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module

remote = load("remote")
transport = load("transport")
TOKEN = "ntn_SECRET_SENTINEL_0123456789"
PASSWORD = "DB_SECRET_SENTINEL_$quotes' spaces"

class RuntimeEnvTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.directory = Path(self.temp.name)
        self.credential = self.directory / "credential"
        self.candidate = self.directory / "candidate"
        self.write(self.credential, "")
        self.write(self.candidate, "MYSQL_PASSWORD=unchanged\nBACKEND_IMAGE_TAG=new\n")
    def write(self, path, value):
        path.write_text(value)
        path.chmod(0o600)
    def apply(self):
        out = io.StringIO()
        with contextlib.redirect_stdout(out):
            remote.runtime_env(self.directory, self.candidate, self.credential)
        self.assertNotIn(TOKEN, out.getvalue())
        return (self.directory / ".env").read_text()
    def test_missing_secret_preserves_existing_token_and_switches(self):
        self.write(self.directory / ".env", "MINIBLOG_NOTION_TOKEN=old_secret_value\nMINIBLOG_NOTION_SYNC_ENABLED=false\nMINIBLOG_NOTION_SYNC_AUTHOR='nickname'\nMINIBLOG_CONTENT_REGISTER_ENABLED=true\n")
        result = self.apply()
        self.assertIn("MINIBLOG_NOTION_TOKEN=old_secret_value\n", result)
        self.assertIn("MINIBLOG_NOTION_SYNC_ENABLED=false\n", result)
        self.assertIn("MINIBLOG_CONTENT_REGISTER_ENABLED=true\n", result)
        self.assertIn("MINIBLOG_NOTION_SYNC_AUTHOR='nickname'\n", result)
        self.assertIn("BACKEND_IMAGE_TAG=new\n", result)
        for name in (".env", ".env.previous"):
            self.assertEqual(stat.S_IMODE((self.directory / name).stat().st_mode), 0o600)
    def test_new_secret_replaces_duplicates_and_removes_writer_from_backup(self):
        self.write(self.credential, TOKEN)
        self.write(self.directory / ".env", "MINIBLOG_NOTION_TOKEN=old\nMINIBLOG_NOTION_TOKEN=duplicate\nMINIBLOG_NOTION_BOOTSTRAP_TOKEN=writer_secret\nMINIBLOG_NOTION_SYNC_ENABLED=false\n")
        result = self.apply()
        self.assertEqual(result.count("MINIBLOG_NOTION_TOKEN="), 1)
        self.assertIn("MINIBLOG_NOTION_TOKEN=" + TOKEN, result)
        self.assertNotIn("BOOTSTRAP", result)
        self.assertNotIn("BOOTSTRAP", (self.directory / ".env.previous").read_text())
        self.assertIn("SYNC_ENABLED=false", result)
    def test_unconfigured_initial_token_does_not_enable_sync(self):
        result = self.apply()
        self.assertNotIn("MINIBLOG_NOTION_TOKEN", result)
        self.assertNotIn("SYNC_ENABLED=true", result)
    def test_invalid_credential_and_symlink_do_not_change_current_env(self):
        target = self.directory / ".env"
        self.write(target, "unchanged")
        for value in (TOKEN + "\n", TOKEN + "$", TOKEN + "'", TOKEN + "\rNEW=true"):
            self.write(self.credential, value)
            with self.assertRaises(remote.SafeError):
                self.apply()
            self.assertEqual(target.read_text(), "unchanged")
        self.write(self.credential, TOKEN)
        target.unlink()
        target.symlink_to(self.candidate)
        with self.assertRaises(remote.SafeError):
            self.apply()
        self.assertIn("BACKEND_IMAGE_TAG=new", self.candidate.read_text())
    def test_replace_is_atomic_and_original_retained_on_failure(self):
        target = self.directory / ".env"
        self.write(target, "old")
        self.write(self.credential, TOKEN)
        actual = os.replace
        def fail_target(source, destination):
            if Path(destination) == target:
                raise OSError(PASSWORD)
            actual(source, destination)
        with mock.patch.object(remote.os, "replace", side_effect=fail_target):
            with self.assertRaises(OSError):
                self.apply()
        self.assertEqual(target.read_text(), "old")
        self.assertEqual(list(self.directory.glob('.notion-ops-*')), [])

FAKE_DOCKER = r'''#!/usr/bin/env python3
import json, os, pathlib, sys
args=sys.argv[1:]
record=pathlib.Path(os.environ['FAKE_CALLS'])
with record.open('a') as output: output.write(json.dumps(args)+'\n')
if args[:2]==['container','inspect']: raise SystemExit(1)
if args[0]=='inspect':
    if '.Config.Env' in args[2]:
        print(os.environ['FAKE_ENV'])
    else:
        print(json.dumps({'image':'sha256:'+'a'*64,'running':True}))
    raise SystemExit(0)
if args[0]=='rm': raise SystemExit(0)
binary=args[args.index('--entrypoint')+1]
mount=args[args.index('--volume')+1].split(':')[0]
envpath=pathlib.Path(args[args.index('--env-file')+1])
with pathlib.Path(os.environ['FAKE_ENVS']).open('a') as output:
    output.write(json.dumps({'binary':binary,'env':envpath.read_text()})+'\n')
if binary.endswith('content-preflight'): raise SystemExit(0)
mode=args[args.index('--mode')+1]
if mode==os.environ.get('FAKE_FAIL'):
    print(os.environ['SECRET_SENTINEL']+' '+os.environ['PASSWORD_SENTINEL'], file=sys.stderr)
    raise SystemExit(1)
sources=json.loads(os.environ['FAKE_SOURCES'])
if mode=='schema_check':
    value={'complete':True,'sources':[{'source_id':v['source_id'],'actual_data_source_id':v['source_id'],'status':'valid'} for v in sources]}
elif mode=='status':
    value={'paused':os.environ.get('FAKE_PAUSED','true')=='true','source_writes_paused':False,'baseline_frozen':os.environ.get('FAKE_BASELINE','true')=='true','sources':sources}
elif mode=='dry_run':
    value={'run':{'status':'completed','counts':{'seen':47,'failed':0,'blocked':0}},'items':[]}
else: value={'run_id':'test-run','items':[{'title':os.environ['SECRET_SENTINEL'],'public_url_available':False}]}
report=args[args.index('--report')+1].replace('/ops',mount)
p=pathlib.Path(report); p.write_text(json.dumps(value)); p.chmod(0o600)
'''

class ReadTaskTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.directory = Path(self.temp.name)
        self.app = self.directory / "app"
        self.app.mkdir()
        self.credential = self.directory / 'credential'
        self.credential.write_text(TOKEN); self.credential.chmod(0o600)
        fake = self.directory / 'docker'
        fake.write_text(FAKE_DOCKER); fake.chmod(0o700)
        self.calls = self.directory / 'calls'
        self.envs = self.directory / 'envs'
        self.sources = [{'source_id':i, 'module_code':'configured', 'enabled':False} for i in sorted(remote.SOURCE_IDS)]
        database = ['MYSQL_HOST=host','MYSQL_PORT=3306','MYSQL_DATABASE=blog','MYSQL_USERNAME=user','MYSQL_PASSWORD='+PASSWORD,
                    'JWT_SECRET=DO_NOT_COPY','MINIBLOG_NOTION_BOOTSTRAP_TOKEN=DO_NOT_COPY','MINIBLOG_NOTION_SYNC_ENABLED=false']
        patch = mock.patch.dict(os.environ, {'PATH':str(self.directory)+os.pathsep+os.environ['PATH'],
            'FAKE_CALLS':str(self.calls),'FAKE_ENVS':str(self.envs),'FAKE_SOURCES':json.dumps(self.sources),
            'FAKE_ENV':json.dumps(database),'SECRET_SENTINEL':TOKEN,'PASSWORD_SENTINEL':PASSWORD})
        patch.start(); self.addCleanup(patch.stop)
    def run_task(self, mode='schema_check'):
        out = io.StringIO()
        with contextlib.redirect_stdout(out):
            remote.read_only(self.app, self.credential, mode, '123-1')
        self.assertNotIn(TOKEN, out.getvalue())
        self.assertNotIn(PASSWORD, out.getvalue())
        return self.app / 'ops/notion/123-1'
    def env_records(self):
        return [json.loads(line) for line in self.envs.read_text().splitlines()]
    def test_schema_has_no_database_or_runtime_secret_environment(self):
        directory = self.run_task()
        calls = [json.loads(line) for line in self.calls.read_text().splitlines()]
        self.assertFalse(any('.Config.Env' in ' '.join(call) for call in calls))
        self.assertEqual(len(self.env_records()), 1)
        self.assertEqual(self.env_records()[0]['env'], 'MINIBLOG_NOTION_TOKEN='+TOKEN+'\n')
        self.assertFalse((directory/'task.env').exists())
        self.assertEqual(stat.S_IMODE(directory.stat().st_mode), 0o700)
        for name in ('schema.json','schema.log','operation.json'):
            self.assertEqual(stat.S_IMODE((directory/name).stat().st_mode), 0o600)
        self.assertIn('sha256:', (directory/'operation.json').read_text())
    def test_scans_use_only_database_and_read_credential_without_dualpause(self):
        directory = self.run_task('dry_run')
        for entry in self.env_records():
            self.assertNotIn('JWT', entry['env'])
            self.assertNotIn('BOOTSTRAP', entry['env'])
        self.assertIn('MYSQL_PASSWORD='+PASSWORD, self.env_records()[-1]['env'])
        self.assertTrue((directory/'status-after.json').exists())
        calls = self.calls.read_text()
        self.assertNotIn(PASSWORD, calls)
        self.assertNotIn(TOKEN, calls)
        self.assertNotIn('--enable-sync', calls)
        self.assertIn('--log-driver',calls)
    def test_preview_public_url_missing_does_not_block_candidates(self):
        directory = self.run_task('bootstrap_preview')
        self.assertFalse(json.loads((directory/'result.json').read_text())['items'][0]['public_url_available'])
    def test_failures_keep_private_evidence_and_remove_credentials(self):
        with mock.patch.dict(os.environ, {'FAKE_FAIL':'dry_run'}):
            with self.assertRaises(remote.SafeError) as failure:
                self.run_task('dry_run')
        self.assertNotIn(TOKEN, str(failure.exception))
        directory = self.app/'ops/notion/123-1'
        self.assertFalse((directory/'task.env').exists())
        self.assertNotIn(TOKEN, (directory/'result.log').read_text())
        self.assertNotIn(PASSWORD, (directory/'result.log').read_text())
        self.assertIn('[REDACTED]', (directory/'result.log').read_text())
        self.assertEqual(stat.S_IMODE((directory/'result.log').stat().st_mode), 0o600)
    def test_baseline_and_paused_gates_prevent_page_scans(self):
        for mode, change in [('bootstrap_preview',{'FAKE_BASELINE':'false'}),('dry_run',{'FAKE_PAUSED':'false'})]:
            with self.subTest(mode=mode), mock.patch.dict(os.environ, change):
                with self.assertRaises(remote.SafeError): self.run_task(mode)
            self.assertNotIn('result.json', self.calls.read_text())
            import shutil
            shutil.rmtree(self.app/'ops')
            self.calls.unlink(); self.envs.unlink()
    def test_conflicting_database_aliases_fail_before_database_cli(self):
        aliases = ['MYSQL_HOST=wrong','MINIBLOG_DATABASE_HOST=actual','MYSQL_PORT=3306',
                   'MYSQL_DATABASE=blog','MYSQL_USERNAME=user','MYSQL_PASSWORD='+PASSWORD,
                   'MINIBLOG_NOTION_SYNC_ENABLED=false']
        with mock.patch.dict(os.environ, {'FAKE_ENV':json.dumps(aliases)}):
            with self.assertRaises(remote.SafeError) as error: self.run_task('dry_run')
        self.assertNotIn(PASSWORD,str(error.exception))
        self.assertEqual(len(self.env_records()),1)
    def test_runtime_scheduler_on_blocks_before_database_cli(self):
        aliases = ['MYSQL_HOST=host','MYSQL_PORT=3306','MYSQL_DATABASE=blog',
                   'MYSQL_USERNAME=user','MYSQL_PASSWORD='+PASSWORD,'MINIBLOG_NOTION_SYNC_ENABLED=true']
        with mock.patch.dict(os.environ, {'FAKE_ENV':json.dumps(aliases)}):
            with self.assertRaises(remote.SafeError): self.run_task('dry_run')
        self.assertEqual(len(self.env_records()),1)
    def test_missing_or_empty_runtime_switch_is_not_proof_of_off(self):
        aliases = ['MYSQL_HOST=host','MYSQL_PORT=3306','MYSQL_DATABASE=blog',
                   'MYSQL_USERNAME=user','MYSQL_PASSWORD='+PASSWORD]
        for extra in ([],['MINIBLOG_NOTION_SYNC_ENABLED=']):
            with self.subTest(extra=extra), mock.patch.dict(os.environ,{'FAKE_ENV':json.dumps(aliases+extra)}):
                with self.assertRaises(remote.SafeError): remote.database_environment()
    def test_initial_baseline_requires_all_disabled_and_runtime_scheduler_off(self):
        self.sources[0]['enabled'] = True
        with mock.patch.dict(os.environ, {'FAKE_BASELINE':'false','FAKE_SOURCES':json.dumps(self.sources)}):
            with self.assertRaises(remote.SafeError): self.run_task('dry_run')
        self.assertNotIn('result.json', self.calls.read_text())
    def test_illegal_mode_and_run_path_are_rejected_before_inspection(self):
        for mode, identifier in [('bootstrap_apply','123-1'),('schema_check','../x')]:
            with self.assertRaises(remote.SafeError):
                remote.read_only(self.app,self.credential,mode,identifier)
        self.assertFalse(self.calls.exists())
    def test_report_symlink_is_rejected(self):
        (self.app/'ops').symlink_to(self.directory)
        with self.assertRaises(remote.SafeError): self.run_task()
        self.assertFalse(self.calls.exists())

class ContainerCleanupTests(unittest.TestCase):
    def test_timeout_removes_only_this_invocations_label(self):
        for owner_matches in (True, False):
            with self.subTest(owner_matches=owner_matches), tempfile.TemporaryDirectory() as directory:
                path = Path(directory)/'123-1'; path.mkdir()
                env_file=path/'task.env'
                env_file.write_text('MINIBLOG_NOTION_TOKEN='+TOKEN+'\nMYSQL_PASSWORD='+PASSWORD+'\n'); env_file.chmod(0o600)
                calls=[]
                def run(command, **kwargs):
                    calls.append(command)
                    if command[1:3] == ['container','inspect']:
                        if '--format' not in command:
                            return subprocess.CompletedProcess(command,1,stdout=b'')
                        return subprocess.CompletedProcess(command,0,stdout=b'owner' if owner_matches else b'someone-else')
                    if command[1] == 'run':
                        raise subprocess.TimeoutExpired(command,720,output=(TOKEN+' '+PASSWORD).encode())
                    return subprocess.CompletedProcess(command,0,stdout=b'')
                with mock.patch.object(remote.secrets,'token_hex',return_value='owner'), \
                     mock.patch.object(remote.subprocess,'run',side_effect=run):
                    with self.assertRaises(remote.SafeError) as error:
                        remote.docker_task('sha256:'+'a'*64,path,path/'task.env','/app/notion-sync',[],'result')
                self.assertNotIn(TOKEN,str(error.exception))
                log=(path/'result.log').read_text()
                self.assertNotIn(TOKEN,log); self.assertNotIn(PASSWORD,log)
                self.assertIn('[REDACTED]',log)
                self.assertEqual(any(c[1:3]==['rm','-f'] for c in calls),owner_matches)
    def test_existing_name_is_never_started_or_deleted(self):
        with tempfile.TemporaryDirectory() as directory, mock.patch.object(remote.subprocess,'run') as run:
            run.return_value = subprocess.CompletedProcess([],0)
            with self.assertRaises(remote.SafeError):
                remote.docker_task('image',Path(directory),Path(directory)/'env','/app/notion-sync',[],'result')
            self.assertEqual(run.call_count,1)

@unittest.skipUnless(shutil.which('ssh-keygen'), 'OpenSSH key parser is required')
class SSHKeyTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.temp=tempfile.TemporaryDirectory(prefix='miniblog-throwaway-key-')
        cls.directory=Path(cls.temp.name)
        cls.directory.chmod(0o700)
        key=cls.directory/'fixture'
        result=subprocess.run(['ssh-keygen','-q','-t','ed25519','-N','','-f',str(key)],
                              stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,check=False)
        if result.returncode: raise RuntimeError('throwaway SSH key generation failed')
        cls.key=key.read_text()
    @classmethod
    def tearDownClass(cls): cls.temp.cleanup()
    def parse(self,path):
        return subprocess.run(['ssh-keygen','-y','-P','','-f',str(path)],stdin=subprocess.DEVNULL,
                              stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,check=False).returncode
    def test_unterminated_and_crlf_keys_fail_before_normalization(self):
        with tempfile.TemporaryDirectory() as directory:
            path=Path(directory)/'key'
            for value in (self.key.rstrip('\n'),self.key.replace('\n','\r\n')):
                transport.private_write(path,value)
                self.assertNotEqual(self.parse(path),0)
                path.unlink()
    def test_normalized_keys_parse_without_key_contents_in_output(self):
        with tempfile.TemporaryDirectory() as directory:
            path=Path(directory)/'key'
            for value in (self.key.rstrip('\n'),self.key.replace('\n','\r\n'),self.key+'\n'):
                output=io.StringIO()
                with contextlib.redirect_stdout(output),contextlib.redirect_stderr(output):
                    transport.write_validated_key(path,value)
                self.assertEqual(self.parse(path),0)
                self.assertEqual(output.getvalue(),'')
                self.assertTrue(path.read_bytes().endswith(b'\n'))
                self.assertFalse(b'\r' in path.read_bytes())
                self.assertEqual(stat.S_IMODE(path.stat().st_mode),0o600)
                path.unlink()
    def test_invalid_key_is_rejected_before_network_with_constant_message(self):
        env={'GITHUB_REF':'refs/heads/main','GITHUB_RUN_ID':'123','GITHUB_RUN_ATTEMPT':'1',
             'SVRD_HOST':'host.private.test','SVRD_USER':'deploy','SVRD_PORT':'22','SVRD_SSH_KEY':TOKEN,
             'SVRD_HOST_FINGERPRINT':'','MINIBLOG_NOTION_TOKEN':TOKEN}
        output=io.StringIO()
        actual=subprocess.run
        calls=[]
        def run(command,**kwargs):
            calls.append(command)
            self.assertEqual(command[0],'ssh-keygen')
            return actual(command,**kwargs)
        with mock.patch.dict(os.environ,env),mock.patch.object(transport.subprocess,'run',side_effect=run), \
             mock.patch.object(transport.sys,'argv',['transport.py','--mode','schema_check']), \
             contextlib.redirect_stdout(output),contextlib.redirect_stderr(output):
            self.assertEqual(transport.main(),1)
        self.assertEqual(len(calls),1)
        self.assertIn('SSH private key format is invalid',output.getvalue())
        self.assertNotIn(TOKEN,output.getvalue())
        self.assertNotIn('host.private.test',output.getvalue())
    def test_encrypted_key_has_fixed_noninteractive_failure(self):
        with tempfile.TemporaryDirectory() as directory:
            path=Path(directory)/'key'
            result=subprocess.CompletedProcess([],255,stderr=('incorrect passphrase '+TOKEN).encode())
            with mock.patch.object(transport.subprocess,'run',return_value=result):
                with self.assertRaises(transport.SafeError) as error: transport.write_validated_key(path,TOKEN)
            self.assertEqual(str(error.exception),'SSH private key requires an unsupported passphrase')

class SSHErrorTests(unittest.TestCase):
    def test_error_categories_do_not_echo_host_path_or_credentials(self):
        cases=[('Load key',255,'SSH private key could not be loaded'),
               ('Permission denied (publickey). Connection closed',255,'SSH authentication was rejected'),
               ('Host key verification failed',255,'SSH host verification failed'),
               ('Could not resolve hostname',255,'SSH destination could not be resolved'),
               ('Connection refused',255,'SSH connection failed'),
               ('mkdir: permission denied',1,'remote command failed'),
               ('unrecognized failure',255,'remote command failed')]
        for raw,code,expected in cases:
            with self.subTest(category=expected):
                value=transport.ssh_error_category((raw+' '+TOKEN+' '+PASSWORD+' host.private.test').encode(),code,'remote command failed')
                self.assertEqual(value,expected)
                self.assertNotIn(TOKEN,value);self.assertNotIn(PASSWORD,value)
                self.assertNotIn('host.private.test',value)

class StageOwnershipTests(unittest.TestCase):
    def invoke(self, operation, path, run='123-1'):
        code=transport.STAGE_CREATE_CODE if operation=='create' else transport.STAGE_CLEANUP_CODE
        return subprocess.run(['python3','-c',code,str(path),run],stdout=subprocess.DEVNULL,
                              stderr=subprocess.DEVNULL,check=False).returncode
    def test_own_marker_cleanup_and_absent_stage_are_successful(self):
        with tempfile.TemporaryDirectory() as directory:
            path=Path(directory)/'stage'
            self.assertEqual(self.invoke('cleanup',path),0)
            self.assertEqual(self.invoke('create',path),0)
            self.assertEqual((path/'.owner').read_bytes(),b'123-1\n')
            self.assertEqual(stat.S_IMODE(path.stat().st_mode),0o700)
            self.assertEqual(stat.S_IMODE((path/'.owner').stat().st_mode),0o600)
            self.assertEqual(self.invoke('cleanup',path),0)
            self.assertFalse(path.exists())
    def test_existing_stage_without_marker_or_with_wrong_marker_is_preserved(self):
        with tempfile.TemporaryDirectory() as directory:
            path=Path(directory)/'stage';path.mkdir()
            value=path/'sentinel';value.write_text('unrelated')
            self.assertNotEqual(self.invoke('create',path),0)
            self.assertNotEqual(self.invoke('cleanup',path),0)
            self.assertEqual(value.read_text(),'unrelated')
            (path/'.owner').write_text('another-run\n')
            self.assertNotEqual(self.invoke('cleanup',path),0)
            (path/'.owner').write_text('123-1\nextra')
            self.assertNotEqual(self.invoke('cleanup',path),0)
            self.assertTrue(value.exists())
    def test_valid_marker_with_any_suffix_is_rejected(self):
        with tempfile.TemporaryDirectory() as directory:
            path=Path(directory)/'stage';path.mkdir()
            (path/'sentinel').write_text('unrelated')
            for suffix in (b'x',b'\n',b'\x00',b'extra'*1024):
                with self.subTest(suffix_length=len(suffix)):
                    (path/'.owner').write_bytes(b'123-1\n'+suffix)
                    self.assertNotEqual(self.invoke('cleanup',path),0)
                    self.assertTrue((path/'sentinel').exists())
            (path/'.owner').write_bytes(b'123-1\\n')
            self.assertNotEqual(self.invoke('cleanup',path),0)
    def test_stage_or_marker_symlink_never_removes_target(self):
        with tempfile.TemporaryDirectory() as directory:
            target=Path(directory)/'target';target.mkdir()
            (target/'.owner').write_text('123-1\n')
            stage=Path(directory)/'stage';stage.symlink_to(target)
            self.assertNotEqual(self.invoke('cleanup',stage),0)
            self.assertTrue(target.exists());self.assertTrue(stage.is_symlink())
            stage.unlink();stage.mkdir()
            (stage/'.owner').symlink_to(target/'.owner')
            self.assertNotEqual(self.invoke('cleanup',stage),0)
            self.assertTrue(stage.exists());self.assertTrue(target.exists())

class TransportAndWorkflowTests(unittest.TestCase):
    def test_transport_uses_private_file_never_token_argv(self):
        commands = []
        def run(command, **kwargs):
            commands.append(command)
            self.assertNotIn(TOKEN, ' '.join(command))
            if command[0] == 'scp':
                source = Path(command[-2])
                self.assertEqual(stat.S_IMODE(source.stat().st_mode), 0o600)
                self.assertEqual(stat.S_IMODE(source.parent.stat().st_mode), 0o700)
                if source.name == 'credential': self.assertEqual(source.read_text(), TOKEN)
            return subprocess.CompletedProcess(command,0,stdout=b'',stderr=b'')
        env = {'GITHUB_REF':'refs/heads/main','GITHUB_RUN_ID':'123','GITHUB_RUN_ATTEMPT':'1',
               'SVRD_HOST':'example.test','SVRD_USER':'deploy','SVRD_PORT':'22','SVRD_SSH_KEY':'PRIVATEKEY',
               'SVRD_HOST_FINGERPRINT':'','MINIBLOG_NOTION_TOKEN':TOKEN}
        output = io.StringIO()
        with mock.patch.dict(os.environ, env), mock.patch.object(transport.subprocess,'run',side_effect=run), \
             mock.patch.object(transport.sys,'argv',['transport.py','--mode','schema_check']), contextlib.redirect_stdout(output):
            self.assertEqual(transport.main(),0)
        self.assertNotIn(TOKEN,output.getvalue())
        self.assertTrue(any('mkdir' in ' '.join(c) for c in commands))
        self.assertTrue(any('shutil.rmtree(stage)' in ' '.join(c) for c in commands))
    def test_ssh_authentication_failure_is_classified_without_raw_output(self):
        env={'GITHUB_REF':'refs/heads/main','GITHUB_RUN_ID':'123','GITHUB_RUN_ATTEMPT':'1',
             'SVRD_HOST':'host.private.test','SVRD_USER':'deploy','SVRD_PORT':'22','SVRD_SSH_KEY':'PRIVATEKEY',
             'SVRD_HOST_FINGERPRINT':'','MINIBLOG_NOTION_TOKEN':TOKEN}
        def run(command,**kwargs):
            if command[0]=='ssh':
                return subprocess.CompletedProcess(command,255,stdout=b'',stderr=('Permission denied (publickey) '+TOKEN+' host.private.test').encode())
            return subprocess.CompletedProcess(command,0,stdout=b'',stderr=b'')
        output=io.StringIO()
        with mock.patch.dict(os.environ,env),mock.patch.object(transport.subprocess,'run',side_effect=run), \
             mock.patch.object(transport.sys,'argv',['transport.py','--mode','schema_check']), \
             contextlib.redirect_stdout(output),contextlib.redirect_stderr(output):
            self.assertEqual(transport.main(),1)
        self.assertIn('SSH authentication was rejected',output.getvalue())
        self.assertNotIn(TOKEN,output.getvalue());self.assertNotIn('host.private.test',output.getvalue())
    def test_stage_creation_failure_does_not_automatically_cleanup(self):
        env={'GITHUB_REF':'refs/heads/main','GITHUB_RUN_ID':'123','GITHUB_RUN_ATTEMPT':'1',
             'SVRD_HOST':'server','SVRD_USER':'deploy','SVRD_PORT':'22','SVRD_SSH_KEY':'PRIVATEKEY',
             'SVRD_HOST_FINGERPRINT':'','MINIBLOG_NOTION_TOKEN':TOKEN}
        calls=[]
        def run(command,**kwargs):
            calls.append(command)
            if command[0]=='ssh':
                return subprocess.CompletedProcess(command,1,stdout=b'',stderr=b'directory exists')
            return subprocess.CompletedProcess(command,0,stdout=b'',stderr=b'')
        output=io.StringIO()
        with mock.patch.dict(os.environ,env),mock.patch.object(transport.subprocess,'run',side_effect=run), \
             mock.patch.object(transport.sys,'argv',['transport.py','--mode','schema_check']), \
             contextlib.redirect_stdout(output),contextlib.redirect_stderr(output):
            self.assertEqual(transport.main(),1)
        ssh_calls=[call for call in calls if call[0]=='ssh']
        self.assertEqual(len(ssh_calls),1)
        self.assertNotIn('shutil.rmtree(stage)',ssh_calls[0][-1])
        self.assertNotIn(TOKEN,output.getvalue())
    def test_transport_cancellation_cleans_private_staging(self):
        commands=[]
        def run(command,**kwargs):
            commands.append(command)
            if command[0]=='ssh' and 'read-only' in command[-1]:
                raise transport.SafeError('operation cancelled')
            return subprocess.CompletedProcess(command,0,stdout=b'',stderr=b'')
        env={'GITHUB_REF':'refs/heads/main','GITHUB_RUN_ID':'123','GITHUB_RUN_ATTEMPT':'1',
             'SVRD_HOST':'server','SVRD_USER':'deploy','SVRD_PORT':'22','SVRD_SSH_KEY':'PRIVATEKEY',
             'SVRD_HOST_FINGERPRINT':'','MINIBLOG_NOTION_TOKEN':TOKEN}
        output=io.StringIO()
        with mock.patch.dict(os.environ,env), mock.patch.object(transport.subprocess,'run',side_effect=run), \
             mock.patch.object(transport.sys,'argv',['transport.py','--mode','schema_check']), \
             contextlib.redirect_stdout(output), contextlib.redirect_stderr(output):
            self.assertEqual(transport.main(),1)
        self.assertNotIn(TOKEN,output.getvalue())
        self.assertEqual(commands[-1][-1],transport.stage_command('cleanup','/tmp/miniblog-notion-ops-123-1','123-1'))
    def test_invalid_transport_never_connects_or_echoes_secret(self):
        for env in ({'GITHUB_REF':'refs/heads/feature'},
                    {'GITHUB_REF':'refs/heads/main','MINIBLOG_NOTION_TOKEN':TOKEN+'\nBAD=true',
                     'SVRD_HOST':'server','SVRD_USER':'deploy','GITHUB_RUN_ID':'1','GITHUB_RUN_ATTEMPT':'1'}):
            output = io.StringIO()
            with mock.patch.dict(os.environ,env), mock.patch.object(transport.subprocess,'run') as run, \
                 mock.patch.object(transport.sys,'argv',['transport.py','--mode','schema_check']), contextlib.redirect_stderr(output):
                self.assertEqual(transport.main(),1)
                run.assert_not_called()
            self.assertNotIn(TOKEN,output.getvalue())
    def test_bad_mode_does_not_echo_argument(self):
        for module in (remote,transport):
            error = io.StringIO()
            with mock.patch.object(__import__('sys'),'argv',['ops.py','--mode',TOKEN]), \
                 contextlib.redirect_stderr(error):
                self.assertEqual(module.main(),1)
            self.assertNotIn(TOKEN,error.getvalue())
    def test_workflows_keep_tokens_out_of_build_ssh_exports_and_write_modes(self):
        cicd=(ROOT/'.github/workflows/cicd.yml').read_text()
        readonly=(ROOT/'.github/workflows/notion-ops.yml').read_text()
        self.assertNotIn('secrets.MINIBLOG_NOTION_BOOTSTRAP_TOKEN',cicd+readonly)
        self.assertNotIn('secrets.MINIBLOG_NOTION_TOKEN',cicd.split('  deploy-app:')[0])
        self.assertEqual(cicd.count('secrets.MINIBLOG_NOTION_TOKEN'),1)
        self.assertNotIn('MINIBLOG_NOTION_TOKEN', next(line for line in cicd.splitlines() if 'envs: _IMG_' in line))
        for denied in ('bootstrap_apply','catalog_prepare','--enable-sync','upload-artifact'):
            self.assertNotIn(denied,readonly)
        self.assertIn("github.ref == 'refs/heads/main'",readonly)
        self.assertIn('group: miniblog-prod',readonly)
        self.assertIn('flock -w 900 9',cicd)
        self.assertIn('debug: false',cicd)
        self.assertIn('runtime-env --app-dir',cicd)
        # Secrets belong to env mappings, never to runnable shell/action script bodies.
        self.assertNotIn('secrets.MINIBLOG_NOTION_TOKEN',cicd[cicd.index('          script: |',cicd.index('Deploy on ServerD')):])
        for workflow in (cicd,readonly):
            for line in workflow.splitlines():
                if 'secrets.MINIBLOG_NOTION_TOKEN' in line:
                    self.assertTrue(line.strip().startswith('MINIBLOG_NOTION_TOKEN:'))

if __name__=='__main__': unittest.main()
