"""Generate the terminal app icons. Requires Pillow: python -m pip install Pillow."""
from pathlib import Path
from PIL import Image, ImageDraw

root = Path(__file__).resolve().parents[1] / 'frontend/assets/icons'
image = Image.new('RGB', (1024, 1024), '#090a0f')
draw = ImageDraw.Draw(image)
draw.rounded_rectangle((120, 170, 904, 854), radius=72, fill='#0d1117', outline='#30363d', width=12)
draw.line((120, 310, 904, 310), fill='#30363d', width=12)
for x in (188, 236, 284):
    draw.ellipse((x, 226, x + 16, 242), fill='#8b949e')
draw.line((272, 448, 414, 576, 272, 704), fill='#3fb950', width=48, joint='curve')
draw.rounded_rectangle((508, 668, 752, 716), radius=12, fill='#3fb950')
for size in (64, 128, 192, 256, 512):
    image.resize((size, size), Image.Resampling.LANCZOS).save(root / f'icon-{size}.png')
for size in (16, 32, 48):
    image.resize((size, size), Image.Resampling.LANCZOS).save(root / f'favicon-{size}.png')
image.resize((180, 180), Image.Resampling.LANCZOS).save(root / 'apple-touch-icon.png')
image.resize((256, 256), Image.Resampling.LANCZOS).save(root / 'favicon.ico', sizes=[(16, 16), (32, 32), (48, 48), (64, 64), (128, 128), (256, 256)])
(root / 'favicon.svg').write_text('''<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32" role="img" aria-label="SlateSSH">
  <rect width="32" height="32" rx="6" fill="#090a0f"/>
  <rect x="3.75" y="5.3" width="24.5" height="21.4" rx="2.25" fill="#0d1117" stroke="#30363d" stroke-width=".5"/>
  <path d="M4 9.7h24" stroke="#30363d" stroke-width=".5"/>
  <path d="M8.5 14l4.5 4-4.5 4M16 21.6h7.5" fill="none" stroke="#3fb950" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
</svg>
''', encoding='utf-8')
