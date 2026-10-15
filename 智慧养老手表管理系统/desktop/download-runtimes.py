from pathlib import Path
from concurrent.futures import ThreadPoolExecutor
import hashlib, json, urllib.request, zipfile

root=Path(__file__).resolve().parent.parent
tools=root/'outputs/desktop-release-20261015/tools'
tools.mkdir(parents=True,exist_ok=True)
lock=json.loads((root/'desktop/runtime-lock.json').read_text(encoding='utf-8'))
def digest(file):
    result=hashlib.sha256()
    with file.open('rb') as f:
        for block in iter(lambda:f.read(1024*1024),b''):result.update(block)
    return result.hexdigest()
def transfer(url,target,start=None,end=None):
    headers={'User-Agent':'Suining-build/2.0'}
    if start is not None:headers['Range']=f'bytes={start}-{end}'
    for attempt in range(4):
        try:
            with urllib.request.urlopen(urllib.request.Request(url,headers=headers),timeout=60) as response,target.open('wb') as output:
                if start is not None:
                    assert response.status==206 and response.headers['Content-Range'].startswith(f'bytes {start}-{end}/')
                while True:
                    block=response.read(65536)
                    if not block:break
                    output.write(block)
            if start is not None:assert target.stat().st_size==end-start+1
            return
        except Exception:
            if attempt==3:raise
def install(name,entry):
    archive=tools/(name+'-verified.zip')
    for candidate in [archive,tools/(name+'.zip')]:
        if candidate.exists() and digest(candidate)==entry['sha256']:
            archive=candidate;break
    else:
        if name=='electron':
            chunk=4*1024*1024;size=entry['size'];parts=[]
            def part(i):
                target=tools/f'electron-part-{i:03d}'
                start,end=i*chunk,min(size-1,(i+1)*chunk-1)
                transfer(entry['url']+f'?part={i}',target,start,end)
                return target
            with ThreadPoolExecutor(max_workers=6) as pool:parts=list(pool.map(part,range((size+chunk-1)//chunk)))
            with archive.open('wb') as output:
                for file in parts:output.write(file.read_bytes())
        else:transfer(entry['url'],archive)
    assert digest(archive)==entry['sha256'],name+' SHA256 mismatch'
    destination=tools/name
    with zipfile.ZipFile(archive) as z:
        for file in z.namelist():
            target=(destination/file).resolve()
            assert target.is_relative_to(destination.resolve()),'Unsafe archive entry'
        z.extractall(destination)
    print(name,entry['version'],'checksum verified',flush=True)
with ThreadPoolExecutor(max_workers=3) as pool:list(pool.map(lambda item:install(*item),lock.items()))
