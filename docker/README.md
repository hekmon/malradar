# MAL Radar

[![Github repo](https://img.shields.io/badge/Github-MALRadar-lightgrey?logo=github)](https://github.com/hekmon/malradar)

My Anime List new finished animes curator / notifier.

## How it works

Please see the [github readme](https://github.com/hekmon/malradar#readme).

### State & Cache

MALRadar stores its cache in a volume at `/var/lib/malradar`.

## Configuration

You will need to [bind mount](https://docs.docker.com/storage/bind-mounts/) the [config json file](https://github.com/hekmon/malradar/blob/master/config.json). An extended example configuration file can be found in the [README](https://github.com/hekmon/malradar#configuration).

## Run it

### Letting docker handles the state volume

```bash
docker run --mount type=bind,source="/home/you/malradar/config.json",target=/etc/malradar/config.json,readonly hekmon/malradar:latest
```

### Binding the state dir

```bash
docker run --mount type=bind,source="/home/you/malradar/config.json",target="/etc/malradar/config.json",readonly --mount type=bind,source="/home/you/malradar/statedir/",target="/var/lib/malradar/" hekmon/malradar:latest
```
