# 完整 RDP 回归（仅本地隔离环境）

`test-full-rdp.py` 使用真实 guacd/FreeRDP，对本地 XRDP 执行 TLS 登录和图像接收；Shadow 场景额外覆盖完整 NLA 与 GFX 持续绘制。所有用户、口令、证书和屏幕内容均为公开的合成测试数据。Docker 运行时禁用网络，不发布任何宿主端口，不访问 NAS 或用户 Windows。

先完成原生引擎构建，再执行：

```bash
version=$(python3 -c 'import json; print(json.load(open("scripts/native/freerdp-source.json"))["version"])')
docker build --platform linux/amd64 --build-arg NATIVE_IMAGE="remote-desktop-native:guac1.6.0-freerdp${version}-amd64" -t remote-desktop-full-rdp-qa scripts/native/qa
mkdir -p build/rdp-session-test
docker run --rm --platform linux/amd64 --network none -v "$PWD/build/rdp-session-test:/qa" remote-desktop-full-rdp-qa
docker build --platform linux/amd64 -f scripts/native/qa/Dockerfile.shadow -t remote-desktop-shadow-qa scripts/native/qa
mkdir -p build/rdp-nla-gfx-test
docker run --rm --platform linux/amd64 --network none -e QA_SERVER=shadow -v "$PWD/build/rdp-nla-gfx-test:/qa" remote-desktop-shadow-qa
```

Shadow 默认验证 GFX；使用 `-e QA_DISABLE_GFX=1` 覆盖无损兼容路径。图像检查拒绝纯色/近黑初始化图块，要求真实明暗变化；NLA/GFX 场景还要求持续绘制至少 8 秒且连接期间没有原生断言。该回归不替代真实 Windows 或 fnOS 验收。生成的文件仅在 `build/`，QA 服务端和依赖不进入 FPK。
