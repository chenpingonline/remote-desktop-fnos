# 远程桌面 for fnOS

通过浏览器管理个人连接，使用 RDP 访问 Windows 桌面，或通过密码认证连接 VNC 桌面。应用基于 Go、Vue 3 和 Apache Guacamole，在飞牛 NAS 上原生运行，无需安装 Java、Node.js 或 Docker。

当前为开发预览。各版本改动见[更新日志](CHANGELOG.md)，设备兼容性与手测步骤见[兼容性说明](docs/COMPATIBILITY.md)。

## 主要功能

- 个人连接、单层分组、搜索与协议筛选；分组删除保留连接；不预置虚构连接。
- RDP / VNC 字段校验和参数白名单；服务端决定目标与协议参数，不接受任意透传。
- 个人凭据元数据；AES-256-GCM 加密账号、域和密码，随机 nonce，绑定归属、ID、类型和密钥版本；列表向凭据所属用户返回用户名和域，密码不回显，编辑时密码留空保留原值。
- 短期内存凭据（60 秒）及单会话领取；票据仅存 SHA-256 摘要（45 秒）；原子消费与状态转换。
- 会话状态、每人 2 / 总共 4 默认配额、30 秒建连超时、待连接过期、取消、幂等关闭、空闲与最长时长限制。管理员可调整配额和时长。
- Guacamole 按参数名握手、Unicode 码点流式解析、内部 UUID / ping、消息大小与写入超时限制。
- 官方 JS 1.6.0 桌面画布、多会话标签、重新连接、缩放、全屏、特殊键与失焦释放；显式 UTF-8 文本剪贴板。
- Windows RDP 跨会话文件剪贴板。两个连接分别启用后，在 A 复制文件，切到 B 的资源管理器即可粘贴。每次最多 64 个普通文件、合计 256 MiB。
- Unix Socket 网关身份；独立开发认证；同源检查、CSRF、对象归属和版本冲突处理。
- 私有 guacd 前台进程组管理，通过应用私有 Unix Socket 通信，支持有限次数的退避重启；引擎退出终止相关会话。
- SQLite WAL、版本化结构、单实例锁、启动遗留会话中断、配对密钥的一致性备份、操作审计。

没有接入共享连接、音频、多屏、ARD 账号认证或 NAS 注销事件。剪贴板兼容性、中文输入、最低 fnOS 版本和长期资源稳定性仍待实机验证。当前单主密钥 `key1`，自动密钥轮换尚未实现。

## 本地启动

需要 Go（见 go.mod）、Node.js 22+。设置仅用于本地的随机访问令牌，例如在终端执行 `export RD_DEV_TOKEN="$(openssl rand -hex 32)"`，保留在当前终端环境，然后运行：

```bash
./scripts/dev.sh
```

打开 http://127.0.0.1:18080/app/remote-desktop/ ，输入该环境变量的值。服务仅监听回环，忽略任何客户端 `X-Trim-*` 身份。开发模式下用户固定为本地开发者。令牌不写入 URL、日志或浏览器持久存储；会话 Cookie 为 HttpOnly / SameSite=Strict。开发数据保存在 `.data`，主密钥默认 `.data/master.key`，请一起备份。

没有 guacd 时仍可管理连接，桌面连接会明确失败。需要本地协议测试时，显式指定受信任的 `--guacd 127.0.0.1:14822`；生产网关模式必须提供 `--engine /absolute/path/guacd`，由应用启动。后端通过应用账号专用目录（0700）中的 `engine/guacd.sock`（0600）访问引擎。

不要对开发端口配置公网转发。生产 Unix Socket 模式只在 fnOS 可信网关后使用，页面通过当前 NAS HTTP 或 HTTPS 地址自动初始化。

## 构建 fnOS 安装包

构建需要 Docker、fnpack 1.2.3、Go 和 Node.js。版本号取自 `packaging/remote-desktop/manifest`。

```bash
./scripts/build.sh amd64 # x86_64，默认架构
./scripts/build.sh arm64 # ARM64
```

脚本按目标架构编译 Go 后端、Guacamole 原生引擎及依赖，并检查包内 ELF 架构。安装包和对应的 SHA-256 校验文件输出到 `dist/`，文件名分别为 `remote-desktop-<版本>-x86.fpk` 和 `remote-desktop-<版本>-arm.fpk`。ARM64 包仍待 NAS 实机验证。

安装不需要填写访问来源、IP、域名或网关 GID。应用通过统一网关获得 NAS 登录身份，页面在当前访问地址下初始化绑定用户和来源的令牌，支持 HTTP／HTTPS 的多个 IP/域名及代理改写内部 Host。Socket 使用包用户默认组和 `0660` 权限。旧版来源/GID 文件仅作为回滚材料保留，不参与启动。

## 验证

```bash
go test -race ./...
go vet ./...
npm --prefix web run build
python3 scripts/test-relay-ui.py
```

浏览器集成测试需要 Playwright Chromium。测试脚本使用独立的本地数据和令牌启动 Go 服务，并通过模拟远程桌面协议验证界面流程。

## 运维与恢复

- 配置 / 密钥：`TRIM_PKGETC`；数据库 / 日志 / 备份：`TRIM_PKGVAR`；程序、私有库、静态文件和 Socket：`TRIM_APPDEST`。
- `cmd/main start|stop|status` 串行执行；重复停止成功；status 运行返回 0、停止返回 3。进程 PID 与 Linux `/proc` 可执行路径、启动时间一起校验，健康检查使用本应用 Socket。
- 升级前停止服务，执行 `VACUUM INTO` 一致性备份，并配对保存主密钥、来源配置、GID 和版本。备份目标空间不足或失败时升级应中止。
- 手动离线备份：停止服务后运行 `bin/remote-desktop --data "$TRIM_PKGVAR" --key "$TRIM_PKGETC/master.key" --origin "https://实际NAS来源" --backup /新的私有备份目录`。数据锁禁止对运行中的数据库重复启动维护进程。
- 恢复时先停止应用，备份当前数据，再将同一备份中的 `app.db`、`master.key`、来源与 GID 分别恢复到业务和配置目录，确认所有权及私有权限后启动。不要留下另一套数据库的 WAL / SHM 文件。旧版本回滚必须使用匹配的数据与密钥。
- 卸载向导默认保留应用数据。选择原存储空间重装时，安装脚本会从数据目录恢复主密钥，以便继续使用原连接和凭据。选择永久删除会同时清空配置、数据库、日志和应用内备份；如需日后恢复，请先将配套的数据库和主密钥备份到应用目录外的私有位置。
- 当前审计最多 10,000 条且保留 90 天；终态会话元数据保留 90 天；运行日志应由宿主管理轮转。进程崩溃后的完整进程树回收与 Go 自身自动恢复仍需 fnOS 验证。

## 第三方组件

Apache JS 从官方发布包生成，未改上游模块，末尾添加 ES 模块导出。版本、校验值和再生成脚本见 `scripts/vendor-client.py`；LICENSE / NOTICE 位于 `web/src/vendor`。原生构建输出保存 Apache 与 Debian 组件版权文件及已安装版本。详见 [docs/DEPENDENCIES.md](docs/DEPENDENCIES.md)。

## 相关文档

- [需求与设计](doc/fnos_remote_desktop_development_v1.0.md)
- [前端开发说明](web/README.md)
- [前端迁移记录](docs/FRONTEND_MIGRATION.md)
