#!/usr/bin/env python3
"""Real Neovim UI smoke test. Requires msgpack; screenshots also need Pillow and fontTools.
Run only against an isolated, already-installed configuration with Tokyo Night and a
TypeScript language server, such as one accepting every recommendation:
  python3 scripts/editor-smoke.py --root /tmp/shoshinvim-test --screenshots /tmp/screens
The root holds config/, data/, state/ and cache/; HOME is root/home, so paths show as ~.
Screenshots of a configuration with Nerd Font icons need a Nerd Font: pass --font.
The README's images are workspace.png (docs/assets/rich-ide.png) and shortcuts.png
(docs/assets/shortcut-guide.png).
"""
import argparse, os, pathlib, select, subprocess, time
import msgpack

class Editor:
    def __init__(self, root, font):
        self.font=font
        env = os.environ.copy()
        for key, directory in [('XDG_CONFIG_HOME','config'),('XDG_DATA_HOME','data'),('XDG_STATE_HOME','state'),('XDG_CACHE_HOME','cache')]:
            env[key] = str(root / directory)
        env['NVIM_APPNAME'] = 'nvim'
        env['HOME'] = str(root / 'home')  # Paths on screen read ~/..., not this machine's.
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
    # Like a terminal, characters the font lacks come from fallback fonts.
    def font_for(self,c):
        if not hasattr(self,'fonts'):
            from PIL import ImageFont
            from fontTools.ttLib import TTFont,TTCollection
            self.fonts=[]
            for path in [self.font,'/System/Library/Fonts/Menlo.ttc','/System/Library/Fonts/Apple Symbols.ttf','/System/Library/Fonts/Supplemental/Arial Unicode.ttf']:
                if os.path.exists(path):
                    tt=TTCollection(path).fonts[0] if path.endswith('.ttc') else TTFont(path)
                    self.fonts.append((ImageFont.truetype(path,16),set(tt.getBestCmap())))
        return next((f for f,cmap in self.fonts if ord(c) in cmap),self.fonts[0][0])
    # Like a terminal, box-drawing and left-block characters fill their cell, so borders join.
    LINES={'─':'lr','│':'ud','┌':'rd','┐':'ld','└':'ru','┘':'lu','├':'udr','┤':'udl','┬':'lrd','┴':'lru','┼':'lrud','╭':'rd','╮':'ld','╰':'ru','╯':'lu'}
    ARCS={'╭':(0,0,180,270),'╮':(-1,0,270,360),'╰':(0,-1,90,180),'╯':(-1,-1,0,90)}
    BLOCKS={'▏':1,'▎':2,'▍':3,'▌':4,'▋':5,'▊':6,'▉':7,'█':8}
    def cell(self,draw,x,y,c,fg,w=10,h=22):
        cx,cy=x+w//2,y+h//2;r=4 if c in self.ARCS else 0
        if c in self.ARCS:
            sx,sy,a0,a1=self.ARCS[c];ox=cx if sx==0 else cx-2*r;oy=cy if sy==0 else cy-2*r
            draw.arc((ox,oy,ox+2*r,oy+2*r),a0,a1,fill=fg)
        if c in self.LINES:
            for side,line in {'l':(x,cy,cx-r,cy),'r':(cx+r,cy,x+w,cy),'u':(cx,y,cx,cy-r),'d':(cx,cy+r,cx,y+h)}.items():
                if side in self.LINES[c]:draw.line(line,fill=fg)
        elif c in self.BLOCKS:draw.rectangle((x,y,x+max(1,w*self.BLOCKS[c]//8)-1,y+h-1),fill=fg)
        else:draw.text((x,y),c,font=self.font_for(c),fill=fg)
    def capture(self,path):
        self.call('nvim_command',['redraw!']);self.pump(.2)
        path.with_suffix('.txt').write_text('\n'.join(''.join(c for c,h in row) for row in self.grid))
        from PIL import Image,ImageDraw
        im=Image.new('RGB',(len(self.grid[0])*10,len(self.grid)*22),self.bg);draw=ImageDraw.Draw(im)
        color=lambda n: '#%06x'%n
        for y,row in enumerate(self.grid):
            for x,(c,h) in enumerate(row):
                attr=self.highlights.get(h,{});fg=attr.get('foreground',self.fg);bg=attr.get('background',self.bg)
                if attr.get('reverse'):fg,bg=bg,fg
                draw.rectangle((x*10,y*22,x*10+9,y*22+21),fill=color(bg))
                if c.strip():self.cell(draw,x*10,y*22,c,color(fg))
        im.save(path)
    def close(self):
        self.p.terminate()
        try:self.p.wait(timeout=5)
        except subprocess.TimeoutExpired:self.p.kill();self.p.wait()
        self.stderr.close()

p=argparse.ArgumentParser();p.add_argument('--root',type=pathlib.Path,required=True);p.add_argument('--screenshots',type=pathlib.Path,required=True)
p.add_argument('--font',default='/System/Library/Fonts/Menlo.ttc',help='monospace font file for the screenshots')
a=p.parse_args();a.screenshots.mkdir(parents=True,exist_ok=True)
project=a.root/'home'/'sample-project';project.mkdir(parents=True,exist_ok=True)
(project/'package.json').write_text('{"name":"shoshin-preview","private":true}\n')
# Line 15 has a type error, so the workspace shows the language server's diagnostics.
(project/'main.ts').write_text('interface Project {\n  name: string;\n  ready: boolean;\n}\n\nconst project: Project = {\n  name: "ShoshinVim",\n  ready: true,\n};\n\nexport function welcome(name: string): string {\n  return `Welcome to ${name}`;\n}\n\nconst count: number = "three";\nconsole.log(welcome(project.name));\n')
(project/'init.lua').write_text('local greeting = "Hello, Shoshin"\nprint(greeting)\n')
e=Editor(a.root,a.font)
try:
    assert e.lua('return vim.g.colors_name')=='tokyonight-night'
    assert not e.lua('return vim.v.errmsg'),e.lua('return vim.v.errmsg')
    e.capture(a.screenshots/'welcome.png')
    e.call('nvim_set_current_dir',[str(project)])
    e.call('nvim_command',['edit '+str(project/'init.lua')])
    e.call('nvim_command',['edit '+str(project/'main.ts')]);e.pump(2)
    e.call('nvim_command',['ShoshinExplore']);e.pump(1)
    assert e.lua("return #vim.api.nvim_list_wins()") >= 2, 'explorer did not open'
    e.call('nvim_command',['wincmd l']);e.pump(.3);e.call('nvim_win_set_cursor',[0,[15,6]])
    end=time.monotonic()+20
    while not e.lua("return #vim.diagnostic.get(0, { severity = vim.diagnostic.severity.ERROR })") and time.monotonic()<end:e.pump(.5)
    assert e.lua("return #vim.diagnostic.get(0, { severity = vim.diagnostic.severity.ERROR })"), 'the language server reported no type error'
    e.pump(1);e.capture(a.screenshots/'workspace.png')
    e.keys('<Esc>');e.call('nvim_command',['wincmd l']);e.keys(' ');e.pump(.8)
    e.capture(a.screenshots/'shortcuts.png')
    assert 'close' in (a.screenshots/'shortcuts.txt').read_text(), 'shortcut popup not visible'
    e.keys('<Esc>')
    e.call('nvim_command',['ShoshinHelp']);e.capture(a.screenshots/'guide.png');e.keys('q')
    for width,height in [(80,24),(52,16)]:
        e.call('nvim_ui_try_resize',[width,height]);e.pump(.4);e.capture(a.screenshots/('workspace-%d.png'%width))
    assert not e.lua('return vim.v.errmsg'),e.lua('return vim.v.errmsg')
    print('PASS: real Neovim startup, theme, sidebar, buffer tabs, diagnostics, Space shortcut hints and workspace guide')
    print(e.lua("return vim.inspect(vim.tbl_map(function(c) return c.name end, vim.lsp.get_clients()))"))
finally:e.close()
