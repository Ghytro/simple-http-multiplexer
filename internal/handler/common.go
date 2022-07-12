package handler

import (
	"log"
	"os"
)

func init() {
	log.SetOutput(os.Stderr)
	log.SetFlags(log.Ldate | log.Ltime | log.Llongfile)
}
