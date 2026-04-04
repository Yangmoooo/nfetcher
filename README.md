# nfetcher

一个用 Go 编写的 `nhentai` 抓取器，用来配合本地部署的 `Komga`：

- 每天定时抓取
- 条件为 `language:chinese + popular-today + page=1`
- 每本漫画保存为一个 `.cbz`
- 默认写入 `Komga` 常见的 `_oneshots` 目录
- 默认只保留最近 `7` 天的数据

## 默认行为

- 时区：`Asia/Shanghai`
- 调度时间：每天 `17:30`
- 搜索条件：`language:chinese`
- 排序方式：`popular-today`
- 抓取页码：第 `1` 页
- 保留天数：`7`
- 全局去重：同一 `gallery_id` 只保留一份归档
- 输出格式：`.cbz`，并附带基础 `ComicInfo.xml`
- `StoryArc`：`nhentai-popular | YYYY-MM-DD`
- `StoryArcNumber`：当天榜单排名

## 快速开始

### 1. 构建镜像

在项目根目录执行：

```bash
docker build -t local/nfetcher:latest .
```

如果构建阶段需要代理，可以直接传 build arg，例如：

```bash
docker build \
  --build-arg HTTP_PROXY=http://127.0.0.1:17890 \
  --build-arg HTTPS_PROXY=http://127.0.0.1:17890 \
  -t local/nfetcher:latest .
```

### 2. 先跑一次 `dry-run`

```bash
docker compose run --rm nfetcher dry-run
```

`dry-run` 会：

- 检查库目录、代理、通知等基础配置
- 请求搜索和详情接口
- 计算去重结果和本次待抓队列
- 不实际下载图片，也不会写入 `.cbz`

### 3. 手动实际抓取一次

```bash
docker compose run --rm nfetcher run-once
```

### 4. 常驻运行

```bash
docker compose up -d
docker compose logs -f nfetcher
```

默认 `RUN_MODE=daemon`，会按 `SCHEDULE_CRON` 定时执行。

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
BARK_BASE_URL=https://bark.example.com
BARK_DEVICE_KEY=xxxxxxxx
BARK_SOUND=paymentsuccess
```

说明：

- 每次任务结束只发送一条通知
- 通知里会带精确执行时间、统计摘要和失败的 gallery id

## 输出结构

默认容器内目录：

- `/nhentai-popular`

根目录 `compose.yaml` 默认映射到宿主机：

- `./library/nhentai/_oneshots`

生成后的实际结构类似：

```text
./library/nhentai/_oneshots/<title> - <gallery-id>.cbz
```

如果标题不可用，会回退成：

```text
./library/nhentai/_oneshots/<gallery-id>.cbz
```

补充说明：

- 文件名会保留 `gallery_id`
- 同一 `gallery_id` 在整个库里只会保留一份
- 每个 `.cbz` 都会写入基础 `ComicInfo.xml`
- `Title` 使用首选标题，不再写入 `Series`
- `StoryArc` 使用 `nhentai-popular | YYYY-MM-DD`
- 去重和 retention 都直接扫描现有 `.cbz`，不依赖额外状态文件
