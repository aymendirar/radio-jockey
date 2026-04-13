# radio 📻 📼 💿

## development

with docker running, run the following commands:

```bash
$ docker compose run --rm codegen # update per-service protos

$ docker compose watch

# look at service logs using any of the following
$ docker compose logs -f # combined all service logs
$ docker compose log -f [server name] # per-service
```
