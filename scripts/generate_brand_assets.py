#!/usr/bin/env python3
"""
Generate complete IAPro Nexus brand assets from master source.
Produces:
  - Root: logo.png, nexus-logo.png, nexus-logo-dark.png, nexus-icon.png, favicon.ico
  - web/public: all icon sizes (16..512), apple-touch-icon, favicon.ico, dark/light logos, manifest
  - assets/brand: all master variations, cards (1200x630), previews
"""

import collections
import os
import sys
from pathlib import Path
from PIL import Image

def clean_extract(in_crop, bg_color=(254, 254, 254)):
    img = in_crop.convert('RGBA')
    w, h = img.size
    pixels = img.load()

    visited = bytearray(w * h)
    queue = collections.deque()

    # Seed border pixels that are whitish
    for x in range(w):
        for y in (0, h - 1):
            r, g, b, _ = pixels[x, y]
            if r >= 230 and g >= 230 and b >= 230:
                visited[y * w + x] = 1
                queue.append((x, y))
    for y in range(h):
        for x in (0, w - 1):
            r, g, b, _ = pixels[x, y]
            if r >= 230 and g >= 230 and b >= 230:
                if not visited[y * w + x]:
                    visited[y * w + x] = 1
                    queue.append((x, y))

    while queue:
        cx, cy = queue.popleft()
        for dx, dy in [(-1, 0), (1, 0), (0, -1), (0, 1)]:
            nx, ny = cx + dx, cy + dy
            if 0 <= nx < w and 0 <= ny < h:
                idx = ny * w + nx
                if not visited[idx]:
                    r, g, b, _ = pixels[nx, ny]
                    if r >= 225 and g >= 225 and b >= 225:
                        visited[idx] = 1
                        queue.append((nx, ny))

    out = Image.new('RGBA', (w, h), (0, 0, 0, 0))
    out_pixels = out.load()
    bg_r, bg_g, bg_b = bg_color

    for y in range(h):
        for x in range(w):
            idx = y * w + x
            if visited[idx]:
                continue
            r, g, b, _ = pixels[x, y]

            is_adjacent = False
            for dx in (-1, 0, 1):
                for dy in (-1, 0, 1):
                    nx, ny = x + dx, y + dy
                    if 0 <= nx < w and 0 <= ny < h and visited[ny * w + nx]:
                        is_adjacent = True
                        break
                if is_adjacent:
                    break

            if is_adjacent:
                min_c = min(r, g, b)
                max_diff = bg_r - min_c
                if max_diff < 5:
                    continue
                alpha = min(1.0, max_diff / 130.0)
                if alpha < 0.05:
                    continue
                fr = int(max(0, min(255, (r - bg_r * (1.0 - alpha)) / alpha)))
                fg = int(max(0, min(255, (g - bg_g * (1.0 - alpha)) / alpha)))
                fb = int(max(0, min(255, (b - bg_b * (1.0 - alpha)) / alpha)))
                out_pixels[x, y] = (fr, fg, fb, int(alpha * 255))
            else:
                out_pixels[x, y] = (r, g, b, 255)

    return out

def generate_dark_variant(full_img):
    w, h = full_img.size
    pixels = full_img.load()
    dark_mode = full_img.copy()
    dm_pixels = dark_mode.load()

    # The text region starts at x >= 335
    for x in range(335, w):
        for y in range(h):
            r, g, b, a = pixels[x, y]
            if a > 10:
                # Keep vibrant blue accent on letter X
                is_blue_slash = (b > 130 and b > r + 45 and b > g + 10)
                if not is_blue_slash:
                    dm_pixels[x, y] = (248, 250, 252, a)
    return dark_mode

def make_square_icon(icon_img, target_size=512):
    bbox = icon_img.getbbox()
    if bbox:
        icon_img = icon_img.crop(bbox)
    w, h = icon_img.size
    # Fill target square canvas with high density (98% fill for crisp anti-aliasing edge)
    scale = (target_size * 0.98) / max(w, h)
    new_w = max(1, int(w * scale))
    new_h = max(1, int(h * scale))
    resized = icon_img.resize((new_w, new_h), Image.Resampling.LANCZOS)

    sq = Image.new('RGBA', (target_size, target_size), (0, 0, 0, 0))
    offset_x = (target_size - new_w) // 2
    offset_y = (target_size - new_h) // 2
    sq.paste(resized, (offset_x, offset_y), resized)
    return sq

def make_og_card(logo_img, bg_color=(8, 10, 15, 255), width=1200, height=630):
    card = Image.new('RGBA', (width, height), bg_color)
    lw, lh = logo_img.size
    target_lw = 720
    scale = target_lw / lw
    target_lh = int(lh * scale)
    scaled_logo = logo_img.resize((target_lw, target_lh), Image.Resampling.LANCZOS)
    ox = (width - target_lw) // 2
    oy = (height - target_lh) // 2
    card.alpha_composite(scaled_logo, (ox, oy))
    return card

def main():
    root = Path(__file__).resolve().parent.parent
    source_path = root / 'assets/brand/source-logo.png'
    if not source_path.exists():
        source_path = Path('/tmp/aionui/general/image-1.png')
    
    if not source_path.exists():
        print(f"Error: source not found at {source_path}", file=sys.stderr)
        sys.exit(1)

    print(f"Loading master source: {source_path}")
    master = Image.open(source_path)

    # 1. Precise crops
    print("Extracting clean transparent icon and full logo...")
    icon_crop = master.crop((104, 492, 438, 728))
    full_crop = master.crop((104, 492, 1154, 728))

    icon_clean = clean_extract(icon_crop)
    full_clean = clean_extract(full_crop)
    full_bbox = full_clean.getbbox()
    if full_bbox:
        full_clean = full_clean.crop(full_bbox)
    full_dark = generate_dark_variant(full_clean)

    # 2. Square icon
    print("Building square 512x512 icon...")
    icon_512 = make_square_icon(icon_clean, 512)

    # Dirs
    web_public = root / 'web/public'
    assets_brand = root / 'assets/brand'
    web_public.mkdir(parents=True, exist_ok=True)
    assets_brand.mkdir(parents=True, exist_ok=True)

    # 3. Save core root files
    print("Saving root brand files...")
    full_clean.save(root / 'nexus-logo.png')
    full_dark.save(root / 'nexus-logo-dark.png')
    icon_512.save(root / 'nexus-icon.png')
    full_clean.save(root / 'logo.png')  # Canonical repo logo

    # 4. Save to assets/brand
    print("Saving assets/brand collection...")
    full_clean.save(assets_brand / 'nexus-logo.png')
    full_dark.save(assets_brand / 'nexus-logo-dark.png')
    icon_512.save(assets_brand / 'nexus-icon.png')
    full_clean.save(assets_brand / 'logo.png')

    # Social cards
    card_dark = make_og_card(full_dark, bg_color=(8, 10, 15, 255))
    card_dark.save(assets_brand / 'nexus-social-card-dark.png')
    card_light = make_og_card(full_clean, bg_color=(255, 255, 255, 255))
    card_light.save(assets_brand / 'nexus-social-card-light.png')

    # 5. Save web/public assets
    print("Saving web/public icons and assets...")
    full_clean.save(web_public / 'nexus-logo.png')
    full_dark.save(web_public / 'nexus-logo-dark.png')
    full_clean.save(web_public / 'logo.png')
    icon_512.save(web_public / 'nexus-icon.png')
    icon_512.save(web_public / 'nexus-icon-512.png')

    icon_sizes = [256, 192, 180, 128, 64, 48, 32, 16]
    for s in icon_sizes:
        resized = icon_512.resize((s, s), Image.Resampling.LANCZOS)
        resized.save(web_public / f'nexus-icon-{s}.png')
        if s == 180:
            resized.save(web_public / 'apple-touch-icon.png')

    # Favicon ICO
    ico_path = web_public / 'favicon.ico'
    icon_512.save(ico_path, format='ICO', sizes=[(16, 16), (32, 32), (48, 48), (64, 64)])
    icon_512.save(root / 'favicon.ico', format='ICO', sizes=[(16, 16), (32, 32), (48, 48), (64, 64)])

    print("Brand assets generated successfully!")

if __name__ == '__main__':
    main()
