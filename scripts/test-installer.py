#!/usr/bin/env python3
"""Fake HTTPS downloads; real archive, hash and filesystem handling."""
import hashlib, io, json, os, pathlib, subprocess, tarfile, tempfile
ROOT = pathlib.Path(__file__).resolve().parents[1]
with tempfile.TemporaryDirectory(prefix='shoshinvim-installer-test-') as temp:
    root = pathlib.Path(temp)
    fixture = root / 'fixtures'; fixture.mkdir()
    binary = b'#!/bin/sh\necho fixture-not-executed\n'
    archive = fixture / 'archive.tar.gz'
    with tarfile.open(archive, 'w:gz') as tar:
        info = tarfile.TarInfo('shoshinvim'); info.size = len(binary); info.mode = 0o755
        tar.addfile(info, io.BytesIO(binary))
    digest = hashlib.sha256(archive.read_bytes()).hexdigest()
    osname = subprocess.check_output(['uname', '-s'], text=True).strip()
    arch = subprocess.check_output(['uname', '-m'], text=True).strip()
    triple = {('Darwin','arm64'):'aarch64-apple-darwin',('Darwin','x86_64'):'x86_64-apple-darwin',('Linux','x86_64'):'x86_64-unknown-linux-gnu',('Linux','aarch64'):'aarch64-unknown-linux-gnu'}[(osname,arch)]
    asset = f'shoshinvim_0.1.0_{triple}.tar.gz'
    (fixture / 'SHA256SUMS').write_text(f'{digest}  {asset}\n')
    mockbin = root / 'mockbin'; mockbin.mkdir()
    curl = mockbin / 'curl'
    curl.write_text('''#!/bin/sh
set -eu
url=; out=
while [ "$#" -gt 0 ]; do
 case "$1" in https://*) url=$1; shift;; -o) out=$2; shift 2;; *) shift;; esac
done
case "$url" in */SHA256SUMS) cp "$FIXTURES/SHA256SUMS" "$out";; *.tar.gz) cp "$FIXTURES/archive.tar.gz" "$out";; *) exit 2;; esac
''')
    curl.chmod(0o755)
    env = dict(os.environ, PATH=str(mockbin)+os.pathsep+os.environ['PATH'], FIXTURES=str(fixture), XDG_STATE_HOME=str(root/'state'))
    dest = root/'bin'
    command=['sh',str(ROOT/'scripts/shoshinvim-installer.sh'),'--version','v0.1.0','--install-dir',str(dest)]
    def run(extra=(), success=True):
        p=subprocess.run(command+list(extra),env=env,text=True,capture_output=True)
        assert (p.returncode == 0) == success, p.stdout+p.stderr
    run()
    assert (dest/'shoshinvim').read_bytes() == binary
    receipt=json.loads((root/'state/shoshinvim/install.json').read_text())
    assert receipt['sha256']==hashlib.sha256(binary).hexdigest()
    run(success=False)
    run(['--force'])
    (dest/'shoshinvim').write_bytes(b'unrelated manual content')
    run(['--force'],success=False)
    assert (dest/'shoshinvim').read_bytes()==b'unrelated manual content'
    (dest/'shoshinvim').write_bytes(binary)
    archive.write_bytes(b'corrupt archive')
    run(['--force'],success=False)
    assert (dest/'shoshinvim').read_bytes()==binary
    print('PASS: install, receipt, upgrade, ownership refusal, corruption refusal; no executable launched')
