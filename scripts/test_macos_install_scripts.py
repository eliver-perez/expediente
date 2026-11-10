#!/usr/bin/env python3
"""Ejecuta los scripts reales con rutas temporales y dobles de servicios del SO.

No ejecuta dscl, launchctl, sudo ni installer contra el equipo de desarrollo.
plutil, Bash y las operaciones de archivos se prueban realmente en la fixture.
"""
import hashlib
import json
from pathlib import Path
import plistlib
import subprocess
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]
MOCK = r'''
import json, os, pathlib, plistlib, signal, subprocess, sys, time, urllib.request
state_path = pathlib.Path(__file__).parent / 'state.json'
s = json.loads(state_path.read_text())
name = pathlib.Path(sys.argv[0]).name
a = sys.argv[1:]
rc = 0
if name == 'dscl':
    xml = '-plist' in a
    a = [v for v in a if v not in ('-plist', '.')]
    op, record = a[:2]
    kind = 'users' if record.startswith('/Users') else 'groups'
    key = record.split('/')[-1]
    if s.get('directory_error'): sys.exit(71)
    if op == '-list': print('\n'.join(s[kind]))
    elif op == '-search':
        attr, value = a[2:4]
        for user, attrs in s[kind].items():
            if attrs.get(attr) == value: print(user + '\t' + attr + ': ' + value)
    elif op == '-read':
        if key not in s[kind]: sys.exit(1)
        attrs = s[kind][key]
        if any(k not in attrs for k in a[2:]): sys.exit(1)
        attrs = {k:attrs[k] for k in a[2:]}
        if xml:
            print(plistlib.dumps({'dsAttrTypeStandard:'+k:[v] for k,v in attrs.items()}).decode())
        else:
            for k,v in attrs.items(): print(k + (':\n ' if ' ' in v else ': ') + v)
    elif op == '-create':
        attrs = s[kind].setdefault(key, {})
        if len(a)>2: attrs[a[2]] = a[3]
        s['account_mutations'].append([record]+a[2:])
elif name == 'sudo':
    if a[0]=='-u': a=a[2:]
    if a[0].endswith('/aidex-folder-helper'):
        if a[1:]!=['preserve-archive']: raise RuntimeError('Operación privilegiada fuera del alcance de la fixture')
        s['archive_finder_checked']=True
        state_path.write_text(json.dumps(s))
        sys.exit(0)
    if not a[0].endswith('/aidex-server'): os.execv(a[0], a)
    op = a[1]
    s['backend'].append(op)
    if s.get('native_backend'):
        state_path.write_text(json.dumps(s))
        os.execv(a[0],a)
    config = pathlib.Path(a[a.index('--config')+1])
    if op == 'recover-macos-installation':
        # Doble del inventario para probar la orquestación del shell. Las
        # decisiones de seguridad se prueban con Go y con el binario del PKG.
        if (config.parent/'state/documental.db').exists():
            print('Error: falta config.json: base de datos potencialmente recuperable; no se inicializó ni borró ningún dato',file=sys.stderr)
            rc=1
        else:
            for relative in ['logs/service.log','state/service.lock','package-version','service-enabled']:
                residue=config.parent/relative
                if residue.exists(): residue.unlink()
            print('[AIDEX recuperación] inventario validado; instalación nueva permitida')
    elif op == 'init':
        if config.exists(): sys.exit(22)
        config.write_text(json.dumps({'listen_address':'127.0.0.1:18090'}))
    elif op == 'migrate' and s.get('migration_error'): rc=73
    elif op == 'bootstrap-ready':
        if s.get('ready_error'):
            print('Error: synthetic database failure',file=sys.stderr); rc=1
        elif not s['admin']:
            print('Error: falta crear el primer administrador',file=sys.stderr); rc=1
elif name == 'launchctl':
    s['launch_calls'].append(a)
    if a[0] == 'print':
        if s.get('probe_error'): rc=s['probe_error']
        elif not s['loaded']: rc=113
        else: print('state = waiting')
    elif a[0] == 'bootout':
        if s.get('bootout_error'):
            rc=s['bootout_error']
            if s.get('disappear_on_bootout'): s['loaded']=False
        elif not s['loaded']: rc=113
        elif not s.get('stuck'):
            if s.get('native_pid'):
                try: os.kill(s['native_pid'],signal.SIGTERM)
                except ProcessLookupError: pass
                deadline=time.monotonic()+10
                while True:
                    status=subprocess.run(['/bin/ps','-p',str(s['native_pid']),'-o','stat='],capture_output=True,text=True,timeout=2).stdout.strip()
                    if not status or status.startswith('Z'): break
                    if time.monotonic()>deadline: raise RuntimeError('El servidor temporal no terminó')
                    time.sleep(.05)
                s['native_pid']=None
            s['loaded']=False
    elif a[0] == 'enable':
        s['disabled']=False
        if s.get('load_on_enable'): s['loaded']=True
    elif a[0] == 'bootstrap':
        if s['loaded'] or s['disabled']: rc=5
        else:
            s['loaded']=True
            s['loaded_arguments']=plistlib.loads(pathlib.Path(a[-1]).read_bytes())['ProgramArguments']
            if s.get('native_backend'):
                with (pathlib.Path(s['plist']).parents[1]/'Application Support/AIDEX/data/logs/service.log').open('ab') as log:
                    process=subprocess.Popen(s['loaded_arguments'],stdout=log,stderr=log,start_new_session=True)
                s['native_pid']=process.pid
    elif a[0] == 'kickstart':
        if not s['loaded']: rc=5
        else: s['loaded_arguments']=plistlib.loads(pathlib.Path(s['plist']).read_bytes())['ProgramArguments']
elif name == 'install':
    for p in a[-2:]: pathlib.Path(p).mkdir(parents=True,exist_ok=True)
elif name == 'lsof': rc=0 if s.get('occupied') else 1
elif name == 'curl':
    s['health_urls'].append(a[-1])
    if s.get('native_backend'):
        try:
            opener=urllib.request.build_opener(urllib.request.ProxyHandler({}))
            with opener.open(a[-1],timeout=2) as response: print(response.read().decode())
        except OSError: rc=22
    elif s['loaded'] and not s.get('unhealthy') and s['loaded_arguments']==s['expected_arguments']: print('{"product":"AIDEX"}')
    else: rc=22
elif name == 'pkgutil':
    if a[0]=='--pkg-info': rc=0 if s.get('receipt') else 1
    elif a[0]=='--forget': s['receipt']=False
    else: raise RuntimeError('Operación pkgutil no prevista en la fixture')
elif name not in ('chown','sleep'): raise RuntimeError(name)
state_path.write_text(json.dumps(s))
sys.exit(rc)
'''


class InstallScriptsTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix='aidex-install-José ', dir='/private/tmp')
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.base = self.root/'Library/Application Support/AIDEX'
        self.data = self.base/'data'
        self.data.mkdir(parents=True)
        self.config = self.data/'config.json'
        self.config.write_text('{"listen_address":"127.0.0.1:18973","fixture":"conservar"}\n')
        self.originals = [self.data/'state/documental.db',self.data/'state/license/installation.json',
                          self.data/'state/uploads/original.pdf',self.root/'originales/administrada.pdf',
                          self.root/'Library/Application Support/AIBID/data/existing.db']
        for p in self.originals:
            p.parent.mkdir(parents=True,exist_ok=True)
            p.write_bytes(b'contenido original de prueba')
        (self.data/'package-version').write_text('1.0.0\n')
        self.plist = self.root/'Library/LaunchDaemons/app.aidex.plist'
        self.plist.parent.mkdir(parents=True)
        self.plist.write_bytes(plistlib.dumps({'ProgramArguments':['aidex-server','serve','--config','config.json']}))
        self.manifest = self.base/'package-files.tsv'
        self.manifest.write_text(hashlib.sha256(self.plist.read_bytes()).hexdigest()+'\t'+str(self.plist)+'\n')
        self.mock = self.root/'mock'
        self.mock.mkdir()
        self.state_path = self.mock/'state.json'
        self.state_path.write_text(json.dumps({
            'users':{'_aidex':{'NFSHomeDirectory':str(self.data),'UserShell':'/usr/bin/false','UniqueID':'402','PrimaryGroupID':'402'}},
            'groups':{'_aidex':{'PrimaryGroupID':'402'}},'admin':True,'loaded':True,'disabled':False,
            'backend':[],'account_mutations':[],'launch_calls':[],'health_urls':[],
            'plist':str(self.plist),
            'expected_arguments':[str(self.base/'bin/aidex-server'),'serve','--config',str(self.config)],
            'loaded_arguments':[str(self.base/'bin/aidex-server'),'serve','--config',str(self.config)]}))
        substitutions = {}
        for system_path in ['/usr/bin/dscl','/usr/bin/sudo','/bin/launchctl','/usr/bin/install',
                            '/usr/sbin/chown','/usr/sbin/lsof','/usr/bin/curl','/bin/sleep']:
            target = self.mock/Path(system_path).name
            target.write_text('#!'+sys.executable+'\n'+MOCK)
            target.chmod(0o755)
            substitutions[system_path] = '"'+str(target)+'"'
        self.substitutions = substitutions
        self.scripts = self.root/'scripts'
        self.scripts.mkdir()
        self.prepare_scripts(ROOT/'packaging/macos')

    def prepare_scripts(self, source_directory):
        """Redirigir únicamente rutas y herramientas del SO, también desde PKG."""
        for name in ['preinstall','postinstall','profile.sh','installer-common.sh']:
            source = (source_directory/name).read_text().replace('@VERSION@','1.0.0')
            if name == 'profile.sh': source=source.replace('prefix=${1-}', "prefix='"+str(self.root)+"'")
            source=source.replace("'/Applications/AIDEX", "'"+str(self.root)+"/Applications/AIDEX")
            for old,new in self.substitutions.items(): source=source.replace(old,new)
            (self.scripts/name).write_text(source)

    def state(self): return json.loads(self.state_path.read_text())
    def change(self, **values):
        state=self.state(); state.update(values); self.state_path.write_text(json.dumps(state))
    def run_script(self, name, success=True):
        result=subprocess.run(['/bin/bash',str(self.scripts/name)],capture_output=True,text=True,timeout=35)
        self.assertEqual(result.returncode==0,success,result.stdout+result.stderr)
        if success and name=='postinstall': self.assert_program_arguments()
        return result
    def assert_program_arguments(self):
        arguments=plistlib.loads(self.plist.read_bytes())['ProgramArguments']
        self.assertEqual(arguments,self.state()['expected_arguments'])
        self.assertEqual(len(arguments),4)
    def assert_preserved(self, config):
        self.assertEqual(self.config.read_bytes(),config)
        for p in self.originals: self.assertEqual(p.read_bytes(),b'contenido original de prueba')
        self.assertFalse(any('aibid' in ' '.join(a).lower() for a in self.state()['launch_calls']))

    def test_reproduces_old_dscl_comparison_and_fixes_upgrade(self):
        raw=subprocess.check_output([str(self.mock/'dscl'),'.','-read','/Users/_aidex','NFSHomeDirectory'],text=True).rstrip('\n')
        self.assertEqual(raw,'NFSHomeDirectory:\n '+str(self.data))
        self.assertNotEqual(raw,'NFSHomeDirectory: '+str(self.data))
        config=self.config.read_bytes()
        self.run_script('preinstall'); self.run_script('postinstall')
        self.assert_preserved(config)
        s=self.state()
        self.assertNotIn('init',s['backend']); self.assertNotIn('configure-license',s['backend'])
        self.assertEqual(s['users']['_aidex']['UniqueID'],'402')
        self.assertEqual(s['health_urls'][-1],'http://127.0.0.1:18973/health/live')

    def test_reinstall_twice(self):
        config=self.config.read_bytes()
        for _ in range(2): self.run_script('preinstall'); self.run_script('postinstall')
        self.assert_preserved(config)
        self.assertEqual(sum(a[0]=='bootstrap' for a in self.state()['launch_calls']),2)

    def test_recovers_duplicate_configuration_in_loaded_daemon(self):
        info=plistlib.loads(self.plist.read_bytes())
        info['ProgramArguments']=self.state()['expected_arguments']+[str(self.config)]
        self.plist.write_bytes(plistlib.dumps(info))
        self.change(loaded_arguments=info['ProgramArguments'])
        config=self.config.read_bytes()
        self.run_script('preinstall'); self.run_script('postinstall')
        self.assert_preserved(config)
        self.assertEqual(self.state()['loaded_arguments'],self.state()['expected_arguments'])
        digest,path=self.manifest.read_text().strip().split('\t')
        self.assertEqual(path,str(self.plist))
        self.assertEqual(digest,hashlib.sha256(self.plist.read_bytes()).hexdigest())

    def test_recovers_extra_arguments_in_unregistered_daemon(self):
        info=plistlib.loads(self.plist.read_bytes())
        info['ProgramArguments']=self.state()['expected_arguments']+[str(self.config),'extra','--config',str(self.config)]
        self.plist.write_bytes(plistlib.dumps(info))
        self.change(loaded=False)
        self.run_script('postinstall')
        self.assertEqual(self.state()['loaded_arguments'],self.state()['expected_arguments'])

    def test_rebuild_escapes_xml_and_remains_idempotent(self):
        binary=str(self.root/'Aplicación & prueba <arm> "1"/aidex-server')
        config=str(self.root/'Documentación & <東京> "1"/config.json')
        for _ in range(3):
            subprocess.run(['/bin/bash','-Eeuo','pipefail','-c',
                '. "$1"; BINARY=$2; CONFIG=$3; PLIST=$4; aidex_rebuild_program_arguments',
                'aidex-plist-test',str(self.scripts/'installer-common.sh'),binary,config,str(self.plist)],
                check=True,capture_output=True,text=True,timeout=10)
            self.assertEqual(plistlib.loads(self.plist.read_bytes())['ProgramArguments'],[binary,'serve','--config',config])

    def test_clean_install_without_admin(self):
        self.change(users={},groups={},admin=False,loaded=False)
        self.config.unlink(); (self.data/'package-version').unlink()
        for p in self.originals[:3]: p.unlink()
        self.run_script('preinstall')
        result=self.run_script('postinstall')
        self.assertIn('primer administrador',result.stdout)
        self.assertIn('init',self.state()['backend'])
        self.assertFalse(self.state()['loaded'])

    def test_partial_configuration_without_version_marker(self):
        (self.data/'package-version').unlink()
        self.change(loaded=False)
        config=self.config.read_bytes()
        self.run_script('preinstall'); self.run_script('postinstall')
        self.assert_preserved(config)
        self.assertTrue(self.state()['loaded'])

    def test_partial_group_without_user(self):
        self.change(users={},loaded=False)
        self.run_script('postinstall')
        self.assertEqual(self.state()['users']['_aidex']['UniqueID'],'402')

    def test_disabled_unregistered_service(self):
        self.change(loaded=False,disabled=True)
        self.run_script('preinstall'); self.run_script('postinstall')
        self.assertTrue(self.state()['loaded']); self.assertFalse(self.state()['disabled'])

    def test_loaded_stopped_job_and_registration_race(self):
        self.change(load_on_enable=True)
        self.run_script('postinstall')
        calls=[a[0] for a in self.state()['launch_calls']]
        self.assertEqual(calls.count('bootout'),2)
        self.assertEqual(calls.count('bootstrap'),1)
        self.assertNotIn('kickstart',calls)
        self.assertEqual(self.state()['loaded_arguments'],self.state()['expected_arguments'])

    def test_missing_configuration_never_reinitializes_data(self):
        self.config.unlink()
        result=self.run_script('postinstall',False)
        self.assertIn('no se inicializó',result.stderr)
        self.assertNotIn('init',self.state()['backend'])
        for p in self.originals: self.assertEqual(p.read_bytes(),b'contenido original de prueba')

    def test_legacy_uninstall_residues_recover_without_uninstaller(self):
        self.config.unlink()
        for path in self.originals[:3]: path.unlink()
        (self.data/'logs').mkdir()
        (self.data/'logs/service.log').write_text('registro anterior')
        (self.data/'state/service.lock').touch()
        self.change(loaded=False,admin=False)
        result=self.run_script('postinstall')
        self.assertIn('recuperar-residuos-reconocidos',result.stdout)
        self.assertIn('primer administrador',result.stdout)
        self.assertLess(self.state()['backend'].index('recover-macos-installation'),self.state()['backend'].index('init'))
        self.assertNotIn('ERROR',result.stdout+result.stderr)

    def test_absent_service_113_is_expected_and_logged(self):
        self.change(loaded=False)
        result=self.run_script('preinstall')
        self.assertIn('servicio=ausente',result.stdout)
        self.assertNotIn('ERROR',result.stdout+result.stderr)

    def test_service_probe_real_error_stops_installation(self):
        self.change(probe_error=5)
        result=self.run_script('preinstall',False)
        self.assertIn('launchctl print, código 5',result.stderr)
        self.assertFalse(self.state()['backend'])

    def test_bootout_real_error_stops_installation(self):
        self.change(bootout_error=5)
        result=self.run_script('preinstall',False)
        self.assertIn('bootout, código 5',result.stderr)
        self.assertFalse(self.state()['backend'])

    def test_bootout_113_requires_confirmed_absence(self):
        self.change(bootout_error=113)
        result=self.run_script('preinstall',False)
        self.assertIn('sigue registrado',result.stderr)
        self.change(disappear_on_bootout=True)
        result=self.run_script('preinstall')
        self.assertNotIn('ERROR',result.stdout+result.stderr)

    def test_directory_service_error_does_not_create_user(self):
        self.change(directory_error=True)
        result=self.run_script('postinstall',False)
        self.assertIn('validar-cuenta-existente',result.stderr)
        self.assertFalse(self.state()['account_mutations'])

    def test_real_bootstrap_error_not_reported_as_missing_admin(self):
        self.change(ready_error=True)
        result=self.run_script('postinstall',False)
        self.assertIn('bootstrap-ready',result.stderr)
        self.assertFalse(self.state()['loaded'])

    def test_failing_command_logs_stage_line_and_code(self):
        self.change(migration_error=True)
        result=self.run_script('postinstall',False)
        self.assertIn('etapa=verificar-migraciones',result.stderr)
        self.assertIn('archivo=postinstall linea=',result.stderr)
        self.assertIn('codigo=73',result.stderr)
        self.assertNotIn('contenido original',result.stderr)

    def test_occupied_port_preserves_data(self):
        self.change(occupied=True)
        config=self.config.read_bytes()
        result=self.run_script('postinstall',False)
        self.assertIn('18973 está ocupado',result.stderr)
        self.assert_preserved(config)

    def test_http_timeout_is_bounded(self):
        self.change(unhealthy=True)
        result=self.run_script('postinstall',False)
        self.assertIn('verificar-respuesta-http',result.stderr)
        self.assertEqual(len(self.state()['health_urls']),30)

    def test_shutdown_timeout_prevents_migration(self):
        self.change(stuck=True)
        result=self.run_script('preinstall',False)
        self.assertIn('60 segundos',result.stderr)
        self.assertFalse(self.state()['backend'])


if __name__=='__main__': unittest.main()
