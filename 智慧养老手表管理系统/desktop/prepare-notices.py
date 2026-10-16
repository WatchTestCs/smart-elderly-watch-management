from pathlib import Path
from io import BytesIO
import json, os, shutil, zipfile

root=Path(__file__).resolve().parent.parent
metadata=json.loads((root/'desktop/version.json').read_text(encoding='utf-8'))
version=os.environ.get('SUINING_VERSION',metadata['version'])
channel=os.environ.get('SUINING_CHANNEL',metadata['channel'])
default_output=root/f'outputs/v{version}-{channel}'
out=Path(os.environ.get('SUINING_OUTPUT_DIR',default_output)).resolve()
release=out/'unsigned/Suining'
licenses=release/'third-party-licenses'
licenses.mkdir(parents=True,exist_ok=True)
inventory=[]

with zipfile.ZipFile(root/f'careold/target/careold-{version}.jar') as jar:
    for name in jar.namelist():
        if not name.startswith('BOOT-INF/lib/') or not name.endswith('.jar'): continue
        artifact=Path(name).name[:-4]
        inventory.append({'type':'java','artifact':artifact})
        with zipfile.ZipFile(BytesIO(jar.read(name))) as dependency:
            for entry in dependency.namelist():
                if entry.endswith('/') or not any(word in Path(entry).name.lower() for word in ['license','notice','copying']):continue
                if '..' in Path(entry).parts:continue
                destination=licenses/'java'/artifact/entry
                destination.parent.mkdir(parents=True,exist_ok=True)
                destination.write_bytes(dependency.read(entry))

modules=root/'oldIot/node_modules'
packages=list(modules.glob('*/package.json'))+list(modules.glob('@*/*/package.json'))
for package in packages:
    data=json.loads(package.read_text(encoding='utf-8'))
    name=data.get('name',package.parent.name)
    inventory.append({'type':'npm','name':name,'version':data.get('version'),'license':data.get('license')})
    for source in package.parent.iterdir():
        if source.is_file() and any(word in source.name.lower() for word in ['license','notice','copying']):
            destination=licenses/'npm'/name.replace('/','__')/source.name
            destination.parent.mkdir(parents=True,exist_ok=True)
            shutil.copy2(source,destination)

shutil.copy2(out/'tools/electron/LICENSE',release/'LICENSE')
shutil.copy2(root/'desktop/使用说明.txt',release/'使用说明.txt')
(release/'dependency-inventory.json').write_text(json.dumps(inventory,ensure_ascii=False,indent=2),encoding='utf-8')
(release/'THIRD_PARTY_NOTICES.txt').write_text(f'''岁宁 {version} · 第三方组件说明

本软件包含 Electron / Chromium、Eclipse Temurin OpenJDK、Spring、H2、Vue、Element Plus、Axios、ECharts 等开源组件。
Electron 及 Chromium 许可见同目录 LICENSE 与 LICENSES.chromium.html；Java 运行环境许可及源代码获取说明见 resources/runtime/legal 与 resources/runtime/NOTICE。
项目 MIT 许可另见 third-party-licenses/project-LICENSE；前端与后端组件许可见 third-party-licenses，字体许可随本地字体资源包含于服务端归档。
dependency-inventory.json 同时记录构建期 npm 依赖；列入清单不代表该包完整随软件运行。
未改变任何开源组件的原有许可。用户选定的岁宁山水 Logo 保留原构图。
''',encoding='utf-8')
project_license=root/'LICENSE' if (root/'LICENSE').exists() else root.parent/'LICENSE'
shutil.copy2(project_license,licenses/'project-LICENSE')
print('Third-party notices and license inventory prepared:',len(inventory))
