#!/usr/bin/env python3
"""Prueba nativa del binario del PKG con datos temporales y puerto libre.

No instala paquetes, no ejecuta launchctl/dscl ni accede a licencias reales.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import plistlib
import pty
import secrets
import select
import socket
import sqlite3
import subprocess
import tempfile
import time
import urllib.request


def run(args, success=True):
    result = subprocess.run(list(map(str,args)), capture_output=True, text=True, timeout=60)
    if (result.returncode == 0) != success:
        raise RuntimeError(result.stdout + result.stderr)
    return result.stdout.strip()


def stop(process):
    if process.poll() is None:
        process.terminate()
        try: process.wait(timeout=5)
        except subprocess.TimeoutExpired:
            process.kill(); process.wait(timeout=5)


def bootstrap(binary, config):
    master, slave = pty.openpty()
    process = subprocess.Popen([str(binary),'bootstrap','--config',str(config)],stdin=slave,stdout=slave,stderr=slave)
    os.close(slave)
    password = secrets.token_urlsafe(18)
    try:
        pending=b''
        deadline=time.monotonic()+30
        for prompt,answer in [('Usuario:','qa-macos'),('Nombre visible:','Prueba reinstalación'),
                             ('Nueva contraseña:',password),('Repite la contraseña:',password)]:
            while prompt.encode() not in pending:
                if process.poll() is not None: raise RuntimeError('El bootstrap terminó antes de completar los campos')
                if time.monotonic()>deadline: raise RuntimeError('Tiempo de bootstrap agotado')
                if select.select([master],[],[],.1)[0]: pending+=os.read(master,4096)
            pending=pending.split(prompt.encode(),1)[1]
            os.write(master,(answer+'\n').encode())
        assert process.wait(timeout=10)==0
        return password
    finally:
        stop(process)
        os.close(master)


def extract(package, destination):
    run(['pkgutil','--expand-full',package,destination])
    return next(destination.rglob('Payload/Library/Application Support/AIDEX/bin/aidex-server'))


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('package',type=Path)
    parser.add_argument('--previous-package',type=Path)
    args=parser.parse_args()
    with tempfile.TemporaryDirectory(prefix='aidex-reinstalar-José ',dir='/private/tmp') as temporary:
        root=Path(temporary)
        binary=extract(args.package.resolve(),root/'package')
        old=extract(args.previous_package.resolve(),root/'previous') if args.previous_package else binary
        payload=binary.parents[4]
        base=binary.parent.parent
        app=payload/'Applications/AIDEX/Desinstalar AIDEX.app'
        daemon=payload/'Library/LaunchDaemons/app.aidex.plist'
        expected_packaged=['/Library/Application Support/AIDEX/bin/aidex-server','serve','--config','/Library/Application Support/AIDEX/data/config.json']
        assert plistlib.loads(daemon.read_bytes())['ProgramArguments']==expected_packaged
        for item in (base/'package-files.tsv').read_text().splitlines():
            digest,name=item.split('\t',1)
            path=payload/name.lstrip('/')
            assert path.resolve().is_relative_to(payload.resolve())
            assert hashlib.sha256(path.read_bytes()).hexdigest()==digest
        run(['codesign','--verify','--deep','--strict',app])
        run(['codesign','--verify','--strict',binary])
        info=plistlib.loads((app/'Contents/Info.plist').read_bytes())
        assert info['CFBundleVersion']=='1.0.0'
        assert info['CFBundleIconFile']=='aidex.icns'
        for name in ['preinstall','postinstall','installer-common.sh']:
            script=next((root/'package').rglob('Scripts/'+name))
            run(['/bin/bash','-n',script])
        helper=next((root/'package').rglob('Scripts/installer-common.sh'))
        with socket.socket() as reservation:
            reservation.bind(('127.0.0.1',0)); port=reservation.getsockname()[1]
        config=root/'privado/config.json'
        # Instalación limpia: migrar no exige que ya exista el administrador.
        fresh=root/'nueva/config.json'
        run([binary,'init','--config',fresh,'--listen',f'127.0.0.1:{port}'])
        run([binary,'migrate','--config',fresh])
        run([binary,'bootstrap-ready','--config',fresh],success=False)
        run([old,'init','--config',config,'--listen',f'127.0.0.1:{port}'])
        options=json.loads(config.read_text())
        # Destino local sin servidor: la fixture no puede contactar producción.
        options['license']['server_url']='https://127.0.0.1:1'
        options['license']['refresh_hours']=0
        config.write_text(json.dumps(options,ensure_ascii=False,indent=2)+'\n')
        config_bytes=config.read_bytes()
        run([old,'migrate','--config',config])
        run([old,'bootstrap-ready','--config',config],success=False)
        bootstrap(old,config)
        state=Path(options['state_directory'])
        identity=state/'license/installation.json'
        identity_bytes=identity.read_bytes()
        original=root/'documentos originales/Documento 東京.pdf'
        original.parent.mkdir(); original.write_bytes(b'original sintetico')
        uploads=state/'uploads/prueba.pdf'
        uploads.parent.mkdir(exist_ok=True); uploads.write_bytes(b'carga sintetica')
        with sqlite3.connect(state/'documental.db') as db:
            db.execute("INSERT INTO libraries(id,name,mode,created_at,updated_at) VALUES('qa-library','Biblioteca sintética','managed','2026-10-29T00:00:00Z','2026-10-29T00:00:00Z')")
            db.commit()
            users=db.execute('SELECT * FROM users ORDER BY id').fetchall()
            audit=db.execute('SELECT * FROM audit_events ORDER BY id').fetchall()
            libraries=db.execute('SELECT * FROM libraries ORDER BY id').fetchall()
        for _ in range(2):
            run([binary,'init','--config',config],success=False)
            run([binary,'check-state','--config',config])
            run([binary,'migrate','--config',config])
            run([binary,'bootstrap-ready','--config',config])
            assert config.read_bytes()==config_bytes
            assert identity.read_bytes()==identity_bytes
            with sqlite3.connect(state/'documental.db') as db:
                assert db.execute('SELECT * FROM users ORDER BY id').fetchall()==users
                assert db.execute('SELECT * FROM audit_events ORDER BY id').fetchall()==audit
                assert db.execute('SELECT * FROM libraries ORDER BY id').fetchall()==libraries
                assert db.execute('PRAGMA integrity_check').fetchone()[0]=='ok'
                assert not db.execute('PRAGMA foreign_key_check').fetchall()
        opener=urllib.request.build_opener(urllib.request.ProxyHandler({}))
        for _ in range(2):
            # Corromper y reparar el LaunchDaemon temporal con el auxiliar real
            # del paquete. Arrancar leyendo el arreglo resultante, en vez de
            # construir a mano un comando correcto que ocultaría el defecto.
            launch_info=plistlib.loads(daemon.read_bytes())
            launch_info['ProgramArguments'] += [str(config),str(config)]
            daemon.write_bytes(plistlib.dumps(launch_info))
            run(['/bin/bash','-Eeuo','pipefail','-c',
                 '. "$1"; BINARY=$2; CONFIG=$3; PLIST=$4; aidex_rebuild_program_arguments',
                 'aidex-plist-test',helper,binary,config,daemon])
            launch_arguments=plistlib.loads(daemon.read_bytes())['ProgramArguments']
            assert launch_arguments==[str(binary),'serve','--config',str(config)]
            assert len(launch_arguments)==4
            with (root/'service.log').open('ab') as log:
                service=subprocess.Popen(launch_arguments,stdout=log,stderr=log)
                try:
                    deadline=time.monotonic()+20
                    while True:
                        if service.poll() is not None: raise RuntimeError('El servicio temporal terminó prematuramente')
                        try:
                            with opener.open(f'http://127.0.0.1:{port}/health/live',timeout=1) as response:
                                assert json.load(response)['product']=='AIDEX'
                            break
                        except OSError:
                            if time.monotonic()>deadline: raise RuntimeError('HTTP temporal no respondió en 20 segundos')
                            time.sleep(.1)
                    with opener.open(f'http://127.0.0.1:{port}/',timeout=3) as response:
                        assert b'AIDEX' in response.read()
                    run([binary,'check-state','--config',config],success=False)
                finally: stop(service)
            assert service.returncode==0
        assert config.read_bytes()==config_bytes
        assert identity.read_bytes()==identity_bytes
        assert original.read_bytes()==b'original sintetico'
        assert uploads.read_bytes()==b'carga sintetica'
        print(json.dumps({'result':'PASS','version':run([binary,'version']),
            'previous_package':bool(args.previous_package),'migrations_repeated':2,'http_starts':2,
            'config_license_users_audit_libraries_originals_preserved':True,
            'clean_install_without_admin':'PASS',
            'duplicate_arguments_repaired_and_exactly_four_after_each_reinstall':'PASS',
            'package_manifest_and_app_signatures':'PASS',
            'not_tested':'Installer.app, elevación gráfica y launchd del sistema real'},ensure_ascii=False,indent=2))


if __name__=='__main__': main()
