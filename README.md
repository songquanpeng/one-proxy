<p align="center">
  <a href="https://github.com/songquanpeng/one-proxy"><img src="https://raw.githubusercontent.com/songquanpeng/one-proxy/main/web/public/logo.png" width="150" height="150" alt="one-proxy logo"></a>
</p>

<div align="center">

# One Proxy

_✨ 轻松管理你的众多订阅，提供一个固定的订阅地址 ✨_

</div>

<p align="center">
  <a href="https://raw.githubusercontent.com/songquanpeng/one-proxy/main/LICENSE">
    <img src="https://img.shields.io/github/license/songquanpeng/one-proxy?color=brightgreen" alt="license">
  </a>
  <a href="https://github.com/songquanpeng/one-proxy/releases/latest">
    <img src="https://img.shields.io/github/v/release/songquanpeng/one-proxy?color=brightgreen&include_prereleases" alt="release">
  </a>
  <a href="https://hub.docker.com/repository/docker/justsong/one-proxy">
    <img src="https://img.shields.io/docker/pulls/justsong/one-proxy?color=brightgreen" alt="docker pull">
  </a>
  <a href="https://github.com/songquanpeng/one-proxy/releases/latest">
    <img src="https://img.shields.io/github/downloads/songquanpeng/one-proxy/total?color=brightgreen&include_prereleases" alt="release">
  </a>
  <a href="https://goreportcard.com/report/github.com/songquanpeng/one-proxy">
    <img src="https://goreportcard.com/badge/github.com/songquanpeng/one-proxy" alt="GoReportCard">
  </a>
</p>

<p align="center">
  <a href="https://github.com/songquanpeng/one-proxy/releases">程序下载</a>
  ·
  <a href="https://github.com/songquanpeng/one-proxy#部署">部署教程</a>
  ·
  <a href="https://github.com/songquanpeng/one-proxy/issues">意见反馈</a>
  ·
  <a href="https://one-proxy.vercel.app/">在线演示</a>
</p>

## 功能
为你的众多订阅地址提供一个固定的更新链接，方便你在各个设备上使用，避免切换订阅时要修改一众设备上的配置。

## 部署
### 基于 Docker 进行部署
执行：`docker run -d --restart always -p 3000:3000 -v /home/ubuntu/data/one-proxy:/data justsong/one-proxy`

数据将会保存在宿主机的 `/home/ubuntu/data/one-proxy` 目录。

### 手动部署
1. 从 [GitHub Releases](https://github.com/songquanpeng/one-proxy/releases/latest) 下载可执行文件或者从源码编译：
   ```shell
   git clone https://github.com/songquanpeng/one-proxy.git
   cd one-proxy/web
   npm install
   npm run build
   cd ..
   go mod download
   go build -ldflags "-s -w" -o one-proxy
   ````
2. 运行：
   ```shell
   chmod u+x one-proxy
   ./one-proxy --port 3000 --log-dir ./logs
   ```
3. 访问 [http://localhost:3000/](http://localhost:3000/) 并登录。初始账号用户名为 `root`，密码为 `123456`。

更加详细的部署教程[参见此处](https://iamazing.cn/page/how-to-deploy-a-website)。

## 配置
系统本身开箱即用。

你可以通过设置环境变量或者命令行参数进行配置。

等到系统启动后，使用 `root` 用户登录系统并做进一步的配置。

### 环境变量
1. `REDIS_CONN_STRING`：设置之后将使用 Redis 作为请求频率限制的存储，而非使用内存存储。
   + 例子：`REDIS_CONN_STRING=redis://default:redispw@localhost:49153`
2. `SESSION_SECRET`：设置之后将使用固定的会话密钥，这样系统重新启动后已登录用户的 cookie 将依旧有效。
   + 例子：`SESSION_SECRET=random_string`
3. `SQL_DSN`：设置之后将使用指定数据库而非 SQLite。
   + 例子：`SQL_DSN=root:123456@tcp(localhost:3306)/one-proxy`

### 命令行参数
1. `--port <port_number>`: 指定服务器监听的端口号，默认为 `3000`。
   + 例子：`--port 3000`
2. `--log-dir <log_dir>`: 指定日志文件夹，如果没有设置，日志将不会被保存。
   + 例子：`--log-dir ./logs`
3. `--version`: 打印系统版本号并退出。