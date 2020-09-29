# Hanging Droplets Cleaner for GitLab Runner with authoscale using DigitalOcean

## IMPORTANT!
This a a modified version to address issues with:
```
dropletID is invalid because cannot be less than 1
```

Original version is from https://gitlab.com/tmaczukin/hanging-droplets-cleaner 

## Description
This tool looks for DigitalOcean droplets that were created and 'removed' by
Runner, but were not removed by DigitalOcean. We've found that in some situations
API may respond with `success` response while the droplet was not removed at
all. This ends in wasted resources and extended billings for this wasted power.

This tool:
- lists machines managed by Runner on a host where the tool is running (using
  one or more configured `runner-prefix` to filter machines),
- lists droplets from DigitalOcean (also filters them by `runner-prefix`),
- compares the list and removes any droplet that doesn't have representation
  as Docker Machine on the host.

Additionally it's possible to enable metrics HTTP endpoint that allows monitoring
systems to track how many droplets are being cleaned-up.

## Usage

### The `service` mode

In this mode tool is started as a service that works continuously and executed
a cleanup in configured intervals.

The tool requires few settings that should be provided. Notice that some of settings
can be also provided with environment variables:

| Setting              | Env                  | Required | Default value                    | Description |
|----------------------|----------------------|----------|----------------------------------|-------------|
| `digitalocean-token` | `DIGITALOCEAN_TOKEN` | yes      | -                                | Access token for DigitalOcean API. Needs to have `write` permissions since it's used to remove droplets. |
| `runner-prefix`      | -                    | yes      | -                                | One ore more prefixes for machine name. This is used to filter locally found machines and droplets present at DigitalOcean. |
| `machines-directory` | `MACHINES_DIRECTORY` | no       | `/root/.docker/machine/machines` | Directory where Docker Machine stores configuration of created machines. This is used to list existing machines. **Must be an absolute path!** |
| `interval`           | `INTERVAL`           | no       | `900`                            | Interval between subsequent cleanup attempts. Provided in seconds. |
| `listen`             | `LISTEN`             | no       | -                                | Address on which metrics server is started. If empty, then the feature is disabled. Provided in form of `1.2.3.4:1234` |

**Example**

```bash
$ ./hanging-droplets-cleaner service \
                             --digitalocean-token DO_TOKEN_HERE \
                             --listen 0.0.0.0:9380 \
                             --runner-prefix runner-abc123- \
                             --runner-prefix runner-def456- \
                             --runner-prefix runner-zyx987-
```

### The `one-shot` mode

In this mode tool executes one cleanup attempt and exits. Additionally it doesn't
do a real cleanup by default. This mode can be used if someone wants only to list
the number of droplets that are no more managed by Runner and could be removed.

With additional flag it can also remove droplets.

| Setting              | Env                  | Required | Default value                    | Description |
|----------------------|----------------------|----------|----------------------------------|-------------|
| `digitalocean-token` | `DIGITALOCEAN_TOKEN` | yes      | -                                | Access token for DigitalOcean API. Needs to have `write` permissions since it's used to remove droplets. |
| `runner-prefix`      | -                    | yes      | -                                | One ore more prefixes for machine name. This is used to filter locally found machines and droplets present at DigitalOcean. |
| `machines-directory` | `MACHINES_DIRECTORY` | no       | `/root/.docker/machine/machines` | Directory where Docker Machine stores configuration of created machines. This is used to list existing machines. |
| `delete`             | -                    | no       | `false`                          | If provided the tool will do a real cleanup and remove droplets from DigitalOcean |

**Examples**

```bash
# To only list droplets that could be removed
$ ./hanging-droplets-cleaner one-shot \
                             --digitalocean-token DO_TOKEN_HERE \
                             --listen 0.0.0.0:9380 \
                             --runner-prefix runner-abc123- \
                             --runner-prefix runner-def456- \
                             --runner-prefix runner-zyx987-

# To only list and remove droplets
$ ./hanging-droplets-cleaner one-shot \
                             --digitalocean-token DO_TOKEN_HERE \
                             --listen 0.0.0.0:9380 \
                             --runner-prefix runner-abc123- \
                             --runner-prefix runner-def456- \
                             --runner-prefix runner-zyx987- \
                             --delete
```

### Using Docker container

Prepared Docker image is configured to run the tool in `service` mode. It also starts the
metrics server by default.

When starting this as Docker container we need to add some additional configuration.

First - we need to mount host's machines directory to the container. With a default
configuration the process inside of Docker containers assumes, that the `machines-directory`
is set to `/machines`. In that case we need to use the `-v /path/to/hosts/machines/:/machines`.

If we want to access metrics server from an external monitoring system, then we should
also bind container's port to some host's port, e.g. `-p 9380:9380` which will bind
container's `9380` port to host's `9380` port on all host's interfaces. Notice that
if you want to access this port from an external machine you will probably need also
to update your firewall.

**Example**

```bash
$ docker run -d \
         --restart always \
         --name hanging_droplets_cleaner \
         --log-driver=syslog \
         --log-opt tag=hanging_droplets_cleaner \
         -e DIGITALOCEAN_TOKEN=$DO_TOKEN \
         -e NO_COLOR=true \
         -v /root/.docker/machine/machines/:/machines \
         -p 9380:9380 \
         registry.gitlab.com/tmaczukin/hanging-droplets-cleaner:0.1 \
            --runner-prefix runner-abc123- \
            --runner-prefix runner-def456- \
            --runner-prefix runner-def-
```

## Author

Tomasz Maczukin, 2017, GitLab

## License

MIT
