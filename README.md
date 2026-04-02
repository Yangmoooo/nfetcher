# nfetcher

一个用 Go 编写的 Docker 抓取器，用来配合本地部署的 `Komga`：

- 每天按 cron 定时抓取 `nhentai` 的 `language:chinese + popular-today + page=1`
- 每本漫画保存为一个 `.cbz`
- 按日期分目录输出到 `Komga` 可扫描的目录
- 只保留最近 `7` 天的数据

当前默认输出目录结构：

- `./data/nhentai-popular/2026-04-01/<title> - <gallery-id>.cbz`

如果标题不可用，会回退成：

- `./data/nhentai-popular/2026-04-01/<gallery-id>.cbz`

## 当前默认行为

- 时区：`Asia/Shanghai`
- 调度时间：每天 `18:00`
- 搜索条件：`language:chinese`
- 排序方式：`popular-today`
- 抓取页码：第 `1` 页
- 多本下载并发：`3`
- 详情并发：`5`
- 单本图片下载并发：`4`
- 全局限速：`4 RPS`
- 全局突发：`8`
- 保留天数：`7`
- 跨日期全局去重：同一 `gallery_id` 只保留一份归档
- 下载调度：按页数降序，优先启动大本以缩短总完工时间

源码目录里，常改项可以可选写进 `.env`，示例见 `.env.example`；其余默认值已经直接写在 `compose.yaml`。

## 目录说明

- 容器内归档目录：`/library/nhentai-popular`
- 宿主机默认映射目录：`./data/nhentai-popular`

`Komga` 应该扫描宿主机上的 `./data/nhentai-popular`，或你自行改过后的对应目录。

## 推荐部署方式：源码目录与部署目录分离

- 部署目录更干净，只保留镜像和运行配置
- 不会让统一 Docker 目录混进 Go 源码、sample JSON、spec/plan 文档
- 构建和运行职责分离

目录结构示例：

```text
/srv/src/nfetcher/                  # 源码仓库
/srv/docker/nfetcher/compose.yaml   # 部署用 compose
```

### 方案说明

- 在**源码目录**里负责构建镜像
- 在**统一 Docker 管理目录**里负责启动容器
- 部署 compose 使用 `image:`，不直接使用 `build:`
- 源码仓库里的 `compose.yaml` / `.env.example` 仍然保留，主要用于**本地构建、调试和一次性试跑**
- `deploy/compose.example.yaml` 则是给**统一 Docker 管理目录**准备的部署模板

这样一来：

- 源码更新后，你只需要重新 build 镜像
- 部署目录只需要 `docker compose up -d`

### 源码目录配置和部署目录配置有什么区别

源码仓库里的文件：

- `compose.yaml`
- `.env.example`（可选）

更偏向“一体化本地使用”：

- 会在当前源码目录里直接 `build`
- 适合开发、调试、一次性试跑
- 默认值已经直接写在 `compose.yaml`
- `.env.example` 只保留常改项，例如抓取时间、保留天数、速率限制、代理和运行用户

部署目录模板：

- `deploy/compose.example.yaml`

更偏向“统一运维目录使用”：

- 直接使用已经构建好的 `image: local/nfetcher:latest`
- 不再从部署目录发起 `build`
- 默认值直接写在 compose 里，拿过去就能用
- 更适合和你现有的其他容器一起管理
- 需要改参数时，直接编辑部署目录里的 `compose.yaml`

### 1. 在源码目录构建镜像

先进入源码目录：

```bash
cd /srv/src/nfetcher
```

如果你**不需要代理**：

```bash
docker build -t local/nfetcher:latest .
```

如果你在中国大陆网络环境里构建，当前 `Dockerfile` 已经默认使用：

- Go 模块：`https://goproxy.cn,direct`
- Go 校验：`sum.golang.google.cn`
- apt 镜像：`https://mirrors.tuna.tsinghua.edu.cn`

也就是说，即使你不额外传这些参数，构建阶段也已经优先走国内下载加速。

如果你**需要代理**，并且你的 `mihomo` 使用的是 **host 网络模式**，那它本质上就是一个“宿主机上的代理服务”。

这时建议优先使用：

- `http://host.docker.internal:17890`

如果你的构建环境里 `host.docker.internal` 不可用，再改成你宿主机实际可达的地址，例如：

- `http://192.168.1.10:17890`

构建示例：

```bash
docker build \
  --add-host host.docker.internal:host-gateway \
  --build-arg HTTP_PROXY=http://host.docker.internal:17890 \
  --build-arg HTTPS_PROXY=http://host.docker.internal:17890 \
  -t local/nfetcher:latest .
```

如果上面的主机名在你的构建环境里不能解析，就把它替换成宿主机实际 IP：

```bash
docker build \
  --add-host host.docker.internal:host-gateway \
  --build-arg HTTP_PROXY=http://<HOST_IP>:17890 \
  --build-arg HTTPS_PROXY=http://<HOST_IP>:17890 \
  -t local/nfetcher:latest .
```

如果你只想覆盖构建镜像加速，也可以显式传入：

```bash
docker build \
  --build-arg GOPROXY=https://goproxy.cn,direct \
  --build-arg GOSUMDB=sum.golang.google.cn \
  --build-arg APT_DEBIAN_MIRROR=https://mirrors.tuna.tsinghua.edu.cn/debian \
  --build-arg APT_SECURITY_MIRROR=https://mirrors.tuna.tsinghua.edu.cn/debian-security \
  -t local/nfetcher:latest .
```

### 2. 在部署目录放置 compose 文件

部署目录里的 `compose.yaml` 可以写成这样：

```yaml
services:
  nfetcher:
    image: local/nfetcher:latest
    container_name: nfetcher
    restart: unless-stopped
    user: "1000:1000"
    environment:
      TZ: Asia/Shanghai
      RUN_MODE: daemon
      SCHEDULE_CRON: "0 18 * * *"
      LIBRARY_DIR: /library/nhentai-popular
      RETENTION_DAYS: "7"
      SEARCH_QUERY: language:chinese
      SEARCH_SORT: popular-today
      SEARCH_PAGE: "1"
      GALLERY_CONCURRENCY: "3"
      DETAIL_CONCURRENCY: "5"
      PAGE_CONCURRENCY: "4"
      REQUEST_RPS: "4"
      REQUEST_BURST: "8"
      HTTP_TIMEOUT: 30s
      RETRY_MAX: "3"
      HTTP_PROXY: http://host.docker.internal:17890
      HTTPS_PROXY: http://host.docker.internal:17890
      NO_PROXY: ""
    volumes:
      - /srv/public/KomgaLibrary/nhentai:/library/nhentai-popular
    extra_hosts:
      - "host.docker.internal:host-gateway"
```

这里有几点：

- `image: local/nfetcher:latest` 表示直接使用你在源码目录构建好的镜像
- `/srv/public/KomgaLibrary/nhentai` 使用你当前 README 里已经确认的宿主机目录
- `extra_hosts` 用来让容器在运行时访问宿主机上的代理服务
- `user: "1000:1000"` 用来避免抓取结果以 `root` 身份写入宿主机目录
- 代理默认写成 `http://host.docker.internal:17890`，适合宿主机上运行 `mihomo` 的场景
- 当前默认是“平衡档”：`GALLERY_CONCURRENCY=3`、`DETAIL_CONCURRENCY=5`、`PAGE_CONCURRENCY=4`、`REQUEST_RPS=4`、`REQUEST_BURST=8`

如果你想直接拿现成模板，可以复制：

```bash
cp deploy/compose.example.yaml /srv/docker/nfetcher/compose.yaml
```

### 3. 按需直接修改部署 compose

对于你这种 `mihomo` 使用 **host 网络模式** 的场景，默认模板已经按推荐方式写好了，所以大多数情况下你**不需要再额外准备 env 文件**。

如果 `host.docker.internal` 在你的运行环境里不可用，就把 compose 里的这两行改成宿主机实际可达 IP：

```yaml
      HTTP_PROXY: http://<HOST_IP>:17890
      HTTPS_PROXY: http://<HOST_IP>:17890
```

如果你希望不是 `1000:1000`，直接修改 compose：

```yaml
    user: "1001:1001"
```

### 4. 在部署目录启动

```bash
cd /srv/docker/nfetcher
docker compose up -d
```

查看日志：

```bash
docker compose logs -f nfetcher
```

### 5. 更新流程

以后更新代码时，建议这样做：

1. 在源码目录拉取/修改代码
2. 在源码目录重新构建镜像
3. 回到部署目录重启服务

示例：

```bash
cd /srv/src/nfetcher
docker build -t local/nfetcher:latest .

cd /srv/docker/nfetcher
docker compose up -d
```

## `mihomo` 使用 host 网络时怎么理解

因为你的 `mihomo` 使用的是 **host 网络模式**，所以对 `nfetcher` 来说，它不是“另一个 bridge 网络里的容器服务”，而更像是：

- **宿主机上的一个代理端口**

这意味着：

- 运行时不要写 `http://mihomo:17890`
- 更推荐写：
  - `http://host.docker.internal:17890`
  - 或 `http://<HOST_IP>:17890`

如果你以后把 `mihomo` 改成普通 bridge 网络，并且和 `nfetcher` 加入同一个 Docker 网络，那时才更适合用容器名访问。

从职责上讲，`nfetcher` 一般**不需要**加入 `Traefik` 的反代网络；如果你只是觉得默认网络名字不好看，可以把 compose 项目名改掉，但没必要为了这个把它强行放进 `rproxy`。

另外，源码目录和部署目录现在都尽量采用同一思路：

- `compose.yaml` 里显式列出运行时 `environment`
- `.env` 只负责提供变量值
- 构建阶段专用变量只保留在源码目录，例如 `BUILD_HTTP_PROXY`

## 为什么现在还保留 `extra_hosts`

这里最容易混淆的是：`HTTP_PROXY` 只是一个环境变量值，例如：

```env
HTTP_PROXY=http://host.docker.internal:17890
```

这行配置只是告诉程序“代理地址叫什么”，**并不会自动让容器认识 `host.docker.internal` 这个主机名**。

因此仍然需要：

```yaml
extra_hosts:
  - "host.docker.internal:host-gateway"
```

它的作用是把：

- `host.docker.internal`

写进容器的 hosts 解析里，让运行容器真的能连到宿主机。

另外要注意，**运行时的 `extra_hosts` 和构建阶段不是一回事**：

- 服务级 `extra_hosts`：作用于 `docker compose up`
- `build.extra_hosts`：作用于 `docker compose build`

所以源码仓库里的 `compose.yaml` 现在两处都保留了 host 映射：

- build 阶段能在需要时访问宿主机代理
- run 阶段也能访问宿主机代理

如果你把代理地址直接写成：

- `http://172.17.0.1:17890`

那通常可以不依赖 `extra_hosts`。但 `172.17.0.1` 不如 `host.docker.internal` 语义清晰，而且并不是所有 Docker 环境都保证它固定不变。

## 全局去重规则

现在的去重规则是按整个库目录生效，而不只是“当天目录”：

- 抓取当天榜单后，会先扫描 `LIBRARY_DIR` 下已经存在的 `.cbz`
- 以 `gallery_id` 为唯一键
- 如果某一本已经存在于任意日期目录中，就跳过当天重复下载

例如：

- 今天榜单有 `25` 本
- 其中 `3` 本已经存在于前几天目录
- 那今天只会新增 `22` 本

这样可以避免同一作品在最近几天反复占用空间。

## 快速开始

### 1. 可选：复制环境变量模板

```bash
cp .env.example .env
```

这一步不是必须的；只有你想覆盖常改项时才需要。

`.env.example` 现在只保留这些本地经常会改的项：

- `TZ`
- `NFETCHER_USER`
- `SCHEDULE_CRON`
- `RETENTION_DAYS`
- `GALLERY_CONCURRENCY`
- `DETAIL_CONCURRENCY`
- `PAGE_CONCURRENCY`
- `REQUEST_RPS`
- `REQUEST_BURST`
- `BUILD_HTTP_PROXY` / `BUILD_HTTPS_PROXY` / `BUILD_NO_PROXY`
- `HTTP_PROXY` / `HTTPS_PROXY` / `NO_PROXY`

### 2. 如需代理或改常用参数，再改 `.env`

如果你本机已经有代理，**不要直接把宿主机的 `127.0.0.1:端口` 填给容器**，因为容器里的 `127.0.0.1` 指向的是容器自己。

如果代理运行在宿主机上，通常应该这样填：

```env
BUILD_HTTP_PROXY=http://host.docker.internal:17890
BUILD_HTTPS_PROXY=http://host.docker.internal:17890
HTTP_PROXY=http://host.docker.internal:17890
HTTPS_PROXY=http://host.docker.internal:17890
```

`compose.yaml` 已经加入：

- `host.docker.internal:host-gateway`

因此在大多数现代 Docker 环境里，这种写法可以让容器访问宿主机代理。

如果你不需要代理，可以把相关变量留空。

### 3. 构建镜像

```bash
docker compose build
```

### 4. 单次试跑

第一次建议先用一次性模式：

```bash
docker compose run --rm -e RUN_MODE=run-once nfetcher run-once
```

成功后，输出文件会出现在：

- `./data/nhentai-popular/<YYYY-MM-DD>/`

### 5. 常驻运行

确认单次试跑正常后，再启动常驻定时模式：

```bash
docker compose up -d
```

查看日志：

```bash
docker compose logs -f nfetcher
```

停止服务：

```bash
docker compose down
```

## 源码目录里建议放进 `.env` 的常改项

- `TZ=Asia/Shanghai`
- `NFETCHER_USER=1000:1000`
- `SCHEDULE_CRON=0 18 * * *`
- `RETENTION_DAYS=7`
- `GALLERY_CONCURRENCY=3`
- `DETAIL_CONCURRENCY=5`
- `PAGE_CONCURRENCY=4`
- `REQUEST_RPS=4`
- `REQUEST_BURST=8`
- `BUILD_HTTP_PROXY` / `BUILD_HTTPS_PROXY` / `BUILD_NO_PROXY`
- `HTTP_PROXY` / `HTTPS_PROXY` / `NO_PROXY`

其余默认值已经写在 `compose.yaml` 里；如果你真的想改搜索条件、页码、并发或挂载路径，直接编辑 `compose.yaml` 即可。

## Komga 使用建议

给 `Komga` 新建一个单独的库，指向：

- `/srv/public/KomgaLibrary/nhentai`

因为抓取结果按日期分目录，所以在 `Komga` 里更接近“最近几天的热门归档”，同时你打开最新日期目录时，本质上就是在看当天热门。

## 命名规则

标题优先级：

1. `title.english`
2. `title.japanese`
3. `gallery_id`

文件名规则：

- 有标题：`<title> - <gallery-id>.cbz`
- 无标题：`<gallery-id>.cbz`

## 当前已验证内容

我已经在当前环境里验证过：

- `go test ./...` 可以通过编译检查
- `docker compose config` 可以正确展开配置

但真正的在线抓取是否成功，还取决于你本机网络、代理可达性，以及目标站点当下的访问状态。

## 常见问题

### 1. `docker compose build` 里访问网络失败

优先检查：

- `BUILD_HTTP_PROXY`
- `BUILD_HTTPS_PROXY`

如果你用的是宿主机本地代理，不要写成：

- `http://127.0.0.1:17890`

而应该优先尝试：

- `http://host.docker.internal:17890`

### 2. 运行时抓不到数据或图片下载失败

优先检查：

- `HTTP_PROXY`
- `HTTPS_PROXY`
- `REQUEST_RPS`
- `REQUEST_BURST`

如果站点不稳定，可以先保持低速率，不要把并发和 RPS 一次拉得太高。

### 3. 想改抓取时间

如果你使用源码目录的 `.env`，直接修改：

```env
SCHEDULE_CRON=0 19 * * *
```

然后重启服务：

```bash
docker compose up -d --build
```

如果你使用部署模板，就直接改部署目录 `compose.yaml` 里的 `SCHEDULE_CRON`。

### 4. 想手动补跑一次

```bash
docker compose run --rm -e RUN_MODE=run-once nfetcher run-once
```

同一天已存在的 `.cbz` 会被跳过，不会重复生成。

### 5. 出现 `.cbz.part: permission denied`

如果你是从旧版本切过来，之前又曾经用 `root` 运行过容器，库目录里可能残留过旧的 `.tmp` 目录或 `root` 拥有的临时文件。

当前版本已经改成把临时归档直接写到目标日期目录中，不再依赖库根目录的 `.tmp`。如果你仍然看到旧残留，可以在宿主机上手动清理它：

```bash
rm -rf /srv/public/KomgaLibrary/nhentai/.tmp
```

如果目标日期目录本身也是旧的 `root` 权限，再把整个库目录修正成你的媒体用户，例如：

```bash
chown -R 1000:1000 /srv/public/KomgaLibrary/nhentai
```
