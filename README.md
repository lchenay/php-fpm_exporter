# Credits

All credits go to https://github.com/discordianfish/nginx_exporter and all [contributors]https://github.com/discordianfish/nginx_exporter/graphs/contributors of the orignal projet.

This is just an adaptation to php fpm

# PHP Fpm Exporter for Prometheus

This is a simple server that periodically scrapes fpm stats and exports them via HTTP for Prometheus
consumption.

To run it:

```bash
./fpm_exporter [flags]
```

Help on flags:
```bash
./fpm_exporter --help
```

## Getting Started
  * All of the core developers are accessible via the [Prometheus Developers Mailinglist](https://groups.google.com/forum/?fromgroups#!forum/prometheus-developers).

## Using Docker

```
docker pull lchenay/php-fpm_exporter

docker run -d -p 9113:9113 lchenay/php-fpm_exporter \
    -fpm.scrape_uri=http://172.17.42.1/fpm_status
```
