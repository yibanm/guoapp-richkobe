"""从位图源图生成霏果鉴各平台图标资源。

源图为 scripts/source/brand_icon_source.png（1024x1024 线稿）。
尺寸表与 scripts/generate_brand_assets.py 保持一致，仅输入从 SVG 换成位图：
线稿自带底色，直接作前景层；monochrome 层用线稿二值化剪影。
需要 Pillow；中文字体默认取 Windows 微软雅黑，可用 --font 覆盖。
"""
import argparse
import io
import json
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont

root = Path(__file__).resolve().parents[1]
parser = argparse.ArgumentParser(description='从位图源图生成霏果鉴平台图标资源。')
parser.add_argument('--output', type=Path, default=root)
parser.add_argument('--source', type=Path,
                    default=root / 'scripts/source/brand_icon_source.png')
parser.add_argument('--font', type=Path,
                    default=Path('C:/Windows/Fonts/msyh.ttc'))
options = parser.parse_args()
output = options.output

source = Image.open(options.source).convert('RGBA')
if source.size != (1024, 1024):
    source = source.resize((1024, 1024), Image.Resampling.LANCZOS)

# monochrome 剪影：线稿深色像素转纯黑不透明，其余透明
gray = source.convert('L')
silhouette = Image.new('RGBA', (1024, 1024), (0, 0, 0, 0))
mask = gray.point(lambda value: 255 if value < 128 else 0)
silhouette.putalpha(mask)
silhouette_black = Image.new('RGBA', (1024, 1024), (0, 0, 0, 255))
silhouette_black.putalpha(mask)

icon = source.copy()
rounded = icon.copy()
corners = Image.new('L', (1024, 1024), 0)
ImageDraw.Draw(corners).rounded_rectangle((0, 0, 1023, 1023), radius=236, fill=255)
rounded.putalpha(corners)
icon = icon.convert('RGB')
adaptive = Image.new('RGBA', (1536, 1536))
adaptive.paste(source, (256, 256))
monochrome = Image.new('RGBA', (1536, 1536))
monochrome.paste(silhouette_black, (256, 256), silhouette_black)


def save(image, name):
    destination = output / name
    destination.parent.mkdir(parents=True, exist_ok=True)
    image.save(destination)


contents = json.loads(
    (root / 'ios/Runner/Assets.xcassets/AppIcon.appiconset/Contents.json')
    .read_text(encoding='utf-8'))
for entry in contents['images']:
    if 'filename' in entry:
        size = round(float(entry['size'].split('x')[0])
                     * float(entry['scale'].rstrip('x')))
        save(icon.resize((size, size), Image.Resampling.LANCZOS),
             'ios/Runner/Assets.xcassets/AppIcon.appiconset/' + entry['filename'])
save(rounded.resize((256, 256), Image.Resampling.LANCZOS),
     'windows/runner/resources/app_icon.ico')
macos_canvas = Image.new('RGBA', (1024, 1024))
macos_badge = rounded.resize((824, 824), Image.Resampling.LANCZOS)
macos_canvas.paste(macos_badge, (100, 100), macos_badge)
macos_contents = json.loads(
    (root / 'macos/Runner/Assets.xcassets/AppIcon.appiconset/Contents.json')
    .read_text(encoding='utf-8'))
for entry in macos_contents['images']:
    if 'filename' in entry:
        size = round(float(entry['size'].split('x')[0])
                     * float(entry['scale'].rstrip('x')))
        save(macos_canvas.resize((size, size), Image.Resampling.LANCZOS),
             'macos/Runner/Assets.xcassets/AppIcon.appiconset/' + entry['filename'])
for density, scale in [('mdpi', 1), ('hdpi', 1.5), ('xhdpi', 2),
                       ('xxhdpi', 3), ('xxxhdpi', 4)]:
    legacy = round(48 * scale)
    layer = round(108 * scale)
    save(rounded.resize((legacy, legacy), Image.Resampling.LANCZOS),
         f'android/app/src/main/res/mipmap-{density}/ic_launcher.png')
    save(adaptive.resize((layer, layer), Image.Resampling.LANCZOS),
         f'android/app/src/main/res/mipmap-{density}/ic_launcher_foreground.png')
    save(monochrome.resize((layer, layer), Image.Resampling.LANCZOS),
         f'android/app/src/main/res/mipmap-{density}/ic_launcher_monochrome.png')

font_path = options.font
if not font_path.exists():
    for candidate in ('C:/Windows/Fonts/msyh.ttc',
                      '/System/Library/Fonts/PingFang.ttc'):
        if Path(candidate).exists():
            font_path = Path(candidate)
            break
font = ImageFont.truetype(str(font_path), 76)
banner = Image.new('RGB', (640, 360), '#F5F9FC')
badge = rounded.resize((180, 180), Image.Resampling.LANCZOS)
banner.paste(badge, (44, 90), badge)
draw = ImageDraw.Draw(banner)
draw.text((255, 128), '霏果鉴', font=font, fill='#3B4552')
save(banner, 'android/app/src/main/res/drawable-xhdpi/tv_banner.png')
save(banner, 'android/app/src/main/res/drawable-xhdpi/tv_banner_all_sources.png')
print('图标资源生成完成')
