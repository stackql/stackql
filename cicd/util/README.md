

## Useful hacks

Robot struggles to kill flask servers, so `pgrep` can help:

```bash
pgrep -f flask | xargs kill -9
```