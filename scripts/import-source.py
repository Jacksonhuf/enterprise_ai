#!/usr/bin/env python3
"""Hydrate pinned upstream source into functional paths without overwriting product files.

No Python runtime is needed by the application. This is a developer source-import utility.
"""
import shutil
import subprocess
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SOURCE = 'https://github.com/Jacksonhuf/WeKnora.git'
REVISION = 'cbdcebabf48c5ee2d6868b370326aa8341703f62'
STAMP = ROOT / '.source-revision'

def copy_missing(source: Path, target: Path):
    if source.is_dir():
        target.mkdir(parents=True, exist_ok=True)
        for child in source.iterdir():
            if child.name not in {'.git', 'node_modules', 'dist'}:
                copy_missing(child, target / child.name)
    elif not target.exists():
        target.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(source, target)

def main():
    if STAMP.exists():
        if STAMP.read_text().strip() != REVISION:
            raise SystemExit('Source revision mismatch. Review migration before importing another revision.')
        if (ROOT/'backend/platform/internal/agent/act.go').exists() and (ROOT/'frontend/src/components/EmptyState.vue').exists() and (ROOT/'backend/platform/website-docs').is_dir():
            print('Pinned upstream source is already available.')
            return
    with tempfile.TemporaryDirectory(prefix='cognara-source-') as temp:
        checkout = Path(temp) / 'source'
        subprocess.run(['git', 'init', '-q', str(checkout)], check=True)
        subprocess.run(['git', '-C', str(checkout), 'remote', 'add', 'origin', SOURCE], check=True)
        subprocess.run(['git', '-C', str(checkout), 'fetch', '--depth=1', 'origin', REVISION], check=True)
        subprocess.run(['git', '-C', str(checkout), 'checkout', '--detach', '-q', 'FETCH_HEAD'], check=True)
        actual = subprocess.check_output(['git', '-C', str(checkout), 'rev-parse', 'HEAD'], text=True).strip()
        if actual != REVISION:
            raise SystemExit('Upstream source revision verification failed')
        copy_missing(checkout/'frontend', ROOT/'frontend')
        copy_missing(checkout/'frontend/src/main.ts', ROOT/'frontend/src/upstream-main.ts')
        backend=ROOT/'backend/platform'
        for name in ['docs','website-docs','cmd','internal','config','migrations','third_party','licenses','testdata','client','go.mod','go.sum','LICENSE','THIRD_PARTY_NOTICES.md','VERSION']:
            copy_missing(checkout/name, backend/name)
        for name in ['client','proto']:
            copy_missing(checkout/'docreader'/name, backend/'docreader'/name)
    STAMP.write_text(REVISION+'\n')
    print('Source imported into frontend/ and backend/platform/. Product files were preserved.')

if __name__ == '__main__':
    main()
