# Runbook: Signing

当前本地脚本使用 `CN=Suining Local Build` 自签名证书。

- 私钥只存在于本机 CurrentUser certificate store；
- 脚本创建的 key 标记为不可导出；
- 仓库和 Release 只能包含公开 CER；
- 不将证书加入 Trusted Root；
- `NotTrusted` 在自签场景不等于文件没有签名；
- 普通 CI 不运行签名阶段。

若改为商业代码签名，必须新增 ADR，使用保护的 release environment，并把签名实现与普通 build 分离。
