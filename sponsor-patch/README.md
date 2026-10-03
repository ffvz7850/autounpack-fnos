# 赞助窗口补丁 (Sponsor Box Patch)

可直接嵌入任意网页/应用设置页的赞助收款码组件，含支付宝和微信收款码。

## 文件说明

| 文件 | 说明 |
|------|------|
| `sponsor-inline.html` | **推荐** — 完全自包含，CSS + base64 图片全部内嵌，复制粘贴即可用 |
| `sponsor.html` | HTML 片段，引用外部图片文件（需配合 `images/` 目录） |
| `sponsor.css` | 独立样式文件，适合已有的样式体系 |
| `images/alipay.jpg` | 支付宝收款码原图 |
| `images/wechat.png` | 微信收款码原图 |

## 使用方法

### 方式一：自包含版本（最简单）

打开 `sponsor-inline.html`，复制全部内容，粘贴到你的设置页面 HTML 中任意位置即可。无需任何外部依赖。

### 方式二：HTML + CSS 分离

1. 在页面 `<head>` 中引入样式：
   ```html
   <link rel="stylesheet" href="sponsor.css">
   ```
2. 将 `sponsor.html` 的内容粘贴到设置页面底部。
3. 确保 `images/` 目录与页面在同一级路径（或修改图片 src 路径）。

### 方式三：Go 后端内嵌（参考 autounpack 实现）

将 `sponsor-inline.html` 的内容作为字符串嵌入 Go 模板中，在渲染设置面板时输出。

## 自定义

- **修改文字**：编辑 `.sponsor-title` 和 `.sponsor-desc` 中的文本
- **修改二维码**：替换 `images/` 目录下的图片，或替换 base64 字符串
- **调整大小**：修改 `.sponsor-qr img` 的 `width` / `height`（默认 96px）
- **调整配色**：修改 `.sponsor-box` 的 `background` 和 `border`

## 效果预览

```
┌─────────────────────────────┐
│      制作不易，感谢支持       │
│    觉得好用请扫码赞助 ~       │
│                             │
│   [支付宝码]    [微信码]     │
│     支付宝        微信       │
└─────────────────────────────┘
```
