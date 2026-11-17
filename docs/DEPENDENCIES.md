# 依赖与来源

| 依赖 | 固定信息 |
| --- | --- |
| Go 模块 | go.mod / go.sum（纯 Go modernc SQLite，coder/websocket） |
| Vue / Vite / TypeScript / Lucide / Playwright | web/package-lock.json；Node 仅构建和 QA 时使用 |
| Apache Guacamole JS | 1.6.0，源码 SHA-256 `81f9fd5a7b4377fb0ee295d0d4fec92e9667f2aafaa3d0ed8937f535deabdee4` |
| Apache Guacamole server | 1.6.0，源码 SHA-256 `8bc45675da96d7b6f39728160181e3d4ff3c08f460f6d26de5805b642bf13f2b` |
| 原生系统依赖 | Debian bookworm；产物内 dependencies.tsv 记录确切版本，系统库尚未切换到 apt 固定快照 |
| FPK 工具 | 本地 fnpack 1.2.3 原生模板；manifest 版本为唯一版本源 |

Apache 官方源码：[1.6.0 发布目录](https://archive.apache.org/dist/guacamole/1.6.0/source/)。JS 再生成脚本 `scripts/vendor-client.py`；原生编译参数见 `scripts/native/Dockerfile`，不启用 SSH、Telnet、Kubernetes、录像转码。

私有库闭包依据 guacd 和所有随包插件的 ldd 提取，不覆盖宿主 /usr/lib；glibc 及动态加载器使用宿主系统，故必须验证目标 fnOS 的 ABI。FreeRDP、LibVNCClient、Cairo、图像、加密和字体依赖以实际构建为准。CA 副本和 Debian 组件版权说明随包保存。正式分发前需按实际 LGPL 等组件补齐对应源码提供与重建材料；当前仅为本地开发预览，没有发布到应用市场。

官方契约核对：[fnOS 统一网关](https://developer.fnnas.com/docs/core-concepts/gateway-registration/)（管理员 Header 为字符串 true/false）、[用户向导](https://developer.fnnas.com/docs/core-concepts/wizard/)、[Guacamole 协议](https://guacamole.apache.org/doc/gug/guacamole-protocol.html)、[JS 隧道实现](https://guacamole.apache.org/doc/guacamole-common-js/main_webapp_modules_Tunnel.js.html)。

2026-11-09 再次通过官方站点核对统一网关、环境变量及应用入口章节。官方规范规定由网关注入用户身份，应用负责业务鉴权；没有把单一安装域名描述为统一网关的限制，也未明确保证代理后的 Host 与浏览器域名一致。项目据此保留严格 Origin 与 CSRF 校验，增加显式来源列表和可信网关同源处理。
