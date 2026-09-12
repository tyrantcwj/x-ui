# x-ui

支持多协议多用户的 xray 面板。本仓库基于 [sing-web/x-ui](https://github.com/sing-web/x-ui) 修改，默认 Xray 模板已去掉 WARP、屏蔽国内等额外配置。

> [English README](./README_EN.md)

# 功能介绍

- 系统状态监控
- 支持多用户多协议，单端口多用户，网页可视化操作
- 支持的协议：vmess、vless、trojan、shadowsocks、dokodemo-door、socks、http
- 支持 vless / trojan reality
- 支持配置更多传输配置
- 流量统计，限制流量，限制到期时间
- 可自定义 xray 配置模板
- 支持 https 访问面板（自备域名 + ssl 证书）
- 支持一键 SSL 证书申请且自动续签
- 更多高级配置项，详见面板

# 安装 & 升级

```
bash <(wget -qO- https://raw.githubusercontent.com/tyrantcwj/x-ui/main/install_CN.sh)
```

## 手动安装 & 升级

1. 首先从本仓库 [Releases](https://github.com/tyrantcwj/x-ui/releases) 下载最新压缩包，一般选择 `amd64` 架构
2. 然后将这个压缩包上传到服务器的 `/root/` 目录下，并使用 `root` 用户登录服务器

> 如果你的服务器 CPU 架构不是 `amd64`，自行将命令中的 `amd64` 替换为其他架构

```
cd /root/
rm x-ui/ /usr/local/x-ui/ /usr/bin/x-ui -rf
tar zxvf x-ui-linux-amd64.tar.gz
chmod +x x-ui/x-ui x-ui/bin/xray-linux-* x-ui/x-ui.sh
cp x-ui/x-ui.sh /usr/bin/x-ui
cp -f x-ui/x-ui.service /etc/systemd/system/
mv x-ui/ /usr/local/
systemctl daemon-reload
systemctl enable x-ui
systemctl restart x-ui
```

## 使用 Docker 安装

> 此 Docker 教程与 Docker 镜像由 [Chasing66](https://github.com/Chasing66) 提供

1. 安装 Docker

```shell
curl -fsSL https://get.docker.com | sh
```

2. 安装 x-ui

```shell
mkdir x-ui && cd x-ui
docker run -itd --network=host \
    -v $PWD/db/:/etc/x-ui/ \
    -v $PWD/cert/:/root/cert/ \
    --name x-ui --restart=unless-stopped \
    misakablog/x-ui:latest
```

> 构建自己的镜像

```shell
docker build -t x-ui .
```

## TG 机器人使用

> 此功能与教程由 [FranzKafkaYu](https://github.com/FranzKafkaYu) 提供

X-UI 支持通过 Tg 机器人实现每日流量通知、面板登录提醒等功能，使用 Tg 机器人需要自行申请。
具体申请教程可以参考[博客链接](https://coderfan.net/how-to-use-telegram-bot-to-alarm-you-when-someone-login-into-your-vps.html)

使用说明：在面板后台设置机器人相关参数，具体包括

- Tg 机器人 Token
- Tg 机器人 ChatId
- Tg 机器人周期运行时间，采用 crontab 语法

参考语法：

- `30 * * * * *` 每一分的第 30 秒进行通知
- `@hourly` 每小时通知
- `@daily` 每天通知（凌晨零点整）
- `@every 8h` 每 8 小时通知

TG 通知内容：

- 节点流量使用
- 面板登录提醒
- 节点到期提醒
- 流量预警提醒

更多功能规划中...

## 建议系统

- CentOS 7+
- Ubuntu 16+
- Debian 8+

## 鸣谢

- vaxilu 的 x-ui 项目：https://github.com/vaxilu/x-ui
- qist 的 xray-ui 项目：https://github.com/qist/xray-ui
- MHSanaei 的 3x-ui 项目：https://github.com/MHSanaei/3x-ui
- sing-web 的 x-ui 项目：https://github.com/sing-web/x-ui

## 免责声明

- 本程序仅供学习了解，非盈利目的，请于下载后 24 小时内删除，不得用作任何商业用途，文字、数据及图片均有所属版权，如转载须注明来源。
- 使用本程序必须遵守部署服务器所在地、所在国家和用户所在国家的法律法规，程序作者不对使用者任何不当行为负责。
