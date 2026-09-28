#!/usr/bin/env python3
"""Drive the real POSIX terminal through the guided setup, resizing and an unsaved exit. No downloads."""
import fcntl, os, pathlib, pty, re, select, signal, struct, subprocess, tempfile, termios, time
binary=str(pathlib.Path('bin/shoshinvim').resolve())
master,slave=pty.openpty();before=termios.tcgetattr(slave)
def size(w,h):fcntl.ioctl(slave,termios.TIOCSWINSZ,struct.pack('HHHH',h,w,0,0))
size(100,32)
env=dict(os.environ,TERM='xterm-256color',NO_COLOR='1')
with tempfile.TemporaryDirectory(prefix='shoshinvim-terminal-') as temp:
 env['XDG_DATA_HOME']=temp+'/data'
 target=temp+'/nvim'
 p=subprocess.Popen([binary,'--config-dir',target],stdin=slave,stdout=slave,stderr=slave,env=env)
 output=bytearray()
 def pump(duration=.3):
  end=time.monotonic()+duration
  while time.monotonic()<end:
   if select.select([master],[],[],.02)[0]:
    data=os.read(master,65536);output.extend(data)
    if b'\x1b[c' in data:os.write(master,b'\x1b[?1;2c')
 def send(data,duration=.3):os.write(master,data);pump(duration)
 def text():return re.sub(rb'\x1b\[[0-9;?<=>]*[ -/]*[@-~]|\x1b\][^\x07]*\x07',b'',bytes(output)).decode(errors='replace')
 def seen(word):return word in text()
 def wait_for(word,timeout):
  end=time.monotonic()+timeout
  while not seen(word) and time.monotonic()<end:pump(.2)
  return seen(word)
 try:
  assert wait_for('Start',15),'welcome did not render:\n'+text()[-2000:]
  assert not seen('Minimal native'),'presets are still offered'
  send(b'\r')                      # Start setup: the Editor form.
  assert seen('Relative'),'editor form did not render'
  for _ in range(17):send(b'\r')   # Accept every recommended choice.
  # Review validates the draft with Neovim before offering to save.
  assert wait_for('downloading',20),'review did not offer saving:\n'+text()[-2000:]
  for w,h in [(120,36),(80,24),(52,16),(40,10),(100,30)]:
   size(w,h);p.send_signal(signal.SIGWINCH);pump()
  assert seen('larger'),'below-minimum terminal was not explained'
  send(b'/');send(b'\x1b',.6)      # Command palette, then Escape.
  send(b'\x03');send(b'\x03')      # Ctrl+C twice leaves the unsaved draft.
  p.wait(timeout=8);pump(.1)
  assert p.returncode==0,p.returncode
  assert not os.path.exists(target),'an unsaved session wrote the target'
  after=termios.tcgetattr(slave)
  # PENDIN is a kernel pending-input flag, not an application terminal mode.
  after[3] &= ~getattr(termios,'PENDIN',0); before[3] &= ~getattr(termios,'PENDIN',0)
  assert after==before,('terminal settings not restored', before, after)
  assert b'\x1b[?1049h' in output and b'\x1b[?1049l' in output,'alternate screen not restored'
  print('PASS: guided setup to review, 120/80/52/40-column resizing, palette, unsaved exit, no writes and terminal restoration')
 finally:
  if p.poll() is None:p.kill();p.wait()
  os.close(master);os.close(slave)
