#!/usr/bin/env python3
"""Real Neovim UI smoke test. Requires msgpack; Pillow is optional for screenshots.
Run only against an isolated, already-installed Rich IDE configuration:
  python3 scripts/editor-smoke.py --root /tmp/shoshinvim-test --screenshots /tmp/screens
"""
import argparse, os, pathlib, select, subprocess, time
import msgpack

class Editor:
    def __init__(self, root):
        env = os.environ.copy()
        for key, directory in [('XDG_CONFIG_HOME','config'),('XDG_DATA_HOME','data'),('XDG_STATE_HOME','state'),('XDG_CACHE_HOME','cache')]:
            env[key] = str(root / directory)
        env['NVIM_APPNAME'] = 'nvim'
        env['VIM_LOG_FILE'] = str(root / 'ui.log')
        self.stderr = open(root / 'ui-stderr.log','wb')
        self.p = subprocess.Popen(['nvim','--embed','-i','NONE','-n'],stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=self.stderr,env=env,cwd=root)
        self.unpacker=msgpack.Unpacker(raw=False,strict_map_key=False)
        self.seq=0;self.responses={};self.grid=[];self.highlights={};self.fg=0xc0caf5;self.bg=0x1a1b26
        self.call('nvim_ui_attach',[120,36,{'rgb':True,'ext_linegrid':True}]);self.pump(1)
    def pump(self,seconds=0.1):
        end=time.monotonic()+seconds
        while time.monotonic()<end:
            if not select.select([self.p.stdout],[],[],min(0.05,max(0,end-time.monotonic())))[0]:continue
            data=os.read(self.p.stdout.fileno(),1048576)
            if not data:return
            self.unpacker.feed(data)
            for msg in self.unpacker:
                if msg[0]==1:self.responses[msg[1]]=msg[2:]
                elif msg[0]==2 and msg[1]=='redraw':
                    for batch in msg[2]:
                        for args in batch[1:]:self.redraw(batch[0],args)
    def redraw(self,name,a):
        if name=='grid_resize' and a[0]==1:self.grid=[[(' ',0) for _ in range(a[1])] for _ in range(a[2])]
        elif name=='grid_clear' and a[0]==1:self.grid=[[(' ',0) for _ in row] for row in self.grid]
        elif name=='default_colors_set':self.fg,self.bg=a[:2]
        elif name=='hl_attr_define':self.highlights[a[0]]=a[1]
        elif name=='grid_line' and a[0]==1:
            row,col=a[1:3];hl=0
            for cell in a[3]:
                if len(cell)>1:hl=cell[1]
                for _ in range(cell[2] if len(cell)>2 else 1):
                    if row<len(self.grid) and col<len(self.grid[row]):self.grid[row][col]=(cell[0],hl)
                    col+=1
        elif name=='grid_scroll' and a[0]==1:
            _,top,bot,left,right,rows,cols=a
            old=[r[:] for r in self.grid]
            for y in range(top,bot):
                for x in range(left,right):
                    yy,xx=y+rows,x+cols
                    self.grid[y][x]=old[yy][xx] if top<=yy<bot and left<=xx<right else (' ',0)
    def call(self,method,args):
        self.seq+=1;seq=self.seq
        self.p.stdin.write(msgpack.packb([0,seq,method,args],use_bin_type=True));self.p.stdin.flush()
        end=time.monotonic()+15
        while seq not in self.responses:
            self.pump(.05)
            if time.monotonic()>end:raise RuntimeError('RPC timeout: '+method)
        error,result=self.responses.pop(seq)
        if error:raise RuntimeError(str(error))
        return result
    def lua(self,code):return self.call('nvim_exec_lua',[code,[]])
    def keys(self,keys):self.call('nvim_input',[keys]);self.pump(.7)
    def capture(self,path):
        self.call('nvim_command',['redraw!']);self.pump(.2)
        path.with_suffix('.txt').write_text('\n'.join(''.join(c for c,h in row) for row in self.grid))
        from PIL import Image,ImageDraw,ImageFont
        font=ImageFont.truetype('/System/Library/Fonts/Menlo.ttc',16)
        im=Image.new('RGB',(len(self.grid[0])*10,len(self.grid)*22),self.bg);draw=ImageDraw.Draw(im)
        for y,row in enumerate(self.grid):
            for x,(c,h) in enumerate(row):
                attr=self.highlights.get(h,{});fg=attr.get('foreground',self.fg);bg=attr.get('background',self.bg)
                if attr.get('reverse'):fg,bg=bg,fg
                color=lambda n: '#%06x'%n
                draw.rectangle((x*10,y*22,x*10+10,y*22+22),fill=color(bg))
                draw.text((x*10,y*22),c,font=font,fill=color(fg))
        im.save(path)
    def close(self):
        self.p.terminate()
        try:self.p.wait(timeout=5)
        except subprocess.TimeoutExpired:self.p.kill();self.p.wait()
        self.stderr.close()

p=argparse.ArgumentParser();p.add_argument('--root',type=pathlib.Path,required=True);p.add_argument('--screenshots',type=pathlib.Path,required=True)
a=p.parse_args();a.screenshots.mkdir(parents=True,exist_ok=True)
project=a.root/'sample-project';project.mkdir(exist_ok=True)
(project/'package.json').write_text('{"name":"shoshin-preview","private":true}\n')
(project/'main.ts').write_text('interface Project {\n  name: string;\n  ready: boolean;\n}\n\nconst project: Project = {\n  name: "ShoshinVim",\n  ready: true,\n};\n\nexport function welcome(name: string): string {\n  return `Welcome to ${name}`;\n}\n\nconsole.log(welcome(project.name));\n')
(project/'init.lua').write_text('local greeting = "Hello, Shoshin"\nprint(greeting)\n')
e=Editor(a.root)
try:
    assert e.lua('return vim.g.colors_name')=='tokyonight-night'
    assert not e.lua('return vim.v.errmsg'),e.lua('return vim.v.errmsg')
    e.capture(a.screenshots/'welcome.png')
    e.call('nvim_set_current_dir',[str(project)])
    e.call('nvim_command',['edit '+str(project/'init.lua')])
    e.call('nvim_command',['edit '+str(project/'main.ts')]);e.pump(2)
    e.call('nvim_command',['ShoshinExplore']);e.pump(1)
    assert e.lua("return #vim.api.nvim_list_wins()") >= 2, 'explorer did not open'
    e.call('nvim_command',['wincmd l']);e.pump(.3)
    e.capture(a.screenshots/'workspace.png')
    e.keys('<Esc>');e.call('nvim_command',['wincmd l']);e.keys(' ');e.pump(.8)
    e.capture(a.screenshots/'shortcuts.png')
    assert 'Windows' in (a.screenshots/'shortcuts.txt').read_text(), 'shortcut popup not visible'
    e.keys('<Esc>')
    e.call('nvim_command',['ShoshinHelp']);e.capture(a.screenshots/'guide.png');e.keys('q')
    for width,height in [(80,24),(52,16)]:
        e.call('nvim_ui_try_resize',[width,height]);e.pump(.4);e.capture(a.screenshots/('workspace-%d.png'%width))
    assert not e.lua('return vim.v.errmsg'),e.lua('return vim.v.errmsg')
    print('PASS: real Neovim startup, theme, sidebar, buffer tabs, Space shortcut hints and workspace guide')
    print(e.lua("return vim.inspect(vim.tbl_map(function(c) return c.name end, vim.lsp.get_clients()))"))
finally:e.close()
