#!/usr/bin/env python3
"""Import the original NEX Hero v1.1 bytes from ZIP into an existing Nexus checkout.
No download, regeneration, implicit git commit or push. Verifies everything first.
Usage: python scripts/import-nex-hybrid-package.py /path/nex_hybrid_github_assets_v1_1.zip
"""
import hashlib
import json
import os
from pathlib import Path
import sys
import zipfile

ROOT = Path(__file__).resolve().parents[1]
MANIFEST = ROOT / 'docs/design/nex/hybrid-v1/asset-manifest.json'

def main():
    if len(sys.argv) != 2:
        raise SystemExit('Uso: python scripts/import-nex-hybrid-package.py /caminho/nex_hybrid_github_assets_v1_1.zip')
    archive = Path(sys.argv[1]).resolve()
    spec = json.loads(MANIFEST.read_text(encoding='utf-8'))
    entries = [*spec['assets'], spec['generator']]
    if len(spec['assets']) != 14:
        raise RuntimeError('O manifesto deve conter exatamente 14 assets')
    verified = []
    with zipfile.ZipFile(archive) as z:
        for entry in entries:
            rel = entry.get('path') or entry.get('file')
            if not isinstance(rel, str) or rel.startswith('/') or '..' in Path(rel).parts:
                raise RuntimeError('Caminho inseguro no manifesto: ' + str(rel))
            path = (ROOT / rel).resolve()
            if not path.is_relative_to(ROOT):
                raise RuntimeError('Caminho fora do repositório: ' + rel)
            try:
                info = z.getinfo(rel)
                data = z.read(info)
            except KeyError:
                raise RuntimeError('Arquivo ausente no ZIP: ' + rel) from None
            sha = hashlib.sha256(data).hexdigest()
            if sha != entry['sha256'] or len(data) != entry['bytes']:
                raise RuntimeError('SHA-256/tamanho incorreto: ' + rel)
            if rel.endswith('.png') and data[:8] != b'\x89PNG\r\n\x1a\n':
                raise RuntimeError('Assinatura PNG inválida: ' + rel)
            if rel.endswith('.ora') and data[:2] != b'PK':
                raise RuntimeError('Assinatura ORA inválida: ' + rel)
            verified.append((path, data))
    # Only after all hashes are correct do we write any file.
    for path, data in verified:
        path.parent.mkdir(parents=True, exist_ok=True)
        temp = path.with_suffix(path.suffix + '.neximport')
        temp.write_bytes(data)
        os.replace(temp, path)
    for item in spec['assets']:
        item['status'] = 'VERSIONED'
    spec['status'] = 'ASSETS_VERSIONED_PENDING_VISUAL_APPROVAL'
    MANIFEST.write_text(json.dumps(spec, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')
    print(f'PASS: {len(verified)} arquivos com SHA-256 validado importados ao clone.')
    print('Próximos comandos no checkout:')
    print('  node scripts/verify-nex-hybrid-assets.mjs --require-binaries')
    print('  git add web/public/nex/hybrid-v1 docs/design/nex/hybrid-v1')
    print('  git commit -m "assets(nex): import source-faithful Hero v1.1 layers"')
    print('  git push')
    print('Importação não aprova automaticamente a arte visual nem publica no Nexus-Site.')

if __name__ == '__main__':
    main()
