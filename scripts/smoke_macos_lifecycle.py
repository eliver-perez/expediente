#!/usr/bin/env python3
"""Ciclos con scripts y binario reales del PKG; cuentas/launchd/recibo simulados.

Todas las rutas se redirigen a una fixture. Nunca ejecuta el desinstalador
gráfico (su auxiliar apunta a la instalación real) ni los servicios del sistema.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil
import signal
import socket
import sqlite3
import subprocess
import sys
import tempfile
import urllib.request

from smoke_macos_reinstall import bootstrap, extract
from test_macos_install_scripts import InstallScriptsTest, MOCK


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('package',type=Path)
    args=parser.parse_args()
    fixture=InstallScriptsTest()
    fixture.setUp()
    try:
        with tempfile.TemporaryDirectory(prefix='aidex-lifecycle-package-',dir='/private/tmp') as temporary:
            binary=extract(args.package.resolve(),Path(temporary)/'expanded')
            payload=binary.parents[4]
            # Ejecutar los scripts extraídos del paquete entregable, con las
            # únicas sustituciones de rutas/cuentas/launchd propias de la fixture.
            fixture.prepare_scripts(binary.parents[5]/'Scripts')
            for item in (binary.parent.parent/'package-files.tsv').read_text().splitlines():
                digest,path=item.split('\t',1)
                assert hashlib.sha256((payload/path.lstrip('/')).read_bytes()).hexdigest()==digest
            # Estos archivos pertenecen exclusivamente a la fixture del test.
            shutil.rmtree(fixture.base)
            fixture.change(users={},groups={},loaded=False,admin=False,native_backend=True,receipt=False)
            pkgutil=fixture.mock/'pkgutil'
            pkgutil.write_text('#!'+sys.executable+'\n'+MOCK)
            pkgutil.chmod(0o755)
            def install():
                fixture.run_script('preinstall')
                shutil.copytree(payload,fixture.root,dirs_exist_ok=True)
                fixture.manifest.write_text(''.join(digest+'\t'+str(fixture.root/path.lstrip('/'))+'\n'
                    for digest,path in (line.split('\t',1) for line in (binary.parent.parent/'package-files.tsv').read_text().splitlines())))
                result=fixture.run_script('postinstall')
                fixture.change(receipt=True)
                return result

            def uninstall(action):
                source=(fixture.root/'Applications/AIDEX/Desinstalar AIDEX.app/Contents/Resources/uninstall-helper.sh').read_text()
                for old,new in [("CODE='/Library/Application Support/AIDEX'",f"CODE='{fixture.base}'"),
                                ("APP='/Applications/AIDEX/Desinstalar AIDEX.app'",f"APP='{fixture.root}/Applications/AIDEX/Desinstalar AIDEX.app'"),
                                ("APPS='/Applications/AIDEX'",f"APPS='{fixture.root}/Applications/AIDEX'"),
                                ("PLIST='/Library/LaunchDaemons/app.aidex.plist'",f"PLIST='{fixture.plist}'"),
                                ('$(/usr/bin/id -u)','0')]:
                    assert old in source
                    source=source.replace(old,new)
                for old,name in [('/bin/launchctl','launchctl'),('/usr/bin/sudo','sudo'),('/usr/sbin/pkgutil','pkgutil')]:
                    source=source.replace(old,'"'+str(fixture.mock/name)+'"')
                helper=fixture.root/'uninstall-fixture.sh'
                helper.write_text(source)
                result=subprocess.run(['/bin/sh',str(helper),action],capture_output=True,text=True,timeout=45)
                if result.returncode: raise RuntimeError(result.stdout+result.stderr)
                assert not fixture.state()['loaded'] and not fixture.state()['receipt']
                assert not fixture.plist.exists()
                assert not (fixture.base/'bin/aidex-server').exists()
                return result.stdout

            def prepare_admin():
                with socket.socket() as reservation:
                    reservation.bind(('127.0.0.1',0)); port=reservation.getsockname()[1]
                options=json.loads(fixture.config.read_text())
                options.update(listen_address=f'127.0.0.1:{port}',public_url=f'http://127.0.0.1:{port}')
                options['license'].update(server_url='https://127.0.0.1:1',refresh_hours=0)
                fixture.config.write_text(json.dumps(options,ensure_ascii=False,indent=2)+'\n')
                password=bootstrap(fixture.base/'bin/aidex-server',fixture.config)
                fixture.run_script('postinstall')
                return password,port

            def login(password,port):
                opener=urllib.request.build_opener(urllib.request.ProxyHandler({}))
                request=urllib.request.Request(f'http://127.0.0.1:{port}/api/v1/auth/login',
                    data=json.dumps({'username':'qa-macos','password':password}).encode(),
                    headers={'Content-Type':'application/json','Origin':f'http://127.0.0.1:{port}'})
                with opener.open(request,timeout=10) as response:
                    assert response.status==200

            install()
            print('PASS instalación limpia; primer administrador pendiente',flush=True)
            assert not fixture.state()['loaded']  # Primer administrador pendiente.
            password,port=prepare_admin()
            login(password,port)
            state=fixture.data/'state'
            config_bytes=fixture.config.read_bytes()
            license_bytes=(state/'license/installation.json').read_bytes()
            external=[]
            for name in ['vinculada','administrada','híbrida']:
                original=fixture.root/'documentos externos'/name/'original.pdf'
                original.parent.mkdir(parents=True); original.write_bytes(b'original sintetico')
                external.append(original)
            uploads=state/'uploads/carga.pdf'
            uploads.parent.mkdir(parents=True,exist_ok=True); uploads.write_bytes(b'carga sintetica')
            # Contenido original alojado dentro de data, fuera del estado privado.
            managed=fixture.data/'documentos administrados/original.docx'
            managed.parent.mkdir(); managed.write_bytes(b'original administrado')
            with sqlite3.connect(state/'documental.db') as db:
                db.execute("INSERT INTO libraries(id,name,mode,created_at,updated_at) VALUES('qa','Sintética','hybrid','2026','2026')")
                user=db.execute("SELECT id,password_hash FROM users WHERE username='qa-macos'").fetchone()
            install()  # Actualización directa con el servicio activo.
            login(password,port)
            print('PASS actualización directa y acceso del administrador anterior',flush=True)
            uninstall('conservar')
            assert fixture.config.read_bytes()==config_bytes
            assert (state/'license/installation.json').read_bytes()==license_bytes
            print('PASS conservación y reinstalación de configuración, usuarios, biblioteca y licencia',flush=True)
            assert uploads.read_bytes()==b'carga sintetica'
            install()
            login(password,port)  # El administrador anterior sigue siendo válido.
            with sqlite3.connect(state/'documental.db') as db:
                assert db.execute("SELECT id,password_hash FROM users WHERE username='qa-macos'").fetchone()==user
                assert db.execute('SELECT count(*) FROM libraries').fetchone()[0]==1
            assert fixture.config.read_bytes()==config_bytes
            assert (state/'license/installation.json').read_bytes()==license_bytes
            # Una base/licencia sin config NO es un residuo de desinstalación.
            # Probar el rechazo desde el postinstall extraído, conservando bytes.
            subprocess.run([str(fixture.mock/'launchctl'),'bootout','system/app.aidex'],
                check=True,capture_output=True,text=True,timeout=15)
            fixture.config.unlink()
            protected={p:p.read_bytes() for p in fixture.data.rglob('*') if p.is_file()}
            init_count=fixture.state()['backend'].count('init')
            rejection=fixture.run_script('postinstall',False)
            assert 'potencialmente recuperable' in rejection.stderr
            assert not fixture.config.exists()
            assert fixture.state()['backend'].count('init')==init_count
            for path,content in protected.items(): assert path.read_bytes()==content
            fixture.config.write_bytes(config_bytes)
            install()
            login(password,port)
            print('PASS rechazo de datos recuperables sin configuración; bytes intactos',flush=True)
            result=uninstall('eliminar')
            assert not fixture.data.exists()
            assert 'Eliminación completa de datos internos verificada' in result
            archives=fixture.base/'Documentos conservados'
            assert next(archives.rglob('carga.pdf')).read_bytes()==b'carga sintetica'
            assert next(archives.rglob('original.docx')).read_bytes()==b'original administrado'
            assert not list(archives.rglob('service.log'))
            assert not list(archives.rglob('documental.db'))
            assert not list(archives.rglob('installation.json'))
            for cycle in range(2):
                install()
                assert not fixture.state()['loaded']
                with sqlite3.connect(fixture.data/'state/documental.db') as db:
                    assert db.execute('SELECT count(*) FROM users').fetchone()[0]==0
                    assert db.execute('SELECT count(*) FROM libraries').fetchone()[0]==0
                    assert db.execute('PRAGMA integrity_check').fetchone()[0]=='ok'
                # La primera cuenta puede crearse de nuevo después del borrado.
                fresh_password,fresh_port=prepare_admin()
                login(fresh_password,fresh_port)
                assert (fixture.data/'state/license/installation.json').read_bytes()!=license_bytes
                if cycle==1:
                    # Reproducir exactamente el borrado antiguo: sin config ni
                    # SQLite, pero con logs y service.lock. Reinstalar DIRECTO:
                    # no ejecutar el desinstalador corregido para facilitarlo.
                    subprocess.run([str(fixture.mock/'launchctl'),'bootout','system/app.aidex'],
                        check=True,capture_output=True,text=True,timeout=15)
                    subprocess.run([str(fixture.base/'bin/aidex-server'),'erase-internal-data','--config',str(fixture.config),'--confirm-erase-internal-data'],
                        check=True,capture_output=True,text=True,timeout=15)
                    assert not fixture.config.exists()
                    assert (fixture.data/'logs/service.log').exists()
                    assert (fixture.data/'state/service.lock').exists()
                    assert not list(fixture.data.rglob('*.db*'))
                    recovery=install()
                    assert 'residuo retirado="logs/service.log"' in recovery.stdout
                    assert 'residuo retirado="state/service.lock"' in recovery.stdout
                    assert 'ERROR' not in recovery.stdout+recovery.stderr
                    assert not fixture.state()['loaded']
                    with sqlite3.connect(fixture.data/'state/documental.db') as db:
                        assert db.execute('SELECT count(*) FROM users').fetchone()[0]==0
                        assert db.execute('PRAGMA integrity_check').fetchone()[0]=='ok'
                    recovered_password,recovered_port=prepare_admin()
                    login(recovered_password,recovered_port)
                    print('PASS postinstall del PKG recupera residuos antiguos sin desinstalador; administrador nuevo y HTTP reales',flush=True)
                uninstall('eliminar')
                assert not fixture.data.exists()
            for original in external: assert original.read_bytes()==b'original sintetico'
            assert next(archives.rglob('carga.pdf')).read_bytes()==b'carga sintetica'
            assert fixture.state()['users']['_aidex']['NFSHomeDirectory']==str(fixture.data)
            print(json.dumps({'result':'PASS','direct_update':True,'keep_reinstall_previous_admin_login':True,
                'complete_erase_fresh_install_repeated':2,'data_directory_absent_after_every_erase':True,
                'configuration_users_library_local_license_preserved_in_keep_mode':True,
                'external_and_uploaded_originals_preserved':True,
                'previous_partial_erase_logs_and_lock_recovered_directly_by_packaged_postinstall':True,
                'first_admin_creation_and_real_server_login_after_recovery':True,
                'packaged_postinstall_refuses_recoverable_data_without_configuration':True,
                'not_tested':'Installer.app, cuenta/launchd/recibo reales y autorización gráfica'},ensure_ascii=False,indent=2))
    finally:
        try:
            pid=fixture.state().get('native_pid')
            if pid:
                try:
                    subprocess.run([str(fixture.mock/'launchctl'),'bootout','system/app.aidex'],
                        capture_output=True,text=True,check=True,timeout=15)
                except (subprocess.SubprocessError,OSError):
                    try: os.kill(pid,signal.SIGKILL)
                    except ProcessLookupError: pass
        finally: fixture.doCleanups()


if __name__=='__main__': main()
