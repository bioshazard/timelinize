#!/usr/bin/env python3
"""Refuse reuse of the pinned runtime if backend/build dependencies have changed."""
import subprocess

BASE = 'v0.0.29-bios.4'
changed = subprocess.check_output(['git', 'diff', '--name-only', BASE, 'HEAD'], text=True).splitlines()
allowed = lambda p: (
    p.startswith(('frontend/', 'docs/'))
    or p in {'FORK.md', 'Dockerfile.frontend-release', 'tools/verify-frontend-release.py', '.github/workflows/docker-publish.yaml'}
    or (p.startswith('tlzapp/') and p.endswith('_test.go'))
)
rejected = [p for p in changed if not allowed(p)]
if rejected:
    raise SystemExit('Frontend-only runtime reuse refused: ' + ', '.join(rejected))
print(f'Runtime reuse guard passed against {BASE}: {len(changed)} scoped source files')
