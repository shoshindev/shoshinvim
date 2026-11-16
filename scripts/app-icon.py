#!/usr/bin/env python3
"""Draw the ShoshinVim app icon and export it for the web, iPhone and macOS.

The icon is an ensō, the Zen brush circle, around a block cursor, in the
interface's sakura pink and crystal blue. It comes in a light and a dark
version, drawn as SVG and exported as PNG and .icns.

Exporting needs macOS with Google Chrome (to turn SVG into PNG), sips and
iconutil. Run from the repository root:

    python3 scripts/app-icon.py            # writes docs/assets/icon/
    python3 scripts/app-icon.py --svg-only # SVG files only, no Chrome needed
"""

import argparse
import math
import os
import shutil
import subprocess
import sys
import tempfile
import time

CHROME = "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
OUT = os.path.join("docs", "assets", "icon")

THEMES = {
    "light": dict(bg0="#FBF8F1", bg1="#ECE6D6", pink="#B35B79", blue="#4D699B",
                  sheen=0.55, rim_top="#FFFFFF", rim_top_a=0.95, rim_bot="#000000", rim_bot_a=0.10, shadow=0.22),
    "dark": dict(bg0="#2C2C3A", bg1="#15151C", pink="#D27E99", blue="#7E9CD8",
                 sheen=0.10, rim_top="#FFFFFF", rim_top_a=0.30, rim_bot="#FFFFFF", rim_bot_a=0.06, shadow=0.45),
}

# A terminal cursor covers one character cell. In SF Mono, the macOS Terminal
# font, a cell is 0.618 em wide and 1.178 em tall.
CELL_RATIO = 0.618 / 1.178


def num(v):
    return f"{v:.2f}".rstrip("0").rstrip(".")


def mix(a, b, t):
    ca = [int(a[i:i + 2], 16) for i in (1, 3, 5)]
    cb = [int(b[i:i + 2], 16) for i in (1, 3, 5)]
    return "#" + "".join(f"{round(x + (y - x) * t):02X}" for x, y in zip(ca, cb))


def squircle(x, y, size):
    """Apple's continuous-corner rounded square: corner radius 22.37% of the
    side with 60% corner smoothing, built the way Figma builds smooth corners."""
    r = size * 0.2237
    k = 0.6
    rad = math.radians
    p = (1 + k) * r
    arc = 90 * (1 - k)
    ln = math.sin(rad(arc / 2)) * r * math.sqrt(2)
    alpha = (90 - arc) / 2
    p34 = r * math.tan(rad(alpha / 2))
    beta = 45 * k
    c = p34 * math.cos(rad(beta))
    d = c * math.tan(rad(beta))
    b = (p - ln - c - d) / 3
    a = 2 * b
    n = num
    return (
        f"M{n(x + size - p)} {n(y)}"
        f"c{n(a)} 0 {n(a + b)} 0 {n(a + b + c)} {n(d)}a{n(r)} {n(r)} 0 0 1 {n(ln)} {n(ln)}"
        f"c{n(d)} {n(c)} {n(d)} {n(b + c)} {n(d)} {n(a + b + c)}"
        f"L{n(x + size)} {n(y + size - p)}"
        f"c0 {n(a)} 0 {n(a + b)} {n(-d)} {n(a + b + c)}a{n(r)} {n(r)} 0 0 1 {n(-ln)} {n(ln)}"
        f"c{n(-c)} {n(d)} {n(-(b + c))} {n(d)} {n(-(a + b + c))} {n(d)}"
        f"L{n(x + p)} {n(y + size)}"
        f"c{n(-a)} 0 {n(-(a + b))} 0 {n(-(a + b + c))} {n(-d)}a{n(r)} {n(r)} 0 0 1 {n(-ln)} {n(-ln)}"
        f"c{n(-d)} {n(-c)} {n(-d)} {n(-(b + c))} {n(-d)} {n(-(a + b + c))}"
        f"L{n(x)} {n(y + p)}"
        f"c0 {n(-a)} 0 {n(-(a + b))} {n(d)} {n(-(a + b + c))}a{n(r)} {n(r)} 0 0 1 {n(ln)} {n(-ln)}"
        f"c{n(c)} {n(-d)} {n(b + c)} {n(-d)} {n(a + b + c)} {n(-d)}Z"
    )


def smoothstep(e0, e1, x):
    t = min(1.0, max(0.0, (x - e0) / (e1 - e0)))
    return t * t * (3 - 2 * t)


def enso(small, cursor_h):
    """The brush circle and the cursor in a 1024 square. `small` is a simpler,
    heavier version for 16 and 32 point sizes, where brush detail is lost."""
    cx, cy = 512, 516
    R = 286 if not small else 282
    wmax = 128 if not small else 178
    th0 = math.radians(128)
    sweep = math.radians(334 if not small else 322)

    def radius(t):
        return R * (1 + 0.022 * math.sin(t * 2 * math.pi + 1.1))

    def width(t):
        return (wmax * (0.86 + 0.14 * smoothstep(0, 0.07, t)) * (1 - 0.32 * t)
                * (1 + 0.10 * math.exp(-((t - 0.56) / 0.10) ** 2)))

    def point(t, off):
        th = th0 + sweep * t
        r = radius(t) + off
        return cx + r * math.cos(th), cy + r * math.sin(th)

    def band(t0, t1, center, half):
        n = max(8, round((t1 - t0) * 300))
        outer, inner = [], []
        for i in range(n + 1):
            t = t0 + (t1 - t0) * i / n
            m, h = center(t), half(t)
            outer.append(point(t, m + h))
            inner.append(point(t, m - h))
        pts = outer + inner[::-1]
        return "".join(("L" if i else "M") + f"{x:.1f} {y:.1f}" for i, (x, y) in enumerate(pts)) + "Z"

    if small:
        # One solid stroke that tapers to a point.
        body = band(0, 1, lambda t: width(t) * 0.04, lambda t: width(t) / 2 * (1 - 0.9 * smoothstep(0.55, 1, t)))
    else:
        # A solid body that thins out, then dry-brush bristles that carry the tail.
        body = band(0, 0.84, lambda t: width(t) * 0.04, lambda t: width(t) / 2 * (1 - smoothstep(0.70, 0.84, t)))
        for o, fw, end in [(-0.36, 0.16, 0.93), (-0.14, 0.20, 1.0), (0.10, 0.17, 0.96), (0.32, 0.14, 0.89)]:
            body += band(0.66, end,
                         lambda t, o=o: width(t) * (o + 0.04),
                         lambda t, fw=fw, end=end: width(t) * fw / 2 * (1 - smoothstep(end - 0.14, end, t)))
    sx, sy = point(0, width(0) * 0.04)
    w0 = width(0)
    ch = cursor_h if not small else cursor_h * 1.25
    cw = ch * CELL_RATIO
    return dict(cx=cx, cy=cy, R=R, body=body, start=(sx, sy, w0 * 0.5, w0 * 0.47),
                cursor=(cx - cw / 2, cy - ch / 2, cw, ch))


def artwork(theme, uid, small=False, cursor_h=186):
    """Background and mark, full bleed in a 1024 square, plus the defs they use."""
    T = THEMES[theme]
    e = enso(small, cursor_h)
    sx, sy, rx, ry = e["start"]
    x, y, w, h = e["cursor"]
    defs = f"""
    <linearGradient id="{uid}bg" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="{T['bg0']}"/><stop offset="1" stop-color="{T['bg1']}"/></linearGradient>
    <radialGradient id="{uid}sh" cx=".32" cy="0" r="1"><stop offset="0" stop-color="#FFFFFF" stop-opacity="{T['sheen']}"/><stop offset=".7" stop-color="#FFFFFF" stop-opacity="0"/></radialGradient>
    <linearGradient id="{uid}ink" x1="0" y1="{e['cy'] - e['R'] - 70}" x2="0" y2="{e['cy'] + e['R'] + 70}" gradientUnits="userSpaceOnUse"><stop offset="0" stop-color="{T['pink']}"/><stop offset=".55" stop-color="{mix(T['pink'], T['blue'], .5)}"/><stop offset="1" stop-color="{T['blue']}"/></linearGradient>
    <linearGradient id="{uid}cur" x1="0" y1="{y:.1f}" x2="0" y2="{y + h:.1f}" gradientUnits="userSpaceOnUse"><stop offset="0" stop-color="{mix(T['blue'], '#FFFFFF', .14)}"/><stop offset="1" stop-color="{mix(T['blue'], '#000000', .06)}"/></linearGradient>
    <filter id="{uid}ds" x="-20%" y="-20%" width="140%" height="150%"><feDropShadow dx="0" dy="16" stdDeviation="20" flood-color="#000000" flood-opacity="{T['shadow']}"/></filter>"""
    body = f"""
    <rect width="1024" height="1024" fill="url(#{uid}bg)"/>
    <rect width="1024" height="1024" fill="url(#{uid}sh)"/>
    <g filter="url(#{uid}ds)">
      <path d="{e['body']}" fill="url(#{uid}ink)"/>
      <ellipse cx="{sx:.1f}" cy="{sy:.1f}" rx="{rx:.1f}" ry="{ry:.1f}" fill="url(#{uid}ink)"/>
      <rect x="{x:.1f}" y="{y:.1f}" width="{w:.1f}" height="{h:.1f}" rx="{max(3, w * 0.05):.1f}" fill="url(#{uid}cur)"/>
    </g>"""
    return defs, body


def rim(theme, uid, shape):
    T = THEMES[theme]
    defs = f"""
    <linearGradient id="{uid}rim" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="{T['rim_top']}" stop-opacity="{T['rim_top_a']}"/><stop offset=".3" stop-color="{T['rim_top']}" stop-opacity="{T['rim_top_a'] * .15:.3f}"/><stop offset=".8" stop-color="{T['rim_bot']}" stop-opacity="0"/><stop offset="1" stop-color="{T['rim_bot']}" stop-opacity="{T['rim_bot_a']}"/></linearGradient>"""
    return defs, f'<path d="{shape}" fill="none" stroke="url(#{uid}rim)" stroke-width="12"/>'


def svg(inner, defs):
    return (f'<svg xmlns="http://www.w3.org/2000/svg" width="1024" height="1024" viewBox="0 0 1024 1024">\n'
            f"<defs>{defs}\n</defs>\n{inner}\n</svg>\n")


def rounded(theme, small=False, cursor_h=186):
    """The icon as people see it on a Home Screen: rounded, transparent corners."""
    uid = "s" if small else "i"
    shape = squircle(0, 0, 1024)
    d1, art = artwork(theme, uid, small, cursor_h)
    d2, edge = rim(theme, uid, shape)
    return svg(f'<g clip-path="url(#{uid}clip)">{art}\n{edge}\n</g>',
               f'<clipPath id="{uid}clip"><path d="{shape}"/></clipPath>{d1}{d2}')


def ios(theme, cursor_h=186):
    """The App Store master: a full square with no transparency. iOS rounds the corners itself."""
    d1, art = artwork(theme, "i", False, cursor_h)
    return svg(art, d1)


def macos(theme, small=False, cursor_h=186):
    """macOS layout: an 824 point shape centered on a 1024 canvas, with a drop shadow."""
    uid = "s" if small else "i"
    inset, size = 100, 824
    shape = squircle(inset, inset, size)
    d1, art = artwork(theme, uid, small, cursor_h)
    d2, edge = rim(theme, uid, squircle(0, 0, 1024))
    scale = size / 1024
    defs = (f'<clipPath id="{uid}clip"><path d="{squircle(0, 0, 1024)}"/></clipPath>{d1}{d2}'
            f'<filter id="{uid}drop" x="-10%" y="-10%" width="120%" height="125%"><feDropShadow dx="0" dy="12" stdDeviation="12" flood-color="#000000" flood-opacity=".30"/></filter>')
    inner = (f'<path d="{shape}" fill="{THEMES[theme]["bg1"]}" filter="url(#{uid}drop)"/>\n'
             f'<g transform="translate({inset} {inset}) scale({scale})"><g clip-path="url(#{uid}clip)">{art}\n{edge}\n</g></g>')
    return svg(inner, defs)


def render(svg_path, png_path, size, transparent=True):
    """Draw an SVG to a PNG with headless Chrome. With transparent=False the
    PNG has no alpha channel, which the App Store requires."""
    with tempfile.TemporaryDirectory() as tmp:
        page = os.path.join(tmp, "page.html")
        with open(page, "w") as f:
            f.write(f'<html><body style="margin:0;background:transparent">'
                    f'<img src="file://{os.path.abspath(svg_path)}" width="{size}" height="{size}" style="display:block"></body></html>')
        shot = os.path.join(tmp, "shot.png")
        proc = subprocess.Popen(
            [CHROME, "--headless", "--hide-scrollbars", "--force-device-scale-factor=1",
             *(["--default-background-color=00000000"] if transparent else []), "--allow-file-access-from-files",
             f"--user-data-dir={os.path.join(tmp, 'profile')}",
             f"--window-size={size},{size}", f"--screenshot={shot}", f"file://{page}"],
            stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        # Headless Chrome can keep running after it writes the file.
        deadline = time.time() + 30
        last = -1
        while time.time() < deadline:
            if os.path.exists(shot):
                cur = os.path.getsize(shot)
                if cur > 0 and cur == last:
                    break
                last = cur
            time.sleep(0.3)
        proc.kill()
        proc.wait()
        if not os.path.exists(shot):
            sys.exit(f"Chrome did not render {svg_path}")
        shutil.move(shot, png_path)


def main():
    ap = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    ap.add_argument("--out", default=OUT, help=f"output folder (default {OUT})")
    ap.add_argument("--svg-only", action="store_true", help="write the SVG files and stop")
    ap.add_argument("--cursor", type=float, default=186, help="cursor height in a 1024 icon (default 186)")
    args = ap.parse_args()
    os.makedirs(args.out, exist_ok=True)
    out = lambda name: os.path.join(args.out, name)

    for theme in THEMES:
        with open(out(f"icon-{theme}.svg"), "w") as f:
            f.write(rounded(theme, cursor_h=args.cursor))
        with open(out(f"icon-{theme}-small.svg"), "w") as f:
            f.write(rounded(theme, small=True, cursor_h=args.cursor))
    if args.svg_only:
        return
    if not os.path.exists(CHROME):
        sys.exit("Google Chrome is needed to export PNG files")

    with tempfile.TemporaryDirectory() as tmp:
        for theme in THEMES:
            render(out(f"icon-{theme}.svg"), out(f"icon-{theme}.png"), 1024)

            src = os.path.join(tmp, f"ios-{theme}.svg")
            with open(src, "w") as f:
                f.write(ios(theme, cursor_h=args.cursor))
            render(src, out(f"ios-{theme}.png"), 1024, transparent=False)

            # A macOS .icns holds each size at 1x and 2x. Sizes up to 32 points
            # use the simpler artwork.
            iconset = os.path.join(tmp, f"{theme}.iconset")
            os.makedirs(iconset)
            for small in (False, True):
                src = os.path.join(tmp, f"macos-{theme}-{small}.svg")
                with open(src, "w") as f:
                    f.write(macos(theme, small=small, cursor_h=args.cursor))
                for pt in ([16, 32] if small else [128, 256, 512]):
                    for scale in (1, 2):
                        name = f"icon_{pt}x{pt}{'@2x' if scale == 2 else ''}.png"
                        render(src, os.path.join(iconset, name), pt * scale)
            subprocess.run(["iconutil", "-c", "icns", iconset, "-o", out(f"macos-{theme}.icns")], check=True)
    print(f"Wrote the icon files to {args.out}")


if __name__ == "__main__":
    main()
