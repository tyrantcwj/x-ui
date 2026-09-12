# x-ui

Multi-protocol, multi-user xray panel. This repository is a fork of [sing-web/x-ui](https://github.com/sing-web/x-ui) with a clean default Xray template (no bundled WARP or CN-blocking routes).

> [中文文档](./README.md)

# Features

- System status monitoring
- Support multi-user multi-protocol, multi-user on same port, web visualization operation
- Supported protocols: vmess, vless, trojan, shadowsocks, dokodemo-door, socks, http
- Support vless / trojan reality
- Support for configuring more transport configurations
- Traffic statistics, limit traffic, limit expiration time
- Customizable xray configuration templates
- Support https access panel (self-provided domain + ssl certificate)
- Support one-click SSL certificate application and automatic renewal
- More advanced configuration items, see panel for details

# Installation & Upgrade

```
bash <(wget -qO- https://raw.githubusercontent.com/tyrantcwj/x-ui/main/install.sh)
```

## Manual installation & upgrade

1. First download the latest tarball from this project's [Releases](https://github.com/tyrantcwj/x-ui/releases), usually choose the `amd64` architecture
2. Then upload the tarball to the `/root/` directory on the server and login to the server with the `root` user

> If your server CPU architecture is not `amd64`, replace `amd64` in the command with another architecture

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

## Installing with Docker

> This Docker tutorial and Docker image is provided by [Chasing66](https://github.com/Chasing66)

1. Installing Docker

```shell
curl -fsSL https://get.docker.com | sh
```

2. Install x-ui

```shell
mkdir x-ui && cd x-ui
docker run -itd --network=host \
    -v $PWD/db/:/etc/x-ui/ \
    -v $PWD/cert/:/root/cert/ \
    --name x-ui --restart=unless-stopped \
    misakablog/x-ui:latest
```

> Build your own image

```shell
docker build -t x-ui .
```

## TG bot usage

> This feature and tutorial is provided by [FranzKafkaYu](https://github.com/FranzKafkaYu)

X-UI supports daily traffic notification and panel login reminder via Tg bot.
The specific application tutorial can be found in this [blog link](https://coderfan.net/how-to-use-telegram-bot-to-alarm-you-when-someone-login-into-your-vps.html)

Instructions: set the bot-related parameters in the background of the panel, including

- Tg Bot Token
- Tg Bot ChatId
- Tg Bot cycle running time, using crontab syntax

Reference syntax:

- `30 * * * * *` notify on the 30th second of every minute
- `@hourly` hourly notification
- `@daily` daily notification (at 00:00)
- `@every 8h` notify every 8 hours

TG notification content:

- Node traffic usage
- Panel login reminder
- Node expiration reminder
- Traffic alert reminder

More features are being planned...

## Recommended Systems

- CentOS 7+
- Ubuntu 16+
- Debian 8+

## Credits

- vaxilu's x-ui project: https://github.com/vaxilu/x-ui
- qist's xray-ui project: https://github.com/qist/xray-ui
- MHSanaei's 3x-ui project: https://github.com/MHSanaei/3x-ui
- sing-web's x-ui project: https://github.com/sing-web/x-ui

## Disclaimer

- This program is for learning and understanding only, not for profit. Please delete it within 24 hours after downloading. It is not for any commercial use. Text, data and images are copyrighted; if reproduced, please indicate the source.
- Use of this program is subject to the laws and regulations of the country where the server is deployed and the country where the user is located. The author of the program is not responsible for any misconduct of the user.
