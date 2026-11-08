---
title: An agent command that runs a viewer's local server
---

A viewer developed as a separate project reads Docket projects through a local server, `docket-node`, which its own CLI `docket-node start | stop | status` keeps running as a login service. This optional command in Docket's plugin, `/docket-server start | stop | status`, wraps that CLI the way `docket-init` wraps `dkt init`:

- installs the executable first when it is missing;
- finds which of the machine's addresses to bind to;
- handles the exceptions;
- `start` ends by giving the deep link with which the viewer's desktop app adds the server.

The `docket-node` CLI does not exist yet, so this waits for it.
