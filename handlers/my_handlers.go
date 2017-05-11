package handlers

type RiaHandler struct {
	HostHandler
}

func (hh *RiaHandler) HandleResponse([]byte) (error) {
	return nil
}