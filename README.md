# nfetcher

一个用 Go 编写的 `nhentai` 抓取器，输出标准 `.cbz` 归档，可配合不同的漫画阅读器或媒体库使用：

- 每天定时抓取
- 条件为 `language:chinese + popular-today + page=1`
- 每本漫画保存为一个 `.cbz`
- 通用示例默认写入 `./output`，容器内目录为 `/nhentai`
- Compose 示例中注释提示 Komga 的 `_oneshots` 目录和代理配置
- 默认只保留最近 `7` 天的数据

## 默认行为

- 时区：`Asia/Shanghai`
- 调度时间：每天 `17:30`
- 搜索条件：`language:chinese`
- 排序方式：`popular-today`
- 抓取页码：第 `1` 页
- 保留天数：`7`
- download URL 签发间隔：`30s`
- 全局去重：同一 `gallery_id` 只保留一份归档
- 输出格式：`.cbz`，通过官方 download endpoint 下载
- `ComicInfo.xml`：保留官方内容，只补 `StoryArc` / `StoryArcNumber`

## 快速开始

### 1. 构建镜像

在项目根目录执行：

```bash
docker build -t local/nfetcher:latest .
```

默认构建会：

- 以 `linux/amd64` 为目标架构构建二进制
- 默认使用官方 Go 模块代理与 Debian APT 镜像源

如果网络环境需要中国镜像或代理，可以覆盖 build args，例如：

```bash
docker build \
  --build-arg HTTP_PROXY=http://127.0.0.1:17890 \
  --build-arg HTTPS_PROXY=http://127.0.0.1:17890 \
  --build-arg GOPROXY=https://goproxy.cn,direct \
  --build-arg GOSUMDB=sum.golang.google.cn \
  --build-arg APT_DEBIAN_MIRROR=http://mirrors.tuna.tsinghua.edu.cn/debian \
  --build-arg APT_SECURITY_MIRROR=http://mirrors.tuna.tsinghua.edu.cn/debian-security \
  -t local/nfetcher:latest .
```

其余参数同理。

### 2. 配置 API key 和 Compose

在 nhentai account settings 中创建 API key，然后将通用 Compose 示例和环境变量模板复制到你的部署目录：

```bash
cp /path/to/nfetcher/compose.example.yaml /path/to/deployment/compose.yaml
cp /path/to/nfetcher/.env.example /path/to/deployment/.env
chmod 600 /path/to/deployment/.env
```

然后编辑部署目录中的 `.env`，填写 `NF_NHENTAI_API_KEY`；如果启用 Bark，再填写 `NF_BARK_BASE_URL` 和 `NF_BARK_DEVICE_KEY`。

nfetcher 会使用官方 `POST /api/v2/galleries/{id}/download?format=cbz` 下载归档，不再逐页抓取 CDN 图片。

默认 `DOWNLOAD_ISSUE_INTERVAL=30s`，用于匹配官方 `10/5min` 的 download URL 签发限流。

真实 `.env` 不要提交到 Git；`.env.example` 只包含空值和默认值模板。

### 3. 先跑一次 `dry-run`

```bash
docker compose run --rm nfetcher dry-run
```

`dry-run` 会：

- 检查库目录、代理、通知等基础配置
- 请求搜索接口
- 计算去重结果和本次待抓队列
- 不实际下载归档，也不会写入 `.cbz`

### 4. 手动实际抓取一次

```bash
docker compose run --rm nfetcher run-once
```

### 5. 常驻运行

```bash
docker compose up -d
docker compose logs -f nfetcher
```

默认 `RUN_MODE=daemon`，会按 `SCHEDULE_CRON` 定时执行。

### Komga 配置

如果使用 Komga，只替换 Compose 的宿主机路径，容器内目标仍保持 `/nhentai`：

```yaml
- ./library/nhentai/_oneshots:/nhentai
```

`Komga` 侧建议这样配：

- 库根目录指向上层目录，例如 `./library/nhentai`
- 在库设置里启用 One-Shots 目录，并填写 `_oneshots`

如果没有启用 One-Shots 处理，`Komga` 会把同一目录下的多个 `.cbz` 当成一个普通系列导入。

## 代理与 Bark

### 代理

如果你的代理运行在宿主机上，例如 `mihomo` 使用 host 网络模式，推荐使用：

```text
http://host.docker.internal:17890
```

```yaml
extra_hosts:
  - "host.docker.internal:host-gateway"
```

### Bark

如果你想在任务结束后收到一条 Bark 通知，设置下面几个变量即可：

```env
NF_BARK_BASE_URL=https://bark.example.com
NF_BARK_DEVICE_KEY=xxxxxxxx
NF_BARK_SOUND=healthnotification
```

说明：

- 每次任务结束只发送一条通知
- 通知里会带精确执行时间、统计摘要和失败的 gallery id

## 不使用 Docker

程序也可以直接运行在宿主机上。准备 Go 环境后，在源码目录执行：

```bash
set -a
. ./.env
set +a
export NF_LIBRARY_DIR=./output
go run ./cmd/nfetcher dry-run
go run ./cmd/nfetcher run-once
```

`NF_LIBRARY_DIR` 默认是 `/nhentai`；裸机运行时建议改成相对路径或宿主机上的绝对路径。

## 输出结构

默认容器内目录：

- `/nhentai`（默认输出目录，可通过 `NF_LIBRARY_DIR` 修改）

通用 Compose 示例映射到宿主机：

- `./output`

生成后的实际结构类似：

```text
./output/<title> - <gallery-id>.cbz
```

如果标题不可用，会回退成：

```text
./output/<gallery-id>.cbz
```

Komga Compose 示例则使用：

```text
./library/nhentai/_oneshots/<title> - <gallery-id>.cbz
```

补充说明：

- 文件名会保留 `gallery_id`
- 同一 `gallery_id` 在整个库里只会保留一份
- 每个 `.cbz` 保留官方 `ComicInfo.xml`，只补 `StoryArc` 和 `StoryArcNumber`
- `StoryArc` 使用 `YYYY-MM-DD`
- 去重和 retention 都直接扫描现有 `.cbz`，不依赖额外状态文件
