# nfetcher

一个从 nhentai 按筛选条件抓取漫画并输出标准 CBZ 归档的工具，可配合不同的漫画阅读器或媒体库使用。

## 默认行为

- 时区：Asia/Shanghai
- 调度时间：每天 17:30
- 搜索条件：language:chinese
- 排序方式：popular-today
- 抓取页码：第 1 页
- 保留时长：最近 7 天
- 下载 URL 签发间隔：30s
- 全局去重：同一 gallery_id 只保留一份归档
- 输出格式：CBZ，通过官方 download endpoint 下载
- ComicInfo.xml：默认保留官方内容；启用 `NF_KOMGA_READ_LIST` 后额外写入 StoryArc / StoryArcNumber

## 使用发布镜像

公共镜像发布在 GHCR：

~~~text
ghcr.io/yangmoooo/nfetcher:latest
~~~

生产环境建议固定具体版本，例如 ghcr.io/yangmoooo/nfetcher:v0.1.0。

### 1. 准备

将 `compose.example.yaml` 和 `.env.example` 复制到部署目录，编辑配置即可。

`.env` 中的 `NF_NHENTAI_API_KEY` 为必填项；Bark 和 `NF_KOMGA_READ_LIST` 为可选配置。你的 Komga 部署可以设置：

```env
NF_KOMGA_READ_LIST=true
```

### 2. 拉取并检查镜像

~~~bash
docker compose pull nfetcher
docker compose config --quiet
~~~

### 3. 先执行 dry-run

~~~bash
docker compose run --rm nfetcher dry-run
~~~

它会检查目录权限、请求搜索接口、计算去重和待抓队列，不会下载归档。

### 4. 手动抓取一次或常驻运行

~~~bash
docker compose run --rm nfetcher run-once
docker compose up -d nfetcher
docker compose logs -f nfetcher
~~~

更新镜像时：

~~~bash
docker compose pull nfetcher
docker compose up -d --force-recreate nfetcher
~~~

确认运行版本：

~~~bash
docker inspect nfetcher --format '{{.Image}}'
docker compose images --no-trunc nfetcher
~~~

latest 是可变 tag；生产环境建议使用 vX.Y.Z。

## 输出目录

程序默认使用容器内 /nhentai，Compose 将它挂载到宿主机 ./output：

~~~text
./output/<title> - <gallery-id>.cbz
~~~

`NF_LIBRARY_DIR` 必须和 volume 的容器内目标一致；retention 使用 CBZ 文件修改时间：

~~~yaml
volumes:
  - ./output:/nhentai
~~~

## Komga 配置

只替换宿主机侧路径，容器内目标仍保持 /nhentai：

~~~yaml
volumes:
  - /komga/library/nhentai/_oneshots:/nhentai
~~~

Komga 库可以指向 /komga/library/nhentai，并在设置中启用 One-Shots 目录 _oneshots。

## 代理与 Bark

公共 Compose 默认不启用代理。需要时取消注释并填写：

~~~yaml
environment:
  HTTP_PROXY: http://host.docker.internal:7890
  HTTPS_PROXY: http://host.docker.internal:7890
~~~

必要时添加：

~~~yaml
extra_hosts:
  - "host.docker.internal:host-gateway"
~~~

Bark 配置写入 `.env` 中的 `NF_BARK_BASE_URL`、`NF_BARK_DEVICE_KEY` 和 `NF_BARK_SOUND`。

## 不使用 Docker

准备 Go 环境后，在源码目录执行：

~~~bash
set -a
. ./.env
set +a
export NF_LIBRARY_DIR=./output
go run ./cmd/nfetcher dry-run
go run ./cmd/nfetcher run-once
~~~

## 本地构建镜像

如需构建本地 linux/amd64 镜像：

~~~bash
docker build -t local/nfetcher:latest .
~~~

网络环境需要代理或镜像时，可通过 Docker build args 覆盖默认值。

## 许可证

MIT
