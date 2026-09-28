#!/usr/bin/env python3
"""Exercise the real POSIX terminal, resizing and unsaved-exit flow. No downloads."""
import fcntl, os, pathlib, pty, select, struct, subprocess, tempfile, termios, time
binary=str(pathlib.Path('bin/shoshinvim').resolve())
master,slave=pty.openpty();before=termios.tcgetattr(slave)
def size(w,h):fcntl.ioctl(slave,termios.TIOCSWINSZ,struct.pack('HHHH',h,w,0,0))
size(80,24)
env=dict(os.environ,TERM='xterm-256color',NO_COLOR='1')
with tempfile.TemporaryDirectory(prefix='shoshinvim-terminal-') as temp:
 p=subprocess.Popen([binary,'--config-dir',temp+'/nvim'],stdin=slave,stdout=slave,stderr=slave,env=env)
 output=bytearray()
 def pump(duration=.3):
  end=time.monotonic()+duration
  while time.monotonic()<end:
   if select.select([master],[],[],.02)[0]:
    data=os.read(master,65536);output.extend(data)
    if b'\x1b[c' in data:os.write(master,b'\x1b[?1;2c')
 def send(data):os.write(master,data);pump()
 try:
  pump(2)
  send(b'1\r');send(b'1\r');send(b'1\r');send(b'1\r')
  for w,h in [(120,36),(52,16),(40,10),(80,24)]:
   size(w,h);p.send_signal(__import__('signal').SIGWINCH);pump()
  send(b'\x10');send(b'\x1b') # Palette then preserve composer on Escape.
  send(b'\x04');send(b'2\r') # Unsaved exit, explicitly discard draft.
  p.wait(timeout=8);pump(.1)
  assert p.returncode==0,p.returncode
  after=termios.tcgetattr(slave)
  # PENDIN is a kernel pending-input flag, not an application terminal mode.
  after[3] &= ~getattr(termios,'PENDIN',0); before[3] &= ~getattr(termios,'PENDIN',0)
  assert after==before,('terminal settings not restored', before, after)
  assert b'\x1b[?1049h' in output and b'\x1b[?1049l' in output,'alternate screen not restored'
  assert b'Rich IDE' in output,'onboarding did not render'
  print('PASS: 120/80/52/40-column resizing, Rich IDE wizard, palette, unsaved exit and terminal restoration')
 finally:
  if p.poll() is None:p.kill();p.wait()
  os.close(master);os.close(slave)
