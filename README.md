# SlateSSH

SlateSSH is a lightweight web SSH and SFTP workspace rebuilt in Go.

## Structure
- `backend/` Go server, API, WebSocket SSH/SFTP, status polling
- `frontend/` static frontend served by the Go backend
- `Dockerfile` single-image deployment
- `docker-compose.yml` local container deployment

## Run locally
```bash
cd backend
go mod tidy
go run ./cmd/slatessh
```

Then open `http://localhost:3210`.

## Docker
```bash
docker compose up --build
```

## 界面与移动端

登录、主机目录、SSH 终端及文件管理采用深色等宽字体界面。PC 保留主机 / 终端 / 文件三栏；手机和平板使用主机抽屉与底部工具面板，支持横竖屏、刘海与主屏幕安全区。

- 快捷栏可横向滑动，键盘开关始终可见。ESC、Tab、方向键、Ctrl+C 直接发送终端控制字符；CTRL 对下一个字母生效。
- 软键盘打开时使用 Visual Viewport 调整终端高度。登录和表单保留原生输入、密码自动填充与滚动，未禁用双指缩放。
- 手机文件编辑使用原生文本框，PC 使用 Monaco。文件右侧的 `···` 可打开复制、下载等操作。
- 断开的 SSH 会话保留输出，并显示重新连接按钮。iOS 切换后台可能挂起网络，返回后请检查会话状态；长期任务建议在服务器使用 tmux / screen。

## PWA 与 iPhone Safari

生产环境需使用 **HTTPS**（本机 localhost 可用于开发）。在 Safari 分享菜单中选择「添加到主屏幕」，如有「作为 Web App 打开」选项，请保持开启。再次从主屏幕启动即可使用独立窗口。

Service Worker 位于 `/sw.js`，Go 服务直接以 JavaScript 类型提供该文件，并禁止长期缓存入口。离线时可加载应用外壳；SSH、SFTP、RDP 仍需网络。Service Worker 仅缓存清单中的公共静态资源，不缓存 API、文件传输和终端输出。已有的主机目录本地缓存仍由应用管理。

新版本就绪后显示更新提示，当前页面关闭全部会话后才允许更新，避免自动重载打断操作。部署修改时应同步修改 `frontend/sw.js` 的缓存版本，以及 `frontend/index.html` 和预缓存清单内 CSS / JS 的版本参数；HTML 与脚本作为同一版本安装。

## 界面验证

前端仍为无需打包的静态页面。Node 仅用于开发测试：

```bash
pnpm install
pnpm exec playwright install chromium webkit
pnpm test
cd backend
go test ./...
```

浏览器测试使用独立的静态服务和模拟 API / SSH 消息，不连接真实主机。覆盖 320–1440px、402×874 手机视口、横屏、登录、主机表单、控制字符、弹窗、键盘视口、PWA 安装及缓存隔离。

Windows Playwright WebKit 的断网模拟会在受 Service Worker 控制的请求上返回内部错误，因此该环境跳过离线重载一项；离线重载在 Chromium 验证。WebKit 测试使用浏览器引擎和模拟视口，不能替代 iPhone 真机验证。发布前请在 Safari 和主屏幕应用中复核：键盘开合 / 中文输入 / 密码填充、横竖屏安全区、快捷栏滑动与连按、后台返回、飞行模式启动与恢复联网。

应用图标可使用安装了 Pillow 的 Python 运行 `python scripts/generate_icons.py` 重新生成。
