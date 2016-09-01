# loglet

![loglet](https://s-media-cache-ak0.pinimg.com/236x/e5/b3/50/e5b3508803a66dfbac17ec646b1e1883.jpg "loglet")

loglet is a small program for forwarding journald log entries to kafka.

It has the following main aspirations:

- be a good citizen in a systemd/journald world
- have no external dependencies other than kafka
- be resilient to network outages
- leverage structured log data when available
- integrate well with container management systems like ECS and kubernetes

## Usage

See `loglet --help` for up to date usage instructions.

