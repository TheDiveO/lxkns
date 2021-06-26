# Features

- finds all 8 types of currently defined Linux-kernel
  [namespaces](https://man7.org/linux/man-pages/man7/namespaces.7.html).

- gives namespaces names (sic!).

- discovers mount points in mount namespaces and derives the mount point
  visibility and VFS path hierarchy. The visibility identifies overmounts, which
  can either appear higher up the VFS path hierarchy but also "in place".

- the Go API supports not only discovery, but also switching namespaces (OS
  thread switching).

- tested with Go 1.13-1.16.

- namespace discovery can be integrated into other applications or run as a
  containerized discovery backend service with REST API and web front-end.

- marshal and unmarshal discovery results to and from JSON – this is especially
  useful for separating the super-privileged scanner process from non-privileged
  frontends.

- web front-end of discovery service can be deployed behind path-rewriting
  reverse proxies without any recompilation or image rebuilding when the first
  rewriting reverse proxy adds `X-Forwarded-Uri` HTTP request headers.

- CLI tools for namespace discovery and analysis.
