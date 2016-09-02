package domain

func BroadcastBool(in <-chan bool, outs []chan<- bool) {
	go func() {
		for i := range in {
			for _, out := range outs {
				out <- i
			}
		}
	}()
}
