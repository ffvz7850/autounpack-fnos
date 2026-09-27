# AutoUnpack for fnOS

飞牛 fnOS 自动解压 FPK 应用。

## 功能

- 自动监控指定目录中的压缩包，多核并行解压
- 支持格式：ZIP/RAR/7Z/TAR/GZ/BZ2/XZ/ZST/ISO 等
- 解压成功后自动将压缩包归档到指定目录，失败的放入 failed/ 子目录
- 密码列表自动尝试解压加密压缩包
- 文件大小稳定检测（等下载完成再解压）
- 分卷压缩包识别（自动跳过非首卷）
- 磁盘空间预检
- 空闲自动休眠
- 任务取消/重试/删除
- Web 管理界面

## 构建

```bash
# 交叉编译
set GOOS=linux GOARCH=amd64 CGO_ENABLED=0
go build -ldflags "-s -w" -o autounpack-amd64 .

# 打包 FPK
fnpack build
```

## 版本

当前版本：1.5.1
