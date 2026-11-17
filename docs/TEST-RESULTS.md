# 本轮验证记录

## 2026-11-17：0.1.47 FreeRDP 3.33.1

- FreeRDP / WinPR 3.33.1 官方源码 SHA-256 已校验，Guacamole 1.6.0 实际链接 freerdp3；导出的私有库不含 freerdp2/winpr2。
- 加载实际导出库读取版本为 3.33.1；FFmpeg、cliprdr、drdynvc、disp 和 rdpgfx 内置通道检查通过；动态依赖闭包与 WinPR MD4 已知向量自检通过。
- 真实 guacd/FreeRDP 对合成 TLS 服务：匹配主机/端口/SHA256 后进入 NLA；证书变化拒绝；无会话授权、共享信任、其他端口共享信任及已信任证书与会话指纹不一致均被拒绝。合成服务没有执行 Windows 登录。
- 新版证书辅助工具验证：错误 SHA 拒绝、其他主机记录保留、重复写入幂等、不同指纹不自动替换、文件权限 0600。
- 真实 guacd/LibVNCClient 对合成 RFB 服务：同时提供 ARD/VNC 密码时选择密码模式；密码模式首帧通过；只有 ARD 时明确拒绝。
- 干净 bookworm-slim 容器、禁用外部网络、UID/GID 925：只读挂载导出的私有运行库，RDP/VNC 插件均初始化并按预期拒绝关闭的目标端口；日志 `build/freerdp-clean-runtime.log`。不作为 fnOS ABI 或真实远端连接的证明。
- 回归发现 FreeRDP 3.31+ 缓存初始化时序与 Guacamole 1.6.0 不兼容，已按上游 bb1ba8e 的实现将 GDI 和绘制回调移至 PostConnect，消除连接前的 NULL cache 断言。另对新版头文件的弃用警告保留为警告，其他编译错误仍阻断构建。
- Go race 测试、前端类型检查/生产构建与 19 项前端单测通过。未安装 NAS；真实 Windows 登录、桌面画面、动态缩放和双向剪贴板仍待实机验收。
- 本机构建环境原有 QEMU 7.0，Ubuntu 仓库 QEMU 8.2.2 也在隔离的 grep 检查中出现内部 SIGSEGV。已校验并启用 tonistiigi 官方发布的 QEMU 10.2.3（发布包 SHA-256 `ca19f8d202b977904f70fa6464b5ef9439740951d40f89730d2f2981f853bfaa`），基础命令与最终原生构建通过。该工具仅用于 Colima 构建虚拟机，不进入 FPK。
- 完整 `scripts/build.sh amd64` 通过，145 个 ELF 的 amd64 架构与包目录校验通过。产物 `dist/remote-desktop-0.1.47-x86.fpk`；包内 manifest、程序 `--version` 均为 0.1.47，包内依赖记录 FreeRDP/WinPR 3.33.1 且无 2.x 库；SHA-256 为 `e709c028fc31777c82bc918b9cdd2890600ae66a8858e2e5071b9fbfb3a307b3`。打包原生文件与干净容器已测导出文件逐个哈希一致。

## 初始验证记录

日期：2026-11-09。测试来自本地开发环境，未连接真实 fnOS / Windows / Mac。

- Go race 检查：用户归属隔离、伪造开发身份、CSRF、未知字段拒绝、凭据认证加密与篡改检测、并发临时凭据单次领取、会话限额、配置版本冲突、配置变更撤销旧票据、8 路并发票据消费、显示同步前保持连接中、配对密钥备份恢复、单实例数据锁、管理员设置、引擎端口冲突与幂等停止。
- Guacamole：中文、组合字符和 emoji 的码点长度、单字节碎片读取、粘连指令、畸形长度、非法 UTF-8、参数重排握手和有界模糊测试种子。
- 前端：vue-tsc 与 Vite 生产构建；本机 Chrome / Playwright 管理流程，包括创建/删除分组、连接、凭据，搜索、筛选、探测失败、引擎不可用状态收敛、明文不回显、浅/深色和窄屏。
- 浏览器模拟协议流程：官方 JS 客户端接收显示、鼠标、键盘、失焦释放 Ctrl、显式 UTF-8 中文/emoji 文本剪贴板、断开及创建新会话重连。模拟服务仅为 scripts/qa-guacd.py，不是实际 RDP 或 VNC。
- 原生 Linux x86：实际构建 guacd 1.6.0、RDP/VNC 插件，FreeRDP 2.11.7、LibVNCClient 0.9.14、OpenSSL 3.0.22；具体 Debian 修订版本见包内 dependencies.tsv。
- 干净 Debian bookworm-slim x86 运行容器：仅挂载产物，ldd 检查无缺库；真实 guacd 的两个协议插件都完成握手，并对刻意关闭的目标端口返回失败。验证工具 cmd/engine-check，日志 build/clean-runtime-smoke.log。这只证明该容器中插件可加载，不证明 fnOS ABI、Windows 认证或 Mac 认证兼容。
- fnpack 1.2.3：成功构建 x86 开发预览 FPK；manifest、Go 注入版本、前端能力版本统一来自 0.1.0。

Computer Use 两次初始化均连接超时，因此实际界面证据来自 Playwright 截图与浏览器操作，没有声称 Computer Use 验收成功。

截图与日志位于 build/（未作为业务数据或包内连接示例）。真实设备、证书、iframe/Safari、长时间稳定性、完整升级回滚与卸载保留数据仍待手测，详见 COMPATIBILITY.md。

## 0.1.1 安装权限回归

收到 fnOS 应用中心“设置目录权限失败”截图。旧包包含 8 个直接断开的链接（另有链接链会经过断开的目标），在隔离 Linux 容器中执行递归解引用 chown 可复现失败；这支持包内链接是安装故障的原因，但未取得 NAS 安装器日志确认其内部命令。新包移除全部符号链接，版权文件保留为普通文件；明确包用户及组为 remote-desktop；新增打包前目录、文件类型、模式及私有数据检查。版本源更新至 0.1.1。

新 FPK 解包复核：0 个符号链接/硬链接，无 .DS_Store，目录模式 0755，包用户声明与 SHA-256 匹配。隔离 Debian x86 容器中递归 chown/chmod 成功；以 UID/GID 12345 的非 root 身份从新包启动 Go 服务与真实 guacd 成功，启动日志版本为 0.1.1。尚未在用户 NAS 上复测安装。

## 0.1.2 启停脚本回归

收到 0.1.1 的“本地应用启动失败”反馈，截图未提供 NAS 版本和服务日志。对旧版完整生命周期脚本进行隔离重现：真实 Linux 原生 Go 服务搭配仅用于监听的测试引擎，首次 start 成功，但服务进程的 FD 9 仍指向 lifecycle.lock，后续 status 在 3 秒超时窗口内无法获取该锁。这是已确认的脚本缺陷，但未获得 NAS 日志来证明它是该设备唯一的失败原因。

0.1.2 在启动后台服务前关闭生命周期锁描述符，使用程序内置的 Unix Socket 健康检查代替 curl，并区别活动 Socket、确定失效的 Socket 和无法确认状态的 Socket；启动失败会保留并反馈最近的服务错误。版本源为 0.1.2。

Linux 原生脚本回归使用 ARM64 Go 业务二进制及模拟引擎（仅用于避免 x86 仿真进程在 /proc/exe 中显示为解释器而干扰进程身份检查），以无 root 权限的 UID/GID 12345 验证完整脚本。此项只验证启停契约，不代表发布 ARM64 FPK，也不代表真实 RDP/VNC。

### NAS 现场检查（2026-11-09）

经用户 OAuth 授权连接其 NAS，确认 fnOS Mainland-PE 1.2.1010 x86，已安装 remote-desktop 0.1.1，数据卷及安装卷均为 1。应用中心系统日志显示：安装后首次启用成功，约 27 秒后报告异常退出；后续启用失败指向启动脚本。通过 OAuth 文件下载读取 server.log，确认旧版服务已在应用 Socket 监听。资源监控的进程详情及服务归属也确认旧版业务进程与 guacd 仍存在。没有读取数据库、主密钥或其他应用的数据。

0.1.2 上传解析及系统约束检查通过。当前 trim-cli 的本地 FPK install 路径会提交首次安装接口，已安装应用返回 10236（系统日志明确为“应用已存在”）；update 子命令又先创建云端下载任务，不能直接升级已暂存本地包。改用同一 OAuth 授权的官方原生 HTTP 代理，update/info 成功，update/task 已提交，但 NAS 系统日志最终记录“更新失败：应用停止失败”。现场安装版本仍为 0.1.1，未声称已完成升级。

浏览器控制工具初始化超时；目标域名的 SSH 默认 22 端口连接超时。当前等待用户提供可用 SSH 连接信息或在 NAS 终端运行定向修复脚本。`scripts/repair-legacy-start-lock.sh` 核对可执行路径、PID 启动时间及遗留 FD 9 后才发送 TERM，不删除数据库/密钥；已在隔离 Linux 容器中验证会释放锁且保留数据。

### NAS 升级与启停验证完成

用户在 NAS 终端对核实的旧版进程发送 TERM 后，应用中心成功完成停止，状态核对为 stopped。重新确认 update/info 与现有卷号后提交新的原位升级任务（仅在上次明确失败且旧服务已停止后重试）；任务返回 status=2、progress=100，应用中心实际安装版本为 0.1.2、运行状态为 running、isOpen=true，服务日志出现 0.1.2 的 Socket 监听记录。

随后通过应用中心实际执行一次停止和启动：两次命令均成功，停止后的查询为 stopped，启动后的查询为 running。再等待并复查，跨过旧版约 27 秒的故障窗口，仍显示 0.1.2 运行中；系统日志记录更新成功、停用成功与新版启用成功，未出现新的异常退出。已核对升级前备份目录存在数据库、32 字节主密钥及来源/GID/版本元数据，只查看文件名与大小，未读取密钥内容。真实 Windows/Mac 桌面、网关页面交互和长期稳定性仍未据此宣称通过。

## 0.1.3 请求来源与令牌续取

根据用户保存连接时报“请求来源或安全令牌无效”的截图，重新核对官方统一网关、环境变量和应用入口规范。当前 NAS 显式来源配置为 desktop.nascp.cn，而用户提供的另一个入口为 nascpfnos.5ddd.com；原实现仅允许安装时单一来源，且升级重启使旧页面的 CSRF 令牌失效。

实现严格来源列表与可信 Unix 网关的实际 HTTPS 同源 Host 校验，不信任 X-Forwarded-Host；WebSocket 同样使用通过校验的来源。区分 ORIGIN_REJECTED 与 CSRF_EXPIRED，后者在任何写入前返回；前端仅对该明确错误续取一次 /me 并保留原请求体重试，要求同一登录用户，不重放网络异常或不确定的结果。

Go 回归覆盖多域名、内部 Host 改写、真实同源 Host、跨站、伪造转发 Header、跨用户令牌、过期令牌和缺少来源。前端三项令牌单元测试、两项向导真实域名规则测试与 Chrome 表单流程通过，表单测试主动注入一次令牌过期响应后仍完成真实本地保存。Computer Use 连接超时；未把本地验证当作用户 NAS 页面点击验收。

0.1.3 已在用户 NAS 原位升级成功：任务 status=2/progress=100，应用中心实际版本 0.1.3、状态 running、isOpen=true。通过应用中心配置接口保存原有 desktop.nascp.cn 与用户提供的 nascpfnos.5ddd.com 两个 HTTPS 来源，随后回读配置确认。当前等待用户刷新新版页面后确认真实保存操作；未把本地模拟过期响应的成功冒充 NAS 页面保存成功。

用户已确认 0.1.3 在 NAS 页面保存成功。

## 0.1.4 移除安装参数

按用户要求移除安装与配置向导中的全部来源/GID 参数。来源不再依赖名单或代理 Host：同源页面通过自定义初始化 Header 提交自身来源，/me 返回绑定该 NAS 用户和来源的 CSRF 令牌；无 CORS 授权，跨站预检/显式跨站初始化被拒绝，写操作验证浏览器 Origin 和绑定令牌。取票和 WebSocket 也绑定创建会话的来源。只在独立开发入口保留开发认证的固定来源配置。Socket 继续使用已验证的包用户默认组与 0660 模式，不接受安装向导的 GID。

Go race 回归覆盖无配置下不同域名/IP/端口/IPv6、内部 Host 改写、来源与用户令牌隔离、跨站与矛盾 Header、跨源取票和 WebSocket 票据拒绝。Linux 非 root 生命周期测试通过：无任何向导值和参数文件即可启动，旧参数文件内容即使无效也不影响升级后的启动，且旧版启动锁恢复仍通过。

## 0.1.7 Win10 认证诊断与 NLA 运行依赖

用户在 0.1.5 连接 NAS 上 Win10 虚拟机时收到认证失败，并确认使用本地账户与实际登录密码。源代码检查发现 769 安全协商/认证与 771 权限拒绝被合并显示为密码错误；现已按 Guacamole 1.6.0 固定消息区分，未知情况保留不确定状态。日志只输出服务生成的会话 ID、枚举原因和校验后的数值状态，不输出任意上游消息或凭据。Go race/vet 及分类回归通过。

原生依赖收集仅覆盖 ldd 输出，漏掉 OpenSSL 3 动态加载的 legacy.so。使用实际 WinPR 2.11.7、公共 MD4("abc") 已知向量复现：将 OPENSSL_MODULES 指向不存在目录会出现 no md4 support 并失败；使用完整 provider 通过。0.1.7 打包该模块、固定私有引擎模块路径，加入同路径的启动前自检。独立 Debian x86 容器仅使用打包目录的库和 provider，自检 PASS；非 root 生命周期回归（该测试的 crypto 为 stub）仍通过。FPK 已检查包含模块、检查程序和版本。

用户授权直接 NAS 测试。0.1.7 原位升级任务 status=2、progress=100，实际版本 0.1.7、running、isOpen=true；NAS server.log 出现 NLA crypto self-test PASS 与 0.1.7 Socket 监听，证明该 NAS 的自检成功。Computer Use 连接超时；OAuth 接口无法作为自定义应用浏览器会话使用，未自动读取用户凭据或绕过身份认证。尚未完成真实 Win10 桌面连接验收，等待用户触发一次重连后读取分类日志，不能把原生自检当作 Win10 登录成功。

### 0.1.7 / 0.1.9 真实重连结果

用户重试仍失败；0.1.7 与 0.1.9 的 NAS 日志均记录 REMOTE_SECURITY_FAILED / Guacamole 769，不能再据此判断为密码错误。0.1.9 的详细分类没有输出，后续检查定位到 Guacamole 把 FreeRDP 所有日志重定向为 debug，默认 info 会丢弃它们。

### 0.1.10 详细诊断链回归

显式启用私有 guacd debug；为避免原始调试日志进入宿主系统日志，定向修改原生 guacd 的日志实现，禁用 syslog 调用，仅经应用持有的 stdout/stderr 管道过滤为固定枚举。实际子进程管道测试通过，验证原始敏感测试文字不会出现在输出中。构建的 guacd 动态符号检查确认不再导入 syslog。

新增 scripts/native/test-certificate-diagnostic.py：只在隔离容器内生成一次性测试证书、模拟 RDP X.224/TLS 服务，用公开测试用户名密码驱动真实 guacd/FreeRDP。证书被拒绝、Guacamole 返回 TLS 错误（519），并可捕获底层 Certificate validation failed，证明诊断链的原生源头可观测。该测试不是 Windows 兼容性验收，也不能证明用户 Win10 的实际原因就是证书。

0.1.10 已在用户 NAS 原位升级完成：任务 status=2/progress=100，实际版本 0.1.10/running/isOpen=true，server.log 记录 NLA crypto self-test PASS 与 0.1.10 监听。升级后尚无新的连接失败记录，等待用户触发重连。Computer Use 仍不可连接，因此不能宣称已代为操作连接或取得真实桌面。

## 0.1.11 不可写用户目录复现与修复

用户在 NAS 运行无凭据探测：Win10 选择 NLA，TLS 1.2 与 AES256-GCM-SHA384 握手成功。该探测只读取证书，不做证书信任验证，也不证明认证成功。

同版本真实 guacd/FreeRDP 在 UID 925、HOME=/root 下复现：当前家目录不可写、error creating directory、certificate store initialization failed，最终 SECURITY_NEGO_CONNECT_FAILED/769；此时未执行证书验证。改为临时独立的 HOME/XDG 目录后证书库成功初始化，测试用自签名证书被严格拒绝并返回 519。两种路径均只连接本地模拟服务，不使用用户账号密码。

修复为服务端启动时准备应用数据 engine 下 0700 的 home/config/cache/data，向私有子进程传递对应环境；继承目录不可写仅记录固定枚举，不记录目录原始值。Go 子进程环境与权限回归、符号链接目录拒绝、全量 race/vet 通过。

0.1.11 FPK 构建和当前 Linux 生命周期回归通过。通过应用中心上传两次均在 download/upload 超时，改用文件模块上传同样在 upload 超时，尚未启动任何 0.1.11 安装/升级任务。内网 192.168.1.88 的 HTTPS 443/5667 与 HTTP 5666 探测均超时，没有降低 TLS 校验或改用明文传输。当前需要用户通过应用中心手动安装本地 dist/remote-desktop-0.1.11-x86.fpk，不能声称已在 NAS 部署或完成 Win10 重连验收。

### 0.1.11 NAS 根因确认及证书拒绝

用户手动安装 0.1.11 后，实际 NAS 版本为 0.1.11/running；启动日志出现 INHERITED_HOME_UNWRITABLE，随后重连出现 CERTIFICATE_VERIFY_FAILED、CERTIFICATE_REJECTED、TLS_CONNECT_FAILED/519。说明该 NAS 确有继承家目录不可写问题，修复后已推进到证书校验；真实连接仍被证书信任校验拒绝。

用户在 Win10 本机证书存储读取的 SHA256 与先前 NAS 无凭据探测完全一致，为 3b81127f1c64164a48ebec52fcca512886afd68e7fa2000ec5fcd1b0e742c025，主体 CN=DESKTOP-CG8D6VR。

准备了显式证书信任工具 scripts/trust-rdp-certificate.py 与 scripts/native/pin-rdp-certificate.c：必须以应用账户运行、显式提供从 Windows 核对的 SHA256，再从保存的 win 连接重新读取对端公开证书并二次比对，才通过 FreeRDP 标准 API 向该应用的 known_hosts2 追加主机/端口/指纹。既有不同指纹不自动替换；不读取任何凭据，不执行登录，不设置 ignore-cert 或自动首次信任。

实际原生引擎的模拟 TLS 验证通过：正确指纹可推进到合成 NLA 协商；证书改变被拒绝；换端口仍拒绝。工具回归还覆盖错误 SHA 不写入、保留其他主机条目、重复操作幂等、不同旧指纹拒绝替换和 known_hosts2 0600。当前仅上传工具，尚未在用户 NAS 上执行信任写入，待用户显式运行确认命令；不宣称已连接 Win10。

## 0.1.12 应用内证书确认、信任管理及自动继续连接

RDP 会话在取出保存凭据前完成无登录信息的证书探测和授权。信任记录按 NAS 用户与连接保存；绑定目标、端口、指纹和版本，确认令牌还绑定来源且短时一次性。正常 CA/名称校验通过可自动继续；自签名、名称不符或既有证书变化须显式确认；过期或尚未生效的证书不可确认。连接菜单与设置页提供查看、重新检查、更新和撤销；撤销/替换断开该连接已有会话。

实际原生握手使用独立会话指纹验证回调，强制核对叶证书及有效期；不使用 ignore-cert、自动信任或共享 known_hosts 授权。当前原生构建四项隔离测试通过：正确指纹推进到合成 NLA；错指纹拒绝；只有旧共享信任、无会话授权时拒绝；共享信任存在也不能覆盖本次错指纹。全部为本地合成服务，不代表真实 Windows 认证成功。

全量 Go race、go vet、前端类型/构建、Node 单元测试通过。Go 覆盖信任持久化、用户隔离、来源/版本/有效期绑定、令牌重放、撤销活动会话以及在确认前不解密凭据。Playwright 五项流程均通过（完整轮次四项通过；桌面流程受同一测试用户的 1 秒探测速率限制影响，增加测试间隔后单项重测通过）。覆盖首次取消不建会话、确认后自动继续、再次连接免重复确认、证书变化拦截/更新、设置页管理、深色撤销、390×600 单层滚动及临时凭据桌面交互。浅色/深色/窄窗口截图已实际查看。Computer Use 连接超时，未取得 NAS 界面操作证据。

0.1.12 x86 FPK 约 78 MiB，包树检查无链接/私有数据/特殊文件；FPK 内 manifest、Go 程序版本及原生会话证书校验代码均核对，SHA-256 校验通过。仅交付用户手动升级测试，本轮未在 NAS 安装，真实 Win10 登录与 Mac 兼容性仍待验收。

## 0.1.16 本机到远端剪贴板发送超时

Computer Use 在用户现有 Chrome 远程桌面页观察到已连接和“剪贴板发送超时，请重试”。核对固定版本 Guacamole 1.6.0 原生 RDP/VNC clipboard/blob 处理与 libguac 分派源码，确认这些成功路径不返回 ACK；旧前端在发送首块前等待 ACK，因此未传出文字。修复为直接顺序发送有大小限制的 UTF-8 块和 end，保留粘贴键在 end 后发送。

浏览器模拟端去除原先虚构的 clipboard/blob ACK 后，8 项 Playwright 流程通过。Node 6 项测试（含空文本、中文/emoji 跨块且完全无 ACK）、前端类型/构建、Go race 回归通过。0.1.16 x86 FPK 构建和包树检查通过，包内 manifest 版本及 SHA-256 已核对。未安装到 NAS，未据本地模拟测试宣称真实 Windows/Mac 端到端粘贴通过。

## 0.1.18 Mac Control+V

Control+V 在 Mac 不产生原生 paste 事件，原处理却统一拦截并等待 paste。现单独处理 Mac 的逻辑 Control+V：在按键手势中读取本机文字，顺序发送剪贴板和远端粘贴键；权限受限时仍发送远端粘贴键，不覆盖远端已有剪贴板，并提示可用 Command+V 或面板。异步读取检查剪贴板版本和活动焦点，长按不重复发送。

10 项浏览器回归通过，新增真实浏览器 Control+V（不注入 paste 事件）的文字发送顺序和权限拒绝后手动发送文字粘贴两项用例；隧道远端为模拟。前端类型/构建、Node 6 项、Go race、FPK 包树检查通过。包内 manifest、Linux amd64 程序 --version 和 SHA-256 核对为 0.1.18。Computer Use 选择用户当前远程桌面页超时，未取得该轮页面操作或实机验收证据；未安装 NAS。

## 0.1.49 FreeRDP 升级后连接超时诊断（2026-11-17）

通过既有 fnOS OAuth 会话直接确认 NAS 为 0.1.48/running；最新 server.log 止于 22:24:18 启动记录，NLA crypto 自检通过，没有本次连接阶段记录。用户此前已确认清晰优先仍超时，不能据此归因为 GFX 或密码错误。Computer Use 初始化超时，未操作用户页面。

0.1.49 纳入此前尚未交付的原生阶段标记，并增加应用到 guacd 的握手开始、失败和就绪记录，以及 guacd 创建客户端/插件加载失败分类。仅输出固定枚举及服务端会话 ID，不输出原始错误、账号、密码或目标。

完整构建、Go race、前端类型检查/构建、Node 单元测试及原生证书回归通过。新包导出的实际私有引擎在隔离 x86 容器中通过 NLA 与 GFX 持续画面测试，八个原生阶段标记均出现；此为合成 FreeRDP shadow 服务，不能替代用户 Windows/fnOS 验收。包内 manifest、x86 ELF 架构和 SHA-256 已核对。日志位于 build/rdp-full-qa/package-0.1.49.log 和 diagnostic-shadow/。

## 0.1.50 / 0.1.51 认证等待定位及修复

0.1.49 和 0.1.50 均经应用中心原位升级成功（status=2/progress=100），实际版本与运行状态核对。0.1.49 的实机重试在 23:08:24 接受证书，23:09:03 才完成登录和启动绘制。0.1.50 开启 FreeRDP 自身 DEBUG 阈值后，23:14:06 进入 Negotiate，23:14:49 才完成凭据准备、选择 NTLM，并在同秒完成 NLA、登录和绘制。用户确认升级前可登录，当前为邮箱用户名、域留空的微软账户。应用 30 秒超时早于这次 43 秒的等待完成。

新增 patch-nla-account-auth.py：空域且无 DOMAIN\user 前缀、或明确 MicrosoftAccount 的账户直接使用 none,ntlm；显式域保持 Negotiate。NLA、密码与证书策略不变。隔离无外网的 FreeRDP shadow 测试使用一次性邮箱形式账户和不响应的本地 KDC：旧引擎停在凭据获取并返回 776；新包引擎完成 NLA 与持续 GFX 图像接收，且 KDC 未收到请求。显式 QA.INVALID 域的对照仍保留 Kerberos 路径并在故障 KDC 上失败，没有被强制改为 NTLM。此对照不是 AD 服务器认证成功测试。

0.1.51 完整构建、Go race、前端类型/构建、Node 单测、原生证书回归、145 个 x86 ELF 检查及包内 manifest/SHA-256 核对通过。回归日志位于 build/rdp-full-qa/kdc-before.log、kdc-after-final.log 和 kdc-explicit-domain.log。

### 0.1.51 NAS 实机验收

应用中心原位升级任务 status=2/progress=100；实际安装 0.1.51、状态 running，启动 NLA crypto 自检通过。用户保持原邮箱用户名、空域及密码重连，日志在 23:20:52 同一秒记录证书接受、NLA_CREDENTIALS_READY、NLA_NTLM_SELECTED、NLA_AUTH_COMPLETE、RDP_LOGIN_COMPLETE 与 RDP_RENDER_STARTED，消除了此前 43 秒的凭据准备等待。用户明确确认“已正常连接，能看到桌面”。本轮未据此扩大为所有域账户、所有 Windows 版本或长时间稳定性的验证。

## 0.1.52 调试日志清理

移除原生 RDP_STAGE 补丁、NTLM 路径提示及应用握手开始/成功日志；握手失败和原有脱敏错误分类继续保留。FreeRDP 日志阈值固定为 WARN，guacd 的 debug 管道仅用于接收其转发的底层警告/错误，应用仍丢弃白名单之外的文本。

Go race、前端类型/构建、Node 单测及完整原生构建通过。诊断回归确认连接进度消息被丢弃，继承 WLOG_LEVEL=DEBUG 时子进程仍使用 WARN，必要错误仍穿过过滤管道。六类证书回归在 WARN 下通过。隔离无网络容器中，0.1.52 导出引擎以邮箱账户和黑洞 KDC 完成 NLA/GFX 持续画面测试，KDC 无请求，原生阶段标记及 FreeRDP 认证进度日志均未出现。

FPK 内 manifest 与服务程序版本 0.1.52、145 个 amd64 ELF、包 SHA-256 已核对；检查包内服务及 RDP 插件，阶段标记已移除，WARN 设置与 none,ntlm 修复保留。产物 dist/remote-desktop-0.1.52-x86.fpk。本轮仅重新打包，未安装 NAS。日志 build/rdp-full-qa/package-0.1.52.log、quiet-0.1.52.log。

### 0.1.52 NAS 原位升级

按用户要求，经应用中心上传解析与约束检查后从 0.1.51 原位升级至 0.1.52，保留现有安装卷与数据卷。升级任务 status=2/progress=100；实际应用版本为 0.1.52、running/isOpen=true。23:36:08 服务启动日志确认 0.1.52 监听应用 Socket，NLA crypto self-test PASS。升级前的阶段诊断仍在历史日志中，新版本启动后未见阶段诊断。本轮验证升级与启动，未代替用户操作 Windows 重连。

## 0.2.0 打包与 NAS 原位升级（2026-11-18）

整合壁纸默认启用及会话开关、设备图标选择和 NAS 配置持久化、连接表单与菜单精简、侧栏版本展示、窄屏会话设置修复。保留 0.1.52 的 FreeRDP 微软账户认证修复和日志级别。

完整 `scripts/build.sh amd64` 通过：Vue 类型检查与构建、Node 单测、Go race、包目录及 145 个 amd64 ELF 校验通过。包内 manifest 和服务程序 `--version` 均为 0.2.0；SHA-256：`64c947b14841b63511aae44011bdee78739bbac371f3ccd79dab51c190def055`。产物 `dist/remote-desktop-0.2.0-x86.fpk`。

应用中心上传解析、系统约束及升级信息检查通过，沿用原安装卷与数据卷，从 0.1.52 原位升级。任务终态 status=2/progress=100；实际安装版本 0.2.0，status=running、isOpen=true。启动日志确认 NLA crypto self-test PASS，0.2.0 监听应用 Socket。本轮未操作实际 Windows 重连；此前新增功能已完成本地浏览器模拟验证。
