# 依赖与来源

| 依赖 | 固定信息 |
| --- | --- |
| Go 模块 | go.mod / go.sum（纯 Go modernc SQLite，coder/websocket） |
| Vue / Vite / TypeScript / Lucide / Playwright | web/package-lock.json；Node 仅构建和 QA 时使用 |
| Apache Guacamole JS | 1.6.0，源码 SHA-256 `81f9fd5a7b4377fb0ee295d0d4fec92e9667f2aafaa3d0ed8937f535deabdee4` |
| Apache Guacamole server | 1.6.0，源码 SHA-256 `8bc45675da96d7b6f39728160181e3d4ff3c08f460f6d26de5805b642bf13f2b` |
| FreeRDP / WinPR | 3.33.1，官方发布源码及 SHA-256 锁定于 `scripts/native/freerdp-source.json`；在 bookworm 中源码编译，不再安装 freerdp2-dev |
| 原生系统依赖 | Debian bookworm；产物内 dependencies.tsv 记录确切版本，系统库尚未切换到 apt 固定快照 |
| FPK 工具 | 本地 fnpack 1.2.3 原生模板；manifest 版本为唯一版本源 |

Apache 官方源码：[1.6.0 发布目录](https://archive.apache.org/dist/guacamole/1.6.0/source/)。JS 再生成脚本 `scripts/vendor-client.py`；原生编译参数见 `scripts/native/Dockerfile`，不启用 SSH、Telnet、Kubernetes、录像转码。

FreeRDP 源码由 `scripts/fetch-freerdp.py` 下载并检查固定 SHA-256；Docker 内再次校验。Guacamole 优先检测 FreeRDP 2，因此构建环境不安装 2.x 开发包，并检查 RDP 插件实际链接 freerdp3。保留 FFmpeg 解码、客户端通道和 WinPR NLA 加密自检；不构建桌面客户端或 RDP 服务端。自行编译的 FreeRDP/WinPR 版本及源码校验和追加到包内 `dependencies.tsv`，来源信息和许可证随包保存。

FreeRDP 使用 Clang、Release/O2、关闭 LTO，并通过 BuildKit 按架构缓存编译目录。源码归档提取后刷新文件时间，防止升级源码时复用旧对象。macOS arm64 主机交叉构建 x86 时需使用正常工作的 x86 模拟器；本地 QEMU 7.0/8.2.2 曾出现编译器及系统 grep 崩溃，切换到经官方发布包 SHA-256 校验的 QEMU 10.2.3 后，grep 隔离检查通过。模拟器属于构建主机工具，不随 FPK 分发。

旧的 `scripts/native/pin-rdp-certificate.c` 辅助工具已改用 FreeRDP 3 证书 API；FreeRDP 3 使用独立 server 证书文件，不能沿用 2.x 的 known_hosts2 测试假设。主应用依然采用后端授权的会话证书指纹，不依赖该辅助工具或共享信任存储。

Guacamole 1.6.0 保留原版本及源码哈希，通过 `patch-freerdp3-lifecycle.py` 回移图形初始化时序修复：FreeRDP 3.31+ 在连接激活时才创建缓存，GDI 与绘制回调须移至 PostConnect。参考上游 [rdp.c（bb1ba8e）](https://github.com/apache/guacamole-server/blob/bb1ba8ed1414ec6a239cea265d4e9c7e23b9c6a3/src/protocols/rdp/rdp.c)。同时仅允许弃用警告不作为编译错误，避免 FreeRDP 3.x 仍保留的弃用声明阻断接口检测；其他编译错误检查保留。

私有库闭包依据 guacd 和所有随包插件的 ldd 提取，不覆盖宿主 /usr/lib；glibc 及动态加载器使用宿主系统，故必须验证目标 fnOS 的 ABI。FreeRDP、LibVNCClient、Cairo、图像、加密和字体依赖以实际构建为准。CA 副本和 Debian 组件版权说明随包保存。正式分发前需按实际 LGPL 等组件补齐对应源码提供与重建材料；各版本安装包通过 GitHub Releases 分发。

官方契约核对：[fnOS 统一网关](https://developer.fnnas.com/docs/core-concepts/gateway-registration/)（管理员 Header 为字符串 true/false）、[用户向导](https://developer.fnnas.com/docs/core-concepts/wizard/)、[Guacamole 协议](https://guacamole.apache.org/doc/gug/guacamole-protocol.html)、[JS 隧道实现](https://guacamole.apache.org/doc/guacamole-common-js/main_webapp_modules_Tunnel.js.html)。

2026-11-09 再次通过官方站点核对统一网关、环境变量及应用入口章节。官方规范规定由网关注入用户身份，应用负责业务鉴权；没有把单一安装域名描述为统一网关的限制，也未明确保证代理后的 Host 与浏览器域名一致。项目据此保留严格 Origin 与 CSRF 校验，增加显式来源列表和可信网关同源处理。

FreeRDP 3 的 Negotiate 会先获取 Kerberos 凭据；邮箱用户名可能在 realm/KDC 查询阶段阻塞。`patch-nla-account-auth.py` 对域为空且未使用 `DOMAIN\\user` 的密码账户、以及明确的 MicrosoftAccount，设置 `AuthenticationPackageList=none,ntlm`，保留 NLA 和会话证书指纹校验。需要 Kerberos 的 AD 账户应填写显式域（或使用 `DOMAIN\\user`），以保留自动协商；仅邮箱式 UPN 且域留空按非域密码账户处理。
