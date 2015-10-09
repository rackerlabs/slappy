package stats

import (
	"fmt"
	"io/ioutil"
	"time"

	"github.com/rackerlabs/slappy/config"
	"github.com/rackerlabs/slappy/log"
)

func Status_file() {
	conf := config.Conf()
	logger := log.Logger()

	filepath := conf.Status_file
	if filepath == "" {
		logger.Info("Not writing status files")
		return
	}

	err := ioutil.WriteFile(filepath, []byte(""), 0755)
	if err != nil {
		logger.Error(fmt.Sprintf("ERROR STATUS FILE : %s", err))
	}

	ticker := time.NewTicker(conf.Status_interval)
	for _ = range ticker.C {
		status := get_status()
		err := ioutil.WriteFile(filepath, []byte(status), 0755)
		if err != nil {
			logger.Error(fmt.Sprintf("ERROR STATUS FILE : %s", err))
		}
		logger.Debug(fmt.Sprintf("SUCCESS STATUS FILE : Wrote %s to %s", status, filepath))
	}
}

func get_status() string {
	// Figure out if we're in an error state
	return "0"
}
