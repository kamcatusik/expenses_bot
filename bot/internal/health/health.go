package health

import (
	"os"
	"time"
)

const healthFile = "/tmp/healthy"

func Beat() { _ = os.WriteFile(healthFile, []byte(time.Now().Format(time.RFC3339)), 0o644) }
