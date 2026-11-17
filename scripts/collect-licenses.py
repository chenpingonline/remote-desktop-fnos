#!/usr/bin/env python3
import json,pathlib,shutil,subprocess
root=pathlib.Path(__file__).resolve().parent.parent
out=root/'packaging/remote-desktop/app/licenses';out.mkdir(parents=True,exist_ok=True)
for p in (root/'web/node_modules').glob('**/LICENSE*'):
 if p.is_file():
  target=out/'npm'/p.relative_to(root/'web/node_modules');target.parent.mkdir(parents=True,exist_ok=True);shutil.copy2(p,target)
raw=subprocess.check_output(['go','list','-m','-json','all'],cwd=root,text=True);decoder=json.JSONDecoder();modules=[]
while raw.strip():
 value,end=decoder.raw_decode(raw.lstrip());raw=raw.lstrip()[end:];modules.append(value)
for m in modules:
 if not m.get('Dir') or m.get('Main'):continue
 for p in pathlib.Path(m['Dir']).iterdir():
  if p.is_file() and p.name.lower().startswith(('license','copying','notice')):
   target=out/'go'/m['Path']/m.get('Version','')/p.name;target.parent.mkdir(parents=True,exist_ok=True);shutil.copy2(p,target)
