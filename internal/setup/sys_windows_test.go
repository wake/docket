package setup

import "os"

// nonRegularKinds are the kinds of non-regular file TestDecideNonRegularIsUnreadable makes; Windows has no FIFOs.
var nonRegularKinds = []string{"dir"}

func mkfifo(string) error { panic("no FIFOs on Windows") }

// umask is 0: Windows has none.
func umask() os.FileMode { return 0 }
