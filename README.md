# nfetcher

一个用 Go 编写的 `nhentai` 抓取器

- 每天定时抓取
- 条件为 `language:chinese + popular-today + page=1`
- 每本漫画保存为一个 `.cbz`
- 按日期目录归档
- 只保留最近 `7` 天的数据
- 用来配合本地部署的 `Kavita`

## 默认行为

- 时区：`Asia/Shanghai`
- 调度时间：每天 `17:30`
- 搜索条件：`language:chinese`
- 排序方式：`popular-today`
- 抓取页码：第 `1` 页
- 保留天数：`7`
- 全局去重：同一 `gallery_id` 只保留一份归档
- 输出格式：`.cbz`，并附带基础 `ComicInfo.xml`

仓库内已经提供两套配置：

- `compose.yaml`：适合在源码目录里直接构建、调试和运行
- `deploy/compose.example.yaml`：适合把运行用的 `compose` 放到单独的 Docker 管理目录

## 快速开始

### 1. 可选：复制本地覆盖配置

如果你想改抓取时间、运行用户、代理或 Bark，可以先复制：

```bash
cp .env.example .env
```

不复制也可以直接运行；`compose.yaml` 已经带了默认值。

常改项主要是：

- `NFETCHER_USER`
- `SCHEDULE_CRON`
- `RETENTION_DAYS`
- `BUILD_HTTP_PROXY` / `BUILD_HTTPS_PROXY`
- `HTTP_PROXY` / `HTTPS_PROXY` / `NO_PROXY`
- `BARK_BASE_URL` / `BARK_DEVICE_KEY` / `BARK_SOUND`

### 2. 构建镜像

在源码目录执行：

```bash
docker build -t local/nfetcher:latest .
```

### 3. 先跑一次 `dry-run`

第一次建议先预检和预览待抓队列：

```bash
docker compose run --rm nfetcher dry-run
```

`dry-run` 会：

- 检查库目录、代理、通知等基础配置
- 请求搜索和详情接口
- 计算去重结果和本次待抓队列
- 不实际下载图片，也不会写入 `.cbz`

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

## 独立部署

如果你想把运行用的 `compose` 放到另一个 Docker 管理目录，推荐这样做：

1. 在源码目录构建镜像
2. 复制 `deploy/compose.example.yaml` 到部署目录
3. 在部署目录修改挂载路径、代理、Bark 和运行用户
4. 在部署目录执行 `docker compose up -d`

示例：

```bash
cd /srv/src/nfetcher
docker build -t local/nfetcher:latest .

mkdir -p /srv/docker/nfetcher
cp deploy/compose.example.yaml /srv/docker/nfetcher/compose.yaml

cd /srv/docker/nfetcher
docker compose up -d
```

`deploy/compose.example.yaml` 默认就是：

- `image: local/nfetcher:latest`
- `user: 1000:1000`
- 运行代理地址：`http://host.docker.internal:17890`
- 库目录挂载：`./library/nhentai:/library/nhentai-popular`

## 代理与 Bark

### 代理

如果你的代理运行在宿主机上，例如 `mihomo` 使用 host 网络模式，推荐使用：

```text
http://host.docker.internal:17890
```

构建阶段用：

- `BUILD_HTTP_PROXY`
- `BUILD_HTTPS_PROXY`
- `BUILD_NO_PROXY`

运行阶段用：

- `HTTP_PROXY`
- `HTTPS_PROXY`
- `NO_PROXY`

仓库里的 `compose.yaml` 和 `deploy/compose.example.yaml` 都已经包含：

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

- `/library/nhentai-popular`

源码目录 `compose.yaml` 默认映射到宿主机：

- `./library/nhentai`

生成后的实际结构类似：

```text
./library/nhentai/2026-04-04/<title> - <gallery-id>/<title> - <gallery-id>.cbz
```

如果标题不可用，会回退成：

```text
./library/nhentai/2026-04-04/<gallery-id>/<gallery-id>.cbz
```

补充说明：

- 文件名和目录名都会保留 `gallery_id`
- 同一 `gallery_id` 在整个库里只会保留一份
- 每个 `.cbz` 都会写入基础 `ComicInfo.xml`，供 `Kavita` 读取

