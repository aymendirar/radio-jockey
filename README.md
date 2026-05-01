# radio jockey 📻 📼 💿

## development

with docker running, run the following commands:

```bash
$ make codegen # generate service proto files

$ make dev # docker compose watch all services

# look at service logs using any of the following
$ docker compose logs -f # combined all service logs
$ docker compose log -f [server name] # per-service

# run production build
$ make prod
```
