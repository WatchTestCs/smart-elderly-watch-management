# Windows 签名

默认构建可不签名；`-SelfSign` 使用本机代码签名证书给 sidecar、主程序及安装器签名，并记录签名结果。自签名不是公开受信任证书，不会自动安装根证书或消除 SmartScreen。

正式证书通过 SUINING_SIGNING_THUMBPRINT 与 SUINING_TIMESTAMP_SERVER 配置；sign.ps1 检查证书用途、私钥、有效期及时间戳。私钥、PFX 和密码不得进入仓库或 Release。

先签 sidecar 再计算内嵌哈希；主程序签名后 bundler 使用 --no-binary-patching，避免破坏已有签名；最终再签 Setup 并生成校验和。
