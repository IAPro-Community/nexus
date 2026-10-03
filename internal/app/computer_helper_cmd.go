package app

import (
	"fmt"
	"os"

	"github.com/IAPro-Community/nexus/internal/control/computer"
)

func computerHelperCmd(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("usage: %s __computer-use-helper", progName())
	}
	return computer.RunNativeHelper(os.Stdin, os.Stdout)
}
