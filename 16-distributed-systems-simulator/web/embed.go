package web

import "embed"

//go:embed index.html app.js styles.css rate-limiting.html rate-limit.js assets/server-box.png assets/user-coral.png assets/load-balancer.png
var Files embed.FS
