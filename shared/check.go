package shared

import "log"

func Check(err error) {
	if err == nil {
		return
	}
	log.Fatal(err)
}
