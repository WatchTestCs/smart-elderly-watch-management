# ���� 2.0.6 �� Դ�빤��

���ļ�������Դ�빤����ڣ�˵����Ŀ¼ְ�𡢹������̺ͱ������б߽硣

�������� **Tauri 2 / Rust ����ǡ�Go ҵ�����Vue 3 / Vite ������ֱ�ģ����**��ɡ�MySQL ��Ψһҵ�����⣬PostgreSQL �����첽����ͶӰ�����ݿ��ɲ��𷽶���������

[��Ʒ������ܹ�](../README.md) �� [2.0.6 �Ľ�](../docs/releases/2.0.6.md) �� [˫��������Ǩ��](../docs/DUAL_DATABASES.md) �� [ʵ��Э��](../docs/ARCHITECTURE.md)

## ����ְ��

| Ŀ¼ | ְ�� | ��Ҫ��� |
| --- | --- | --- |
| `desktop` | ԭ�����ڡ���ʵ�������޳�ʼ����Go ��ϣУ������������ | `src/main.rs`��`capabilities`��`build.ps1` |
| `backend` | ҵ�� API���豸���롢�Ự���ջ��������ҵ | `cmd/server`��`internal/handlers`��`internal/care`��`internal/analytics` |
| `backend/migrations` | MySQL ҵ����� PG Ǩ�ơ�PG ���� schema | ��Ƕ SQL ��汾У�� |
| `oldIot` | Vue ·�ɡ����������ҳ�桢���塢��ͼ��QA | `src`��`vite.config.js`��`qa` |
| `watch-sim` | �ֱ��ӿ������ж˿ڷ���ģ�� | ���߼������ļ� |

Rust ���� Go��Vue �� Go ��ȡ��Ƕ������Դ������ͬԴ API���״γ�ʼ��������ԭ�� IPC��Go ����ҵ���д��ʹ�� MySQL��������ҵ�����ύ outbox �¼�д�� PG��ʧ��ʱ�������С�`oldIot` ��ģ���� `careold-go` ����ʷ��������ǰ����Ҫ Java��Electron��H2 ��� Webpack ��������

## Windows ��������

׼�� PowerShell 7��Node.js��`^20.19.0 || >=22.12.0`����Go��Rust��C++ Build Tools �� Windows SDK��Go ģ��������� 1.24.0������ 2.0.6 ����ʹ�� Go 1.27.2��Rust ��������� 1.85���汾�����ļ���ģ������Ϊ׼��

```powershell
./desktop/validate-version.ps1
./desktop/build.ps1
# ��ѡ������ǩ�����ǹ���������֤��
./desktop/build.ps1 -SelfSign
```

����˳������������װ �� Vue �������� �� ���� Go ��Ƕ��Դ �� Go ���ԡ���̬��顢���� �� sidecar ��ϣ �� Tauri ���� �� NSIS Setup���ű�֧�� `CARGO_TARGET_DIR`����װ��λ�ڹ���Ŀ¼ `outputs/v2.0.6-preview/Suining-Setup-2.0.6-win-x64.exe`���汾У�鸲��ǰ�ˡ����桢Cargo��Go �汾�ļ������ļ��еĲ�Ʒ�汾��

Ĭ�ϲ�ǩ�����ⰲװ ZIP��Դ�� ZIP��˵����ժҪ�ǵ����������裬`build.ps1` �����Զ�����ȫ������������Setup ȱʧ WebView2 ʱ������װ���ⰲװ�������� WebView2��������ͬĿ¼������ EXE��

## ǰ��˿���

��һ���ն˽��� `backend`�����ÿ����Ⲣ��������

```powershell
$env:MYSQL_DSN='suining:�滻����@tcp(127.0.0.1:13306)/suining'
$env:PG_ANALYTICS_URL='postgres://suining_analytics:�滻����@127.0.0.1:15432/suining_analytics?sslmode=disable'
$env:ADDR='127.0.0.1:8795'
go run ./cmd/server
```

��һ���ն˽��� `oldIot`��

```powershell
npm ci
npm run dev
```

Vite �� `/restful` ������ `127.0.0.1:8795`���� API ģʽ��ֱ�ӿ������������湹��������Ƕ��ǰ�ˡ����������������ͨ�� Tauri ��ʼ������Ա��ʹ��ԭ����������״γ�ʼ������Ӧ�ſ� HTTP ��ʼ����Ȩ��

������ TLS ����ֻ�������ܿر���������Զ����������֤֤�顣PG ���ݲ����ã�����ҵ���Կ�ʹ�ã��������¼����ѹ����������������ʹ�� `SUINING_MYSQL_DSN` / `SUINING_PG_ANALYTICS_URL`�������޸ĺ��������� `SUINING_DATABASE_URL` ������Ϊ������ˡ�

## ���ԡ�������Ǩ��

- [��� README](backend/README.md)��Go ���ԡ�ר�� MySQL / PG ���ɻ��������ߵ�����������
- [ǰ�� README](oldIot/README.md)������������Playwright QA���Ự����Ӧʽҳ�档
- [˫��ָ��](../docs/DUAL_DATABASES.md)��Compose��Ȩ�ޡ��¼����ԡ��� PG ���� MySQL ��Ǩ�ơ���װ�����Զ�Ǩ�����ݿ⣻ͼƬ���и��ƣ��� Java/H2 ��ֱ��֧�֡�
- [�ջ�����](../docs/CARE_SERVICE_DEPLOYMENT.md)������ֵ�ء�֪ͨ���ء��ʹ��ִ���˹���Ӧ������رպ󱾵� Go ��֮�˳������������פֵ�ط���

MySQL ������ʱ�������ܴ򿪣�ҵ�� API ��ʾ�����ã��ָ�����Ự�����ط�ʧ��ҵ��д�롣PG ������ֻ�ӳٷ�����Ĭ�� Windows ͼƬ����־��������Ϣλ�� `%APPDATA%\����`���Ự�� Go �ڴ棬ҵ����ͼƬ�ֱ𱸷ݡ�

���� 2.0.6 �ѹ�������ɰ汾��ժҪ�����÷����飬δִ�а�װ����ԭ�����洰�����ա���ʷ Linux ��������֤������������� Linux ԭ�������������� `preview`���˴����ϱ���������Զ�̽����������� [���ϼ�¼](../docs/INTEGRATION_2.0.6.md)����ǰ��װ��δ���������ϣ����β����� Release��
