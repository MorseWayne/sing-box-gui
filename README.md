<div align="center">
  <img src="build/appicon.png" alt="Sing-Box-GUI" width="120">
  <h1>Sing-Box-GUI</h1>
  <p>一个基于 Vue3 和 Wails 开发的 Sing-Box 图形界面客户端。</p>
  <p>
    <a href="https://github.com/MorseWayne/sing-box-gui/blob/main/LICENSE"><img src="https://img.shields.io/github/license/MorseWayne/sing-box-gui" alt="License"></a>
  </p>
  <hr />
  <p>本项目参考并基于 <a href="https://github.com/GUI-for-Cores/GUI.for.SingBox">GUI.for.SingBox</a> 进行开发及优化。</p>
</div>

## 项目简介

Sing-Box-GUI 旨在提供一个简洁、高效且美观的 Sing-Box 核心管理界面。利用 Wails 框架结合 Go 的强大性能与 Vue3 的现代前端开发体验，实现跨平台的桌面支持。

## 编译安装

在开始编译之前，请确保您的开发环境已安装以下工具：

- **Go** (1.20+)
- **Node.js** (LTS)
- **pnpm** (`npm i -g pnpm`)
- **Wails CLI** (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)

### 编译步骤

1. **克隆仓库**

   ```bash
   git clone https://github.com/MorseWayne/sing-box-gui.git
   cd sing-box-gui
   ```

2. **前端依赖安装** (可选，wails build 过程中会自动执行)

   ```bash
   cd frontend
   pnpm install
   cd ..
   ```

3. **执行编译**

   ```bash
   # 直接编译生成可执行文件
   wails build
   ```

编译完成后，生成的可执行文件将位于 `build/bin` 目录下。

## 开源协议

本项目采用 [GPL-3.0 License](LICENSE) 开源协议。
