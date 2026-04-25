# Magic FRPc - 快速启动指南

## 项目初始化步骤

### 1. 安装 Wails v3 CLI

```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@latest
```

### 2. 安装 Go 依赖

```bash
go mod download
```

### 3. 安装前端依赖

```bash
cd frontend
npm install
```

### 4. 启动开发服务器

```bash
# 返回项目根目录
cd ..

# 启动 Wails 开发模式
wails3 dev
```

## 项目结构

```
magic_frpc/
├── main.go                    # 应用入口
├── go.mod                     # Go 模块定义
├── wails.json                 # Wails 配置
├── Makefile                   # 构建脚本
│
├── internal/                  # Go 内部包
│   ├── app/                   # Wails 应用绑定
│   │   ├── app.go
│   │   └── lifecycle.go
│   ├── config/                # 配置管理
│   │   └── manager.go
│   ├── frpc/                  # frpc 进程管理
│   │   └── process.go
│   ├── version/               # 版本管理
│   │   └── manager.go
│   ├── store/                 # 数据存储
│   │   └── sqlite.go
│   └── platform/              # 平台检测
│       └── platform.go
│
├── pkg/                       # 公共包
│   └── types/                 # 类型定义
│       ├── config.go
│       ├── process.go
│       ├── version.go
│       └── settings.go
│
└── frontend/                  # 前端代码
    ├── src/
    │   ├── lib/
    │   │   ├── api/           # API 封装
    │   │   ├── stores/        # 状态管理
    │   │   ├── i18n/          # 国际化
    │   │   ├── utils/         # 工具函数
    │   │   └── components/    # UI 组件
    │   ├── routes/            # 页面路由
    │   ├── App.svelte         # 根组件
    │   ├── main.ts            # 前端入口
    │   └── app.css            # 全局样式
    ├── package.json
    ├── vite.config.ts
    └── svelte.config.js
```

## 技术栈

| 组件 | 技术 | 版本 |
|------|------|------|
| 后端框架 | Wails v3 | alpha |
| 后端语言 | Go | 1.21+ |
| 前端框架 | Svelte 5 | 5.19+ |
| CSS 框架 | TailwindCSS 4 | 4.0+ |
| UI 组件 | daisyUI 5 | 5.0+ |
| 图标库 | lucide-svelte | 0.469+ |
| 国际化 | svelte-i18n | 4.0+ |
| 数据库 | SQLite | - |

## 常用命令

```bash
# 开发模式
make dev

# 构建应用
make build

# 安装依赖
make install-deps

# 清理构建产物
make clean

# 运行测试
make test
```

## 功能特性

- [x] 项目基础框架
- [x] Wails v3 + Svelte 5 集成
- [x] TailwindCSS 4 + daisyUI 5 样式
- [x] 暗黑模式支持
- [x] 国际化支持 (中/英)
- [ ] 配置管理模块
- [ ] frpc 进程管理
- [ ] 版本管理
